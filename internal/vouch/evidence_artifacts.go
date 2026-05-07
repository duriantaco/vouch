package vouch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ObligationIndex struct {
	ByID map[string]Obligation
}

type ArtifactLinkOptions struct {
	RequireSigned bool
}

func NewObligationIndex(plans map[string]VerificationPlan) ObligationIndex {
	index := ObligationIndex{ByID: make(map[string]Obligation)}
	for _, plan := range plans {
		for _, obligation := range plan.Obligations {
			index.ByID[obligation.ID] = obligation
		}
	}
	return index
}

func LinkEvidenceArtifacts(repo string, artifacts []EvidenceArtifact, index ObligationIndex, opts ArtifactLinkOptions) ([]ArtifactResult, []InvalidEvidence) {
	results := make([]ArtifactResult, 0, len(artifacts))
	var invalid []InvalidEvidence
	for _, artifact := range artifacts {
		result := ArtifactResult{
			ID:     artifact.ID,
			Kind:   artifact.Kind,
			Path:   artifact.Path,
			Status: "valid",
		}

		if artifact.ExitCode == nil {
			result.addIssue("missing_exit_code", "artifact exit_code is required")
		} else if *artifact.ExitCode != 0 {
			result.addIssue("non_zero_exit", fmt.Sprintf("artifact command exited with code %d", *artifact.ExitCode))
		}

		for _, obligationID := range artifact.Obligations {
			obligation, ok := index.ByID[obligationID]
			if !ok {
				result.addIssue("unknown_obligation", fmt.Sprintf("unknown obligation %q", obligationID))
				continue
			}
			if artifact.Kind != EvidenceVerifierOutput && obligation.RequiredEvidence != artifact.Kind {
				result.addIssue("kind_mismatch", fmt.Sprintf("kind %q does not satisfy obligation %s required evidence %q", artifact.Kind, obligationID, obligation.RequiredEvidence))
			}
		}

		var data []byte
		if artifact.Path != "" {
			resolved, err := resolveArtifactPath(repo, artifact.Path)
			result.ResolvedPath = resolved
			if err != nil {
				result.addIssue("artifact_path_escape", err.Error())
			} else {
				bytes, err := os.ReadFile(resolved)
				if err != nil {
					result.addIssue("artifact_missing", fmt.Sprintf("cannot read artifact path %s: %v", artifact.Path, err))
				} else {
					data = bytes
					if artifact.SHA256 != "" {
						actual := sha256Hex(data)
						if !strings.EqualFold(actual, artifact.SHA256) {
							result.addIssue("sha256_mismatch", fmt.Sprintf("artifact %s sha256 mismatch: expected %s got %s", artifact.ID, artifact.SHA256, actual))
						} else {
							result.HashVerified = true
						}
					}
				}
			}
		} else {
			result.addIssue("artifact_path_required", fmt.Sprintf("%s evidence requires an artifact path", artifact.Kind))
		}

		if opts.RequireSigned && len(data) > 0 {
			verifyCosignBundle(repo, artifact, result.ResolvedPath, &result)
		}

		if artifact.Kind == EvidenceVerifierOutput {
			if len(data) > 0 {
				output, issues := importVerifierOutput(data, artifact.Obligations, index)
				if len(issues) == 0 {
					result.VerifierOutput = &output
					result.VerifierFindings = cloneFindings(output.Findings)
					result.CoveredObligations = cloneStrings(output.Obligations)
				}
				for _, issue := range issues {
					result.addIssue("verifier_output_import", issue)
				}
			}
		} else if artifact.Kind == EvidenceTestCoverage {
			if len(data) > 0 {
				covered, failed, issues := importJUnitEvidence(data, artifact.Obligations)
				result.CoveredObligations = covered
				result.FailedTests = failed
				for _, issue := range issues {
					result.addIssue("junit_import", issue)
				}
			}
		} else if len(data) > 0 {
			covered, issues := importGenericEvidence(data, artifact.Obligations)
			result.CoveredObligations = covered
			for _, issue := range issues {
				result.addIssue("artifact_import", issue)
			}
		}

		if len(result.Issues) > 0 {
			result.Status = "invalid"
			for _, issue := range result.Issues {
				code, message, _ := strings.Cut(issue, ": ")
				if message == "" {
					message = issue
				}
				invalid = append(invalid, InvalidEvidence{
					Artifact: artifact.ID,
					Code:     code,
					Message:  message,
				})
			}
		}
		results = append(results, result)
	}
	return results, invalid
}

