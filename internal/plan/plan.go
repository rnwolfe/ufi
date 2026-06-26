// Package plan persists declarative-config change plans for the reviewed-artifact apply flow
// (contract §2): a `--dry-run` preview writes a plan keyed by a content hash, and
// `ufi apply <hash>` executes exactly that persisted request. Plans live under
// $XDG_STATE_HOME/ufi/plans/<hash>.json.
package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Plan is a single previewed, executable config change.
type Plan struct {
	Hash      string          `json:"hash"`
	Op        string          `json:"op"`
	Method    string          `json:"method"`
	Path      string          `json:"path"`
	Body      json.RawMessage `json:"body,omitempty"`
	Summary   map[string]any  `json:"summary,omitempty"`
	CreatedAt string          `json:"created_at,omitempty"`
}

// Hash is a stable 12-hex digest over the executable parts of the change (op|method|path|body),
// so re-previewing the same change yields the same hash.
func Hash(op, method, path string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(op + "\n" + method + "\n" + path + "\n"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// New builds a Plan and stamps its hash + creation time.
func New(op, method, path string, body []byte, summary map[string]any) Plan {
	return Plan{
		Hash:      Hash(op, method, path, body),
		Op:        op,
		Method:    method,
		Path:      path,
		Body:      json.RawMessage(body),
		Summary:   summary,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// Save writes the plan to the state dir (0600).
func Save(p Plan) error {
	dir, err := plansDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, p.Hash+".json"), b, 0o600)
}

// Load returns the plan for a hash, or ok=false if none is persisted.
func Load(hash string) (Plan, bool, error) {
	dir, err := plansDir()
	if err != nil {
		return Plan{}, false, err
	}
	b, err := os.ReadFile(filepath.Join(dir, hash+".json"))
	if os.IsNotExist(err) {
		return Plan{}, false, nil
	}
	if err != nil {
		return Plan{}, false, err
	}
	var p Plan
	if err := json.Unmarshal(b, &p); err != nil {
		return Plan{}, false, err
	}
	return p, true, nil
}

func plansDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "ufi", "plans"), nil
}
