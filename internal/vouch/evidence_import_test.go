package vouch

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceImportJUnitAndManifestlessGate(t *testing.T) {
	repo := bootstrapFixture(t)
	writeText(t, filepath.Join(repo, ".vouch", "artifacts", "pytest.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="pytest" tests="1" failures="0" errors="0" skipped="0">
  <testcase classname="tests.auth.test_password_reset" name="test_token_expiry" file="tests/auth/test_password_reset.py"></testcase>
</testsuite>
`)
	var stdout, stderr bytes.Buffer

	if code := Main([]string{"--repo", repo, "bootstrap"}, &stdout, &stderr); code != 0 {
		t.Fatalf("bootstrap failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"--repo", repo, "compile"}, &stdout, &stderr); code != 0 {
		t.Fatalf("compile failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"--repo", repo, "evidence", "import", "junit", ".vouch/artifacts/pytest.xml"}, &stdout, &stderr); code != 0 {
		t.Fatalf("evidence import failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Linked obligations: 1") {
		t.Fatalf("unexpected import output: %s", stdout.String())
	}
	manifest := mustLoadEvidenceManifest(t, filepath.Join(repo, ".vouch", "evidence", "manifest.json"))
	if len(manifest.Links) != 1 {
		t.Fatalf("expected one evidence link, got %#v", manifest.Links)
	}
	link := manifest.Links[0]
	if link.ObligationID != "auth.password_reset.required_test.token_expiry" || link.Status != "passed" {
		t.Fatalf("unexpected evidence link: %#v", link)
	}
	if link.Testcase != "tests/auth/test_password_reset.py::test_token_expiry" {
		t.Fatalf("expected source-file testcase link, got %#v", link)
	}

	stdout.Reset()
	stderr.Reset()
	code := Main([]string{"--repo", repo, "gate"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected gate to block missing non-JUnit evidence: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Release decision: block",
		"Component:",
		"auth.password_reset",
		"Covered:",
		"required_test.token_expiry",
		"Missing:",
		"security.security_sensitive_changes_require_explicit_evidence",
		"accepted evidence: security_check",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected gate output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestEvidenceImportJUnitRejectsBeforeCompile(t *testing.T) {
	repo := bootstrapFixture(t)
	writeText(t, filepath.Join(repo, ".vouch", "artifacts", "pytest.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="pytest" tests="1">
  <testcase classname="tests.auth.test_password_reset" name="test_token_expiry" file="tests/auth/test_password_reset.py"></testcase>
</testsuite>
`)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"--repo", repo, "evidence", "import", "junit", ".vouch/artifacts/pytest.xml"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected import to fail before compile: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "run vouch compile first") {
		t.Fatalf("expected compile-first error, got stderr:\n%s", stderr.String())
	}
}

func mustLoadEvidenceManifest(t *testing.T, path string) EvidenceManifest {
	t.Helper()
	manifest, err := LoadJSON[EvidenceManifest](path)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}