func verifyCosignBundle(repo string, artifact EvidenceArtifact, artifactPath string, result *ArtifactResult) {
	if artifact.SignatureBundle == "" {
		result.addIssue("missing_signature_bundle", "signature_bundle is required when signed evidence is enforced")
		return
	}
	if artifact.SignerIdentity == "" {
		result.addIssue("missing_signer_identity", "signer_identity is required when signed evidence is enforced")
		return
	}
	if artifact.SignerOIDCIssuer == "" {
		result.addIssue("missing_signer_oidc_issuer", "signer_oidc_issuer is required when signed evidence is enforced")
		return
	}
	bundlePath, err := resolveArtifactPath(repo, artifact.SignatureBundle)
	if err != nil {
		result.addIssue("signature_bundle_path_escape", err.Error())
		return
	}
	if _, err := os.Stat(bundlePath); err != nil {
		result.addIssue("signature_bundle_missing", fmt.Sprintf("cannot read signature bundle %s: %v", artifact.SignatureBundle, err))
		return
	}
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		result.addIssue("cosign_missing", "cosign is required to verify signed evidence")
		return
	}
	cmd := exec.Command(cosignPath,
		"verify-blob",
		artifactPath,
		"--bundle", bundlePath,
		"--certificate-identity="+artifact.SignerIdentity,
		"--certificate-oidc-issuer="+artifact.SignerOIDCIssuer,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		result.addIssue("signature_verify", message)
		return
	}
	result.SignatureVerified = true
}

func (r *ArtifactResult) addIssue(code string, message string) {
	r.Issues = append(r.Issues, code+": "+message)
}

func resolveArtifactPath(repo string, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, fmt.Errorf("absolute artifact paths are not allowed: %s", path)
	}
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return filepath.Join(repo, path), err
	}
	resolved, err := filepath.Abs(filepath.Join(absRepo, path))
	if err != nil {
		return filepath.Join(absRepo, path), err
	}
	if !pathWithin(absRepo, resolved) {
		return resolved, fmt.Errorf("artifact path escapes repo: %s", path)
	}
	if _, err := os.Stat(resolved); err == nil {
		evalRepo, repoErr := filepath.EvalSymlinks(absRepo)
		evalResolved, resolvedErr := filepath.EvalSymlinks(resolved)
		if repoErr == nil && resolvedErr == nil && !pathWithin(evalRepo, evalResolved) {
			return resolved, fmt.Errorf("artifact symlink escapes repo: %s", path)
		}
	}
	return resolved, nil
}

func pathWithin(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type junitTestSuites struct {
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Skipped  int              `xml:"skipped,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
	Cases    []junitTestCase  `xml:"testcase"`
}

type junitTestSuite struct {
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Skipped  int              `xml:"skipped,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
	Cases    []junitTestCase  `xml:"testcase"`
}

type junitTestCase struct {
	Classname string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	File      string        `xml:"file,attr"`
	Failure   *junitProblem `xml:"failure"`
	Error     *junitProblem `xml:"error"`
	Skipped   *junitProblem `xml:"skipped"`
}

type junitProblem struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func importJUnitEvidence(data []byte, obligationIDs []string) ([]string, []string, []string) {
	var root junitTestSuites
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, nil, []string{fmt.Sprintf("cannot parse JUnit XML: %v", err)}
	}

	cases := collectJUnitCases(root)
	if len(cases) == 0 {
		return nil, nil, []string{"JUnit XML contains no testcase elements"}
	}

	obligations := stringSet(obligationIDs)
	coveredSet := make(map[string]bool)
	var failed []string
	failures, errors, skipped := junitProblemCounts(root)
	for _, testCase := range cases {
		if testCase.Failure != nil || testCase.Error != nil || testCase.Skipped != nil {
			failed = append(failed, testCase.Label())
			continue
		}
		matched := matchedObligations(testCase, obligations)
		for _, obligationID := range matched {
			coveredSet[obligationID] = true
		}
	}

	var covered []string
	var issues []string
	if failures > 0 || errors > 0 || skipped > 0 {
		issues = append(issues, fmt.Sprintf("JUnit suite reports failures=%d errors=%d skipped=%d", failures, errors, skipped))
	}
	for _, obligationID := range obligationIDs {
		if coveredSet[obligationID] {
			covered = append(covered, obligationID)
		} else {
			issues = append(issues, fmt.Sprintf("no passing JUnit testcase references obligation %s", obligationID))
		}
	}
	if len(failed) > 0 {
		issues = append(issues, fmt.Sprintf("JUnit has failing/error/skipped testcases: %s", strings.Join(failed, ", ")))
	}
	return covered, failed, issues
}

