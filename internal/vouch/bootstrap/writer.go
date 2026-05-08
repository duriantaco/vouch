package bootstrap

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func writeResult(repo string, result Result) ([]string, error) {
	var wrote []string
	for _, draft := range result.Drafts {
		path := filepath.Join(repo, filepath.FromSlash(draft.IntentPath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(renderIntent(draft)), 0o644); err != nil {
			return nil, err
		}
		wrote = append(wrote, draft.IntentPath)
	}
	reportPath := filepath.Join(repo, ".vouch", "build", "bootstrap-report.json")
	if err := writeJSON(reportPath, result); err != nil {
		return nil, err
	}
	wrote = append(wrote, filepath.ToSlash(filepath.Join(".vouch", "build", "bootstrap-report.json")))
	return wrote, nil
}

func needsWrite(repo string, result Result) bool {
	for _, draft := range result.Drafts {
		path := filepath.Join(repo, filepath.FromSlash(draft.IntentPath))
		data, err := os.ReadFile(path)
		if err != nil || string(data) != renderIntent(draft) {
			return true
		}
	}
	return false
}

func renderIntent(draft Draft) string {
	var b strings.Builder
	writeScalar(&b, "feature", draft.Component)
	writeScalar(&b, "owner", draft.Owner)
	writeScalar(&b, "risk", draft.Risk)
	writeScalar(&b, "goal", "Drafted from repository signals; human review required before treating this as product intent.")
	writeList(&b, "owned_paths", draft.Paths, nil)
	writeList(&b, "behavior", behaviorItems(draft), draft.Obligations)
	writeList(&b, "security", securityItems(draft), securityObligations(draft.Obligations))
	writeList(&b, "required_tests", requiredTestItems(draft.Obligations), requiredTestObligations(draft.Obligations))
	writeList(&b, "runtime_metrics", []string{"vouch.gate.decision"}, nil)
	b.WriteString("runtime_alerts: []\n")
	b.WriteString("rollback:\n")
	b.WriteString("  strategy: revert_change\n")
	return b.String()
}

func writeScalar(b *strings.Builder, key string, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(quote(value))
	b.WriteByte('\n')
}

func writeList(b *strings.Builder, key string, values []string, obligations []Obligation) {
	b.WriteString(key)
	b.WriteString(":\n")
	if len(values) == 0 {
		b.WriteString("  []\n")
		return
	}
	for i, value := range values {
		if i < len(obligations) {
			writeGeneratedComment(b, obligations[i])
		}
		b.WriteString("  - ")
		b.WriteString(quote(value))
		b.WriteByte('\n')
	}
}

func writeGeneratedComment(b *strings.Builder, obligation Obligation) {
	b.WriteString("  # generated:\n")
	b.WriteString("  #   by: vouch.bootstrap\n")
	b.WriteString("  #   mode: deterministic\n")
	b.WriteString("  #   confidence: high\n")
	b.WriteString("  #   obligation_id: ")
	b.WriteString(obligation.ID)
	b.WriteByte('\n')
	if obligation.Generated.Source.File != "" {
		b.WriteString("  #   source: ")
		b.WriteString(obligation.Generated.Source.File)
		if obligation.Generated.Source.Symbol != "" {
			b.WriteString("::")
			b.WriteString(obligation.Generated.Source.Symbol)
		}
		b.WriteByte('\n')
	}
}

func behaviorItems(draft Draft) []string {
	var items []string
	for _, obligation := range requiredTestObligations(draft.Obligations) {
		items = append(items, draft.Component+" behavior remains covered by "+obligation.Description)
	}
	if len(items) == 0 {
		items = append(items, draft.Component+" behavior remains covered by accepted evidence")
	}
	return unique(items)
}

func securityItems(draft Draft) []string {
	if draft.Risk == "high" {
		return []string{"security-sensitive changes require explicit evidence"}
	}
	return []string{"no owned-path changes bypass this contract"}
}

func requiredTestItems(obligations []Obligation) []string {
	var items []string
	for _, obligation := range requiredTestObligations(obligations) {
		items = append(items, obligation.Description)
	}
	return items
}

func requiredTestObligations(obligations []Obligation) []Obligation {
	var out []Obligation
	for _, obligation := range obligations {
		if obligation.Kind == "required_test" {
			out = append(out, obligation)
		}
	}
	return out
}

func securityObligations(obligations []Obligation) []Obligation {
	var out []Obligation
	for _, obligation := range obligations {
		if obligation.Kind == "security" {
			out = append(out, obligation)
		}
	}
	return out
}

func quote(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func writeJSON(path string, value any) error {
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data.Bytes(), 0o644)
}
