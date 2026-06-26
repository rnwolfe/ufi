// Package auth resolves and persists UniFi API credentials. Precedence: environment
// (UNIFI_API_KEY / UNIFI_CLOUD_API_KEY) overrides the OS keyring, which overrides a 0600 XDG
// file. Secrets are read from stdin/env, never argv (contract §7). Storage degrades gracefully:
// if no OS keyring backend exists (common on headless hosts) it falls back to the 0600 file
// rather than blocking on a passphrase prompt.
package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
)

const (
	service = "ufi"
	credKey = "credentials"
)

// Creds is the resolved credential set for a run.
type Creds struct {
	Host        string
	APIKey      string
	CloudAPIKey string
	Insecure    bool
	Source      string // env | keyring | file | none
}

type stored struct {
	Host        string `json:"host,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	CloudAPIKey string `json:"cloud_api_key,omitempty"`
}

// Resolve merges env, keyring, and file. flagHost/flagInsecure come from the CLI (kong already
// folds UNIFI_HOST/UNIFI_INSECURE into them); the API keys are read from env here, never argv.
func Resolve(flagHost string, flagInsecure bool) Creds {
	s, src := load()
	c := Creds{Host: s.Host, APIKey: s.APIKey, CloudAPIKey: s.CloudAPIKey, Source: src, Insecure: flagInsecure}
	if v := os.Getenv("UNIFI_API_KEY"); v != "" {
		c.APIKey = v
		c.Source = "env"
	}
	if v := os.Getenv("UNIFI_CLOUD_API_KEY"); v != "" {
		c.CloudAPIKey = v
		if c.Source == "" || c.Source == "none" {
			c.Source = "env"
		}
	}
	if flagHost != "" {
		c.Host = flagHost
	}
	return c
}

// SaveLocal stores the console host + local API key (keyring, else 0600 file).
func SaveLocal(host, apiKey string) (string, error) {
	s, _ := load()
	s.Host = host
	s.APIKey = apiKey
	return save(s)
}

// SaveCloud stores the Site Manager cloud API key.
func SaveCloud(apiKey string) (string, error) {
	s, _ := load()
	s.CloudAPIKey = apiKey
	return save(s)
}

// Clear removes stored credentials from both the keyring and the file. It reports whether
// anything was removed.
func Clear() (bool, error) {
	removed := false
	if kr, err := openKeyring(); err == nil {
		if err := kr.Remove(credKey); err == nil {
			removed = true
		}
	}
	p, err := filePath()
	if err == nil {
		if err := os.Remove(p); err == nil {
			removed = true
		} else if !os.IsNotExist(err) {
			return removed, err
		}
	}
	return removed, nil
}

// FilePath returns the credential file path (for perm checks / doctor); it does not create it.
func FilePath() (string, error) { return filePath() }

// InsecureFilePerms reports true if the credential file exists and is group/other readable.
func InsecureFilePerms() (bool, string) {
	p, err := filePath()
	if err != nil {
		return false, ""
	}
	fi, err := os.Stat(p)
	if err != nil {
		return false, ""
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return true, p
	}
	return false, p
}

func load() (stored, string) {
	if kr, err := openKeyring(); err == nil {
		if it, err := kr.Get(credKey); err == nil {
			var s stored
			if json.Unmarshal(it.Data, &s) == nil && (s.APIKey != "" || s.Host != "" || s.CloudAPIKey != "") {
				return s, "keyring"
			}
		}
	}
	if s, ok := loadFile(); ok {
		return s, "file"
	}
	return stored{}, "none"
}

func save(s stored) (string, error) {
	data, _ := json.Marshal(s)
	if kr, err := openKeyring(); err == nil {
		if err := kr.Set(keyring.Item{Key: credKey, Data: data, Label: "ufi credentials"}); err == nil {
			return "keyring", nil
		}
	}
	if err := saveFile(s); err != nil {
		return "", err
	}
	return "file", nil
}

// openKeyring restricts to OS-native backends only — never the passphrase-prompting file
// backend, which would deadlock a headless agent (contract §7).
func openKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName:              service,
		KeychainName:             service,
		KeychainTrustApplication: true,
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.SecretServiceBackend,
			keyring.WinCredBackend,
		},
	})
}

func filePath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, service, "credentials"), nil
}

func loadFile() (stored, bool) {
	p, err := filePath()
	if err != nil {
		return stored{}, false
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return stored{}, false
	}
	var s stored
	if json.Unmarshal(b, &s) != nil {
		return stored{}, false
	}
	if s.APIKey == "" && s.Host == "" && s.CloudAPIKey == "" {
		return stored{}, false
	}
	return s, true
}

func saveFile(s stored) error {
	p, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

// ErrNoCreds is returned when no API key can be resolved.
var ErrNoCreds = errors.New("no credentials")