func importGenericEvidence(data []byte, obligationIDs []string) ([]string, []string) {
	tokens := evidenceTokens(data)
	var covered []string
	var issues []string
	if issue, ok := artifactStatusIssue(data); ok {
		issues = append(issues, issue)
	}
	for _, obligationID := range obligationIDs {
		if tokens[obligationID] {
			covered = append(covered, obligationID)
		} else {
			issues = append(issues, fmt.Sprintf("artifact content does not reference obligation %s", obligationID))
		}
	}
	return covered, issues
}

func importVerifierOutput(data []byte, artifactObligations []string, index ObligationIndex) (VerifierOutput, []string) {
	var output VerifierOutput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return output, []string{fmt.Sprintf("cannot parse verifier output JSON: %v", err)}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return output, []string{"trailing JSON content after verifier output"}
	}

	var issues []string
	if output.Version != VerifierOutputVersion {
		issues = append(issues, fmt.Sprintf("version must be %s", VerifierOutputVersion))
	}
	if strings.TrimSpace(output.Verifier) == "" {
		issues = append(issues, "verifier is required")
	}
	if output.PromptVersion != VerifierPromptVersion {
		issues = append(issues, fmt.Sprintf("prompt_version must be %s", VerifierPromptVersion))
	}
	if strings.TrimSpace(output.Model) == "" {
		issues = append(issues, "model is required")
	}
	if len(output.Obligations) == 0 {
		issues = append(issues, "obligations must reference at least one obligation")
	}
	if output.Confidence < 0 || output.Confidence > 1 {
		issues = append(issues, "confidence must be between 0 and 1")
	}
	if duplicates := duplicateStrings(output.Obligations); len(duplicates) > 0 {
		issues = append(issues, "obligations contain duplicate values: "+strings.Join(duplicates, ", "))
	}
	if !sameStringSet(output.Obligations, artifactObligations) {
		issues = append(issues, "output obligations must match manifest artifact obligations")
	}
	for _, obligationID := range output.Obligations {
		if strings.TrimSpace(obligationID) == "" {
			issues = append(issues, "obligations must be non-empty")
			continue
		}
		if _, ok := index.ByID[obligationID]; !ok {
			issues = append(issues, fmt.Sprintf("unknown obligation %q", obligationID))
		}
	}
	outputObligations := stringSet(output.Obligations)
	for i, finding := range output.Findings {
		issues = append(issues, validateVerifierFinding(output, finding, i, outputObligations, index)...)
	}
	return output, issues
}

func validateVerifierFinding(output VerifierOutput, finding Finding, index int, outputObligations map[string]bool, obligations ObligationIndex) []string {
	owner := fmt.Sprintf("findings[%d]", index)
	var issues []string
	if strings.TrimSpace(finding.Verifier) == "" {
		issues = append(issues, owner+" verifier is required")
	} else if finding.Verifier != output.Verifier {
		issues = append(issues, fmt.Sprintf("%s verifier %q does not match output verifier %q", owner, finding.Verifier, output.Verifier))
	}
	if !validVerifierDecision(finding.Decision) {
		issues = append(issues, fmt.Sprintf("%s decision must be pass or block", owner))
	}
	if !validFindingSeverity(finding.Severity) {
		issues = append(issues, fmt.Sprintf("%s severity must be low, medium, high, or critical", owner))
	}
	if strings.TrimSpace(finding.Claim) == "" {
		issues = append(issues, owner+" claim is required")
	}
	if strings.TrimSpace(finding.Evidence) == "" {
		issues = append(issues, owner+" evidence is required")
	}
	if finding.Decision == "block" && strings.TrimSpace(finding.RequiredFix) == "" {
		issues = append(issues, owner+" required_fix is required for block decisions")
	}
	if len(finding.Obligations) == 0 {
		issues = append(issues, owner+" obligations must reference at least one obligation")
	}
	if duplicates := duplicateStrings(finding.Obligations); len(duplicates) > 0 {
		issues = append(issues, owner+" obligations contain duplicate values: "+strings.Join(duplicates, ", "))
	}
	for _, obligationID := range finding.Obligations {
		if strings.TrimSpace(obligationID) == "" {
			issues = append(issues, owner+" obligations must be non-empty")
			continue
		}
		if !outputObligations[obligationID] {
			issues = append(issues, fmt.Sprintf("%s obligation %q is not listed in output obligations", owner, obligationID))
		}
		if _, ok := obligations.ByID[obligationID]; !ok {
			issues = append(issues, fmt.Sprintf("%s references unknown obligation %q", owner, obligationID))
		}
	}
	return issues
}

