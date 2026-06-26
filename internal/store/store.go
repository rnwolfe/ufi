// Package store is the placeholder target: a local JSON-backed device store. The real tool
// replaces this package with the UniFi API client (cli-implement wires X-API-KEY auth + the
// Integration/Site Manager APIs). It exists so the skeleton compiles, runs, and is fully
// testable offline; the schema deliberately mirrors a trimmed UniFi device shape.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Device is a trimmed placeholder shape (real fields are mapped from the UniFi wire by
// cli-implement; see spec.md "Wire vs. output naming").
type Device struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"`
	State string `json:"state"`
}

type Store struct{ path string }

func New(path string) *Store { return &Store{path: path} }

// DefaultPath resolves the store location (XDG-aware), overridable via UFI_STORE.
func DefaultPath() string {
	if p := os.Getenv("UFI_STORE"); p != "" {
		return p
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "ufi", "devices.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "ufi", "devices.json")
}

func (s *Store) load() ([]Device, error) {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []Device{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []Device
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) save(items []Device) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *Store) List() ([]Device, error) { return s.load() }

func (s *Store) Get(id string) (Device, bool, error) {
	items, err := s.load()
	if err != nil {
		return Device{}, false, err
	}
	for _, it := range items {
		if it.ID == id {
			return it, true, nil
		}
	}
	return Device{}, false, nil
}

// Add appends a device with a deterministic next-integer id (placeholder seam for tests).
func (s *Store) Add(name string) (Device, error) {
	items, err := s.load()
	if err != nil {
		return Device{}, err
	}
	max := 0
	for _, it := range items {
		if n, err := strconv.Atoi(it.ID); err == nil && n > max {
			max = n
		}
	}
	it := Device{ID: strconv.Itoa(max + 1), Name: name}
	return it, s.save(append(items, it))
}

// Delete removes a device by id. Idempotent: removing a missing id is not an error.
// The bool reports whether the device existed.
func (s *Store) Delete(id string) (bool, error) {
	items, err := s.load()
	if err != nil {
		return false, err
	}
	out := items[:0:0]
	existed := false
	for _, it := range items {
		if it.ID == id {
			existed = true
			continue
		}
		out = append(out, it)
	}
	if !existed {
		return false, nil
	}
	return true, s.save(out)
}
