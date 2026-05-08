package vouch

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	bootstrap "github.com/duriantaco/vouch/internal/vouch/bootstrap"
)

func TestBootstrapDryRunDraftsConservativeContracts(t *testing.T) {
	repo := bootstrapFixture(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"--repo", repo, "bootstrap", "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("bootstrap --dry-run failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Detected 1 contract drafts:",
		"HIGH auth.password_reset",
		"internal/auth/password_reset.go",
		"test: tests/auth/test_password_reset.py::test_token_expiry",
		"required_test.token_expiry",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected dry-run output to contain %q, got:\n%s", want, out)
		}
	}
	if _, err := LoadJSON[bootstrap.Result](filepath.Join(repo, ".vouch", "build", "bootstrap-report.json")); err == nil {
		t.Fatal("dry-run wrote bootstrap report")
	}
	if fileExists(filepath.Join(repo, ".vouch", "intents", "auth.password_reset.yaml")) {
		t.Fatal("dry-run wrote intent file")
	}
}

func TestBootstrapWritesIntentReportAndCompileCompatibleContract(t *testing.T) {
	repo := bootstrapFixture(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"--repo", repo, "bootstrap"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("bootstrap failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	intentPath := filepath.Join(repo, ".vouch", "intents", "auth.password_reset.yaml")
	reportPath := filepath.Join(repo, ".vouch", "build", "bootstrap-report.json")
	if !fileExists(filepath.Join(repo, ".vouch", "policy", "release-policy.json")) {
		t.Fatal("bootstrap did not initialize the default release policy")
	}
	if !fileExists(intentPath) {
		t.Fatal("bootstrap did not write the generated intent")
	}
	report := mustLoadBootstrapReport(t, reportPath)
	if len(report.Drafts) != 1 {
		t.Fatalf("expected one bootstrap draft, got %#v", report.Drafts)
	}
	obligations := report.Drafts[0].Obligations
	if len(obligations) == 0 {
		t.Fatalf("expected generated obligations, got %#v", report.Drafts[0])
	}
	if obligations[0].Generated.By != "vouch.bootstrap" || obligations[0].Generated.Source.File == "" {
		t.Fatalf("expected structured provenance on obligation, got %#v", obligations[0])
	}

	spec, err := CompileIntentFile(intentPath, filepath.Join(repo, ".vouch", "specs", "auth.password_reset.spec.json"))
	if err != nil {
		t.Fatalf("generated intent did not compile: %v", err)
	}
	if spec.ID != "auth.password_reset" || spec.Risk != RiskHigh || spec.Owner != "security-team" {
		t.Fatalf("compiled unexpected spec metadata: %#v", spec)
	}
	if !contains(spec.OwnedPaths, "internal/auth/password_reset.go") {
		t.Fatalf("expected generated spec to preserve source path, got %#v", spec.OwnedPaths)
	}
	if !contains(spec.Tests.Required, "token expiry") {
		t.Fatalf("expected generated spec to preserve required test, got %#v", spec.Tests.Required)
	}
}

func TestBootstrapCheckFailsUntilGeneratedFilesAreCurrent(t *testing.T) {
	repo := bootstrapFixture(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"--repo", repo, "bootstrap", "--check"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected bootstrap --check to fail before files are written: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Bootstrap check failed") {
		t.Fatalf("expected check failure message, got:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"--repo", repo, "bootstrap"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("bootstrap write failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"--repo", repo, "bootstrap", "--check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected bootstrap --check to pass after write: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "Bootstrap check failed") {
		t.Fatalf("unexpected check failure after write:\n%s", stdout.String())
	}
}

func bootstrapFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeText(t, filepath.Join(repo, "pyproject.toml"), `[tool.pytest.ini_options]
testpaths = ["tests"]
`)
	writeText(t, filepath.Join(repo, "CODEOWNERS"), `/internal/auth/** @security-team
/tests/auth/** @security-team
`)
	writeText(t, filepath.Join(repo, "internal", "auth", "password_reset.go"), `package auth

func ResetPassword() {}
`)
	writeText(t, filepath.Join(repo, "tests", "auth", "test_password_reset.py"), `def test_token_expiry():
    assert True
`)
	writeText(t, filepath.Join(repo, ".github", "workflows", "ci.yml"), `name: CI
`)
	return repo
}

func mustLoadBootstrapReport(t *testing.T, path string) bootstrap.Result {
	t.Helper()
	report, err := LoadJSON[bootstrap.Result](path)
	if err != nil {
		t.Fatal(err)
	}
	return report
}
