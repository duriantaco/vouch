package vouch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ObligationIndex struct {
	ByID map[string]Obligation
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

func LinkEvidenceArtifacts(repo string, artifacts []EvidenceArtifact, index ObligationIndex) ([]ArtifactResult, []InvalidEvidence) {
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
			if obligation.RequiredEvidence != artifact.Kind {
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

		if artifact.Kind == EvidenceTestCoverage {
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
