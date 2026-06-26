package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestSchemaGolden is the contract-stability gate (contract §10): the full command tree,
// flags, exit codes, and conformance block must match the committed snapshot. A rename/removal
// is a reviewed diff, not a silent break. Regenerate intentionally with:
//
//	UFI_UPDATE_GOLDEN=1 go test ./internal/cli -run TestSchemaGolden
//
// The volatile build "version" field is excluded from the snapshot.
func TestSchemaGolden(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit = %d", code)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("schema not JSON: %v", err)
	}
	delete(got, "version")
	norm, _ := json.MarshalIndent(got, "", "  ")
	norm = append(norm, '\n')

	path := filepath.Join("testdata", "schema.json")
	if os.Getenv("UFI_UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, norm, 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with UFI_UPDATE_GOLDEN=1 to create): %v", err)
	}
	if string(norm) != string(want) {
		t.Fatalf("schema drift vs testdata/schema.json — review the diff; if intended, regenerate with UFI_UPDATE_GOLDEN=1")
	}
}