func validVerifierDecision(decision string) bool {
	switch decision {
	case "pass", "block":
		return true
	default:
		return false
	}
}

func validFindingSeverity(severity string) bool {
	switch severity {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	set := stringSet(left)
	if len(set) != len(left) {
		return false
	}
	for _, value := range right {
		if !set[value] {
			return false
		}
	}
	return true
}

func duplicateStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	duplicates := []string{}
	for _, value := range values {
		if seen[value] && !containsString(duplicates, value) {
			duplicates = append(duplicates, value)
		}
		seen[value] = true
	}
	return duplicates
}

func artifactStatusIssue(data []byte) (string, bool) {
	status, ok := artifactStatus(data)
	if !ok {
		return "", false
	}
	if passingArtifactStatus(status) {
		return "", false
	}
	return fmt.Sprintf("artifact status %q is not a passing status", status), true
}

func artifactStatus(data []byte) (string, bool) {
	var value any
	if err := json.Unmarshal(data, &value); err == nil {
		if object, ok := value.(map[string]any); ok {
			if status, ok := object["status"]; ok {
				return fmt.Sprint(status), true
			}
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.TrimSpace(key) == "status" {
			status := strings.Trim(strings.TrimSpace(value), `"'`)
			return status, true
		}
	}
	return "", false
}

func passingArtifactStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass", "passed", "success", "succeeded", "ok":
		return true
	default:
		return false
	}
}

func evidenceTokens(data []byte) map[string]bool {
	tokens := make(map[string]bool)
	var value any
	if err := json.Unmarshal(data, &value); err == nil {
		collectJSONTokens(value, tokens)
		return tokens
	}
	for _, line := range strings.Split(string(data), "\n") {
		token := strings.TrimSpace(line)
		token = strings.TrimPrefix(token, "- ")
		if _, value, ok := strings.Cut(token, ":"); ok {
			token = strings.TrimSpace(value)
		}
		token = strings.Trim(token, `"',[]`)
		if token != "" {
			tokens[token] = true
		}
	}
	return tokens
}

func collectJSONTokens(value any, tokens map[string]bool) {
	switch typed := value.(type) {
	case string:
		tokens[typed] = true
	case []any:
		for _, item := range typed {
			collectJSONTokens(item, tokens)
		}
	case map[string]any:
		for _, item := range typed {
			collectJSONTokens(item, tokens)
		}
	}
}

func collectJUnitCases(root junitTestSuites) []junitTestCase {
	cases := append([]junitTestCase(nil), root.Cases...)
	for _, suite := range root.Suites {
		cases = append(cases, collectJUnitSuiteCases(suite)...)
	}
	return cases
}

func collectJUnitSuiteCases(suite junitTestSuite) []junitTestCase {
	cases := append([]junitTestCase(nil), suite.Cases...)
	for _, child := range suite.Suites {
		cases = append(cases, collectJUnitSuiteCases(child)...)
	}
	return cases
}

func matchedObligations(testCase junitTestCase, obligations map[string]bool) []string {
	var matched []string
	for obligationID := range obligations {
		if testCase.Classname == obligationID || testCase.Name == obligationID {
			matched = append(matched, obligationID)
		}
	}
	return matched
}

func junitProblemCounts(root junitTestSuites) (int, int, int) {
	failures := root.Failures
	errors := root.Errors
	skipped := root.Skipped
	for _, suite := range root.Suites {
		childFailures, childErrors, childSkipped := junitSuiteProblemCounts(suite)
		failures += childFailures
		errors += childErrors
		skipped += childSkipped
	}
	return failures, errors, skipped
}

func junitSuiteProblemCounts(suite junitTestSuite) (int, int, int) {
	failures := suite.Failures
	errors := suite.Errors
	skipped := suite.Skipped
	for _, child := range suite.Suites {
		childFailures, childErrors, childSkipped := junitSuiteProblemCounts(child)
		failures += childFailures
		errors += childErrors
		skipped += childSkipped
	}
	return failures, errors, skipped
}

func (tc junitTestCase) Label() string {
	if tc.Classname == "" {
		return tc.Name
	}
	if tc.Name == "" {
		return tc.Classname
	}
	return tc.Classname + "." + tc.Name
}
