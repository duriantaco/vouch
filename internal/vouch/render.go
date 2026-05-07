package vouch

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func RenderVerification(evidence Evidence) string {
	if len(evidence.Findings) == 0 {
		return "Evidence verification: pass\n"
	}
	var b strings.Builder
	b.WriteString("Evidence verification:\n")
	for _, finding := range evidence.Findings {
		fmt.Fprintf(&b, "- [%s] %s/%s: %s\n", strings.ToUpper(finding.Decision), finding.Verifier, finding.Severity, finding.Claim)
		fmt.Fprintf(&b, "  evidence: %s\n", finding.Evidence)
		if finding.RequiredFix != "" {
			fmt.Fprintf(&b, "  required fix: %s\n", finding.RequiredFix)
		}
	}
	return b.String()
}

func RenderGate(evidence Evidence) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Release decision: %s\n", evidence.Decision)
	for _, reason := range evidence.Reasons {
		fmt.Fprintf(&b, "- %s\n", reason)
	}
	return b.String()
}

func GateResultFromEvidence(evidence Evidence) GateResult {
	return GateResult{
		Version:            "vouch.gate_result.v0",
		Decision:           evidence.Decision,
		Reasons:            cloneStrings(evidence.Reasons),
		PolicyPath:         evidence.PolicyPath,
		Compilation:        evidence.Compilation,
		SpecErrors:         cloneStrings(evidence.SpecErrors),
		ManifestErrors:     cloneStrings(evidence.ManifestErrors),
		ArtifactResults:    cloneArtifactResults(evidence.ArtifactResults),
		InvalidEvidence:    cloneInvalidEvidence(evidence.InvalidEvidence),
		MissingObligations: obligationTextMap(evidence.MissingObligations),
		CoveredObligations: obligationTextMap(evidence.CoveredObligations),
		PolicyRulesFired:   cloneStrings(evidence.PolicyResult.RulesFired),
		FiredPolicyRule:    evidence.PolicyResult.FiredPolicyRule,
	}
}

func cloneStrings(values []string) []string {
	out := make([]string, 0, len(values))
	return append(out, values...)
}

func cloneArtifactResults(values []ArtifactResult) []ArtifactResult {
	out := make([]ArtifactResult, 0, len(values))
	return append(out, values...)
}

func cloneInvalidEvidence(values []InvalidEvidence) []InvalidEvidence {
	out := make([]InvalidEvidence, 0, len(values))
	return append(out, values...)
}

