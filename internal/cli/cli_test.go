package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(""), &out, &errb)
	return out.String(), errb.String(), code
}

func useTempStore(t *testing.T) {
	t.Helper()
	t.Setenv("UFI_STORE", filepath.Join(t.TempDir(), "devices.json"))
	t.Setenv("UNIFI_API_KEY", "")
	t.Setenv("NO_COLOR", "1")
}

func TestMutationBlockedByDefault(t *testing.T) {
	useTempStore(t)
	out, errb, code := run(t, "device", "restart", "d1", "--json")
	if code != 12 {
		t.Fatalf("exit = %d, want 12 (MUTATION_BLOCKED)", code)
	}
	if !strings.Contains(errb, "MUTATION_BLOCKED") {
		t.Fatalf("stderr missing MUTATION_BLOCKED: %s", errb)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("stdout should be empty on error, got: %s", out)
	}
}

// The gate opens with --allow-mutations; --dry-run previews the action and changes nothing.
func TestMutationGateAllowsDryRun(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "device", "restart", "d1", "--allow-mutations", "--dry-run", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "dry_run") || !strings.Contains(out, "RESTART") {
		t.Fatalf("dry-run output missing marker/action: %s", out)
	}
}

// Config writes are blocked by default and, when allowed, emit a reviewed-artifact plan + hash.
func TestConfigPreviewBlockedByDefault(t *testing.T) {
	useTempStore(t)
	_, errb, code := run(t, "network", "create", "--data", "{}", "--json")
	if code != 12 {
		t.Fatalf("exit = %d, want 12 (MUTATION_BLOCKED)", code)
	}
	if !strings.Contains(errb, "MUTATION_BLOCKED") {
		t.Fatalf("stderr missing MUTATION_BLOCKED: %s", errb)
	}
}

func TestConfigPreviewEmitsHash(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "network", "create", "--data", "{}", "--allow-mutations", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if h, _ := m["hash"].(string); h == "" {
		t.Fatalf("config preview missing plan hash: %s", out)
	}
}

func TestApplyUnknownHash(t *testing.T) {
	useTempStore(t)
	_, errb, code := run(t, "apply", "deadbeef", "--allow-mutations", "--json")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage / PLAN_NOT_FOUND)", code)
	}
	if !strings.Contains(errb, "PLAN_NOT_FOUND") {
		t.Fatalf("stderr missing PLAN_NOT_FOUND: %s", errb)
	}
}

// --write is an accepted alias for --allow-mutations.
func TestWriteAliasOpensGate(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "device", "restart", "d1", "--write", "--dry-run", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (--write should open the gate)", code)
	}
	if !strings.Contains(out, "RESTART") {
		t.Fatalf("unexpected: %s", out)
	}
}

// UFI_HELP=agent makes a help request print the embedded SKILL contract.
func TestAgentHelpMode(t *testing.T) {
	t.Setenv("UFI_HELP", "agent")
	out, _, code := run(t, "--help")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "name: ufi") {
		t.Fatalf("UFI_HELP=agent should print the embedded SKILL.md, got: %.80s", out)
	}
}

func TestSchemaHasSafetyAndExitCodes(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit = %d, want 0", code)
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("schema not valid JSON: %v", err)
	}
	if _, ok := s["safety"]; !ok {
		t.Fatalf("schema missing safety state")
	}
	codes, ok := s["exit_codes"].(map[string]any)
	if !ok {
		t.Fatalf("schema missing exit_codes")
	}
	if _, ok := codes["unsupported"]; !ok {
		t.Fatalf("exit_codes missing the ufi-specific 'unsupported' code")
	}
	if !strings.Contains(out, "device") {
		t.Fatalf("schema missing the device command surface")
	}
}

func TestDidYouMean(t *testing.T) {
	useTempStore(t)
	_, errb, code := run(t, "devce", "list")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage)", code)
	}
	if !strings.Contains(errb, "did you mean") || !strings.Contains(errb, "device") {
		t.Fatalf("missing suggestion: %s", errb)
	}
}

func TestVersionCheck(t *testing.T) {
	useTempStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v999.0.0"}`))
	}))
	defer srv.Close()
	t.Setenv("UFI_RELEASES_URL", srv.URL)

	out, _, code := run(t, "version", "--check", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", err, out)
	}
	if m["current"] == nil {
		t.Fatalf("missing current: %v", m)
	}
	if m["latest"] != "v999.0.0" {
		t.Fatalf("latest = %v, want v999.0.0", m["latest"])
	}
	if _, ok := m["upgrade"]; !ok {
		t.Fatalf("missing upgrade hint: %v", m)
	}
}

func TestVersionCheckFailSilent(t *testing.T) {
	useTempStore(t)
	t.Setenv("UFI_RELEASES_URL", "http://127.0.0.1:0") // unreachable → fail-silent
	out, _, code := run(t, "version", "--check", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (fail-silent), got %d", code, code)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", err, out)
	}
	if m["updateAvailable"] != false {
		t.Fatalf("updateAvailable = %v, want false on failure", m["updateAvailable"])
	}
}