func RenderGateResultJSON(evidence Evidence) (string, error) {
	data, err := json.MarshalIndent(GateResultFromEvidence(evidence), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func RenderEvidence(evidence Evidence) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Vouch Evidence Report")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Repo: %s\n", evidence.Repo)
	fmt.Fprintf(&b, "Manifest: %s\n", evidence.ManifestPath)
	fmt.Fprintf(&b, "Task: %s - %s\n", evidence.Manifest.Task.ID, evidence.Manifest.Task.Summary)
	fmt.Fprintf(&b, "Risk: %s\n", evidence.Manifest.Change.Risk)
	fmt.Fprintf(&b, "Decision: %s\n", evidence.Decision)
	fmt.Fprintf(&b, "Compiled specs: %d/%d (%d skipped), obligations: %d\n", evidence.Compilation.SpecsCompiled, evidence.Compilation.SpecsLoaded, evidence.Compilation.SpecsSkipped, evidence.Compilation.ObligationsBuilt)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Spec Coverage")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "IR obligations covered: %s\n", obligationCoverageLine(evidence))
	fmt.Fprintf(&b, "Required tests covered: %s\n", coverageLine(evidence.RequiredTests, evidence.CoveredTests))
	fmt.Fprintf(&b, "Security invariants covered: %s\n", coverageLine(evidence.RequiredSecurity, evidence.CoveredSecurity))
	for _, specID := range sortedRenderKeys(evidence.MissingObligations) {
		missing := evidence.MissingObligations[specID]
		for _, obligation := range missing {
			if obligation.Kind == ObligationRequiredTest || obligation.Kind == ObligationSecurity {
				continue
			}
			fmt.Fprintf(&b, "- Missing %s obligation for %s: %s\n", obligation.Kind, specID, obligation.Text)
		}
	}
	for _, specID := range sortedRenderKeys(evidence.MissingTests) {
		missing := evidence.MissingTests[specID]
		if len(missing) > 0 {
			fmt.Fprintf(&b, "- Missing tests for %s: %s\n", specID, strings.Join(missing, ", "))
		}
	}
	for _, specID := range sortedRenderKeys(evidence.MissingSecurity) {
		missing := evidence.MissingSecurity[specID]
		if len(missing) > 0 {
			fmt.Fprintf(&b, "- Missing security checks for %s: %s\n", specID, strings.Join(missing, ", "))
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Evidence Verification")
	fmt.Fprintln(&b)
	if len(evidence.Findings) == 0 {
		fmt.Fprintln(&b, "- pass")
	} else {
		for _, finding := range evidence.Findings {
			fmt.Fprintf(&b, "- %s %s/%s: %s\n", strings.ToUpper(finding.Decision), finding.Verifier, finding.Severity, finding.Claim)
			fmt.Fprintf(&b, "  Evidence: %s\n", finding.Evidence)
			if finding.RequiredFix != "" {
				fmt.Fprintf(&b, "  Required fix: %s\n", finding.RequiredFix)
			}
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Release Gate")
	fmt.Fprintln(&b)
	if len(evidence.Reasons) == 0 {
		fmt.Fprintln(&b, "- no release-gate reasons recorded")
	} else {
		for _, reason := range evidence.Reasons {
			fmt.Fprintf(&b, "- %s\n", reason)
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Runtime And Rollback")
	fmt.Fprintln(&b)
	if len(evidence.Manifest.Runtime.Metrics) > 0 {
		fmt.Fprintf(&b, "Watch: %s\n", strings.Join(evidence.Manifest.Runtime.Metrics, ", "))
	}
	if evidence.Manifest.Runtime.Canary.Enabled {
		fmt.Fprintf(&b, "Canary: enabled at %d percent\n", evidence.Manifest.Runtime.Canary.InitialPercent)
	}
	if evidence.Manifest.Rollback.Strategy != "" {
		fmt.Fprintf(&b, "Rollback: %s\n", evidence.Manifest.Rollback.Strategy)
	}
	if evidence.Manifest.Rollback.Flag != "" {
		fmt.Fprintf(&b, "Flag: %s\n", evidence.Manifest.Rollback.Flag)
	}
	if len(evidence.Manifest.Uncertainties) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "## Uncertainties")
		fmt.Fprintln(&b)
		for _, item := range evidence.Manifest.Uncertainties {
			fmt.Fprintf(&b, "- %s\n", item)
		}
	}
	return b.String()
}

func RenderJSON(evidence Evidence) (string, error) {
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func coverageLine(required map[string][]string, covered map[string][]string) string {
	total := 0
	seen := 0
	for _, specID := range sortedRenderKeys(required) {
		items := required[specID]
		total += len(items)
		seen += len(covered[specID])
	}
	return fmt.Sprintf("%d/%d", seen, total)
}

func obligationCoverageLine(evidence Evidence) string {
	total := 0
	seen := 0
	for _, specID := range sortedRenderKeys(evidence.RequiredObligations) {
		obligations := evidence.RequiredObligations[specID]
		total += len(obligations)
		seen += len(evidence.CoveredObligations[specID])
	}
	return fmt.Sprintf("%d/%d", seen, total)
}

func sortedRenderKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func obligationTextMap(items map[string][]Obligation) map[string][]string {
	out := make(map[string][]string, len(items))
	for _, specID := range sortedRenderKeys(items) {
		for _, obligation := range items[specID] {
			out[specID] = append(out[specID], obligation.ID)
		}
	}
	return out
}
