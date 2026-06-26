package cli

import (
	"context"
	"os"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/ufi/internal/errs"
	"github.com/rnwolfe/ufi/internal/skill"
	"github.com/rnwolfe/ufi/internal/version"
)

// --- auth -------------------------------------------------------------------
// Auth is API-key only (X-API-KEY): no OAuth, no session, no refresh (spec: Auth). The
// skeleton reports env-var presence; cli-implement wires OS-keyring storage and live
// validation against GET /info. Secrets are read from stdin/env, never argv (contract §7).

type AuthCmd struct {
	Status  AuthStatusCmd  `cmd:"" help:"Show which credentials are stored and for which console."`
	Login   AuthLoginCmd   `cmd:"" help:"Store + validate an API key (read from stdin/env, never argv)."`
	Logout  AuthLogoutCmd  `cmd:"" help:"Remove stored credentials."`
	Refresh AuthRefreshCmd `cmd:"" help:"No-op: API keys do not expire or refresh."`
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *Runtime) error {
	return rt.Out.Emit(map[string]any{
		"console":       rt.Cfg.Host,
		"has_local_key": os.Getenv("UNIFI_API_KEY") != "",
		"has_cloud_key": os.Getenv("UNIFI_CLOUD_API_KEY") != "",
		"valid":         nil,
		"note":          "env-var presence only; keyring lookup + live validation wired by cli-implement",
	})
}

type AuthLoginCmd struct{}

func (c *AuthLoginCmd) Run(rt *Runtime) error {
	return errs.New(errs.ExitUnsupported, "NOT_IMPLEMENTED",
		"auth login is not yet wired",
		"cli-implement reads the API key from stdin/env (never argv), stores it in the OS keyring, and validates it against GET /info")
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rt *Runtime) error {
	return rt.Out.Emit(map[string]any{"ok": true, "cleared": false, "note": "keyring removal wired by cli-implement"})
}

type AuthRefreshCmd struct{}

func (c *AuthRefreshCmd) Run(rt *Runtime) error {
	rt.Out.Info("API keys do not expire or refresh; nothing to do")
	return rt.Out.Emit(map[string]any{"ok": true, "refreshed": false})
}

// --- doctor -----------------------------------------------------------------

type DoctorCmd struct{}

func (c *DoctorCmd) Run(rt *Runtime) error {
	hostSet := rt.Cfg.Host != ""
	keySet := os.Getenv("UNIFI_API_KEY") != ""
	hostDetail := "not set — pass --host or UNIFI_HOST"
	if hostSet {
		hostDetail = rt.Cfg.Host
	}
	keyDetail := "not set — set UNIFI_API_KEY or run `ufi auth login`"
	if keySet {
		keyDetail = "present (redacted)"
	}
	checks := []map[string]any{
		{"name": "host", "ok": hostSet, "detail": hostDetail},
		{"name": "api_key", "ok": keySet, "detail": keyDetail},
		{"name": "connectivity", "ok": true, "detail": "live console/TLS/key/clock checks wired by cli-implement"},
	}
	allOK := true
	for _, ch := range checks {
		if ok, _ := ch["ok"].(bool); !ok {
			allOK = false
		}
	}
	if !allOK {
		return errs.New(errs.ExitConfig, "DOCTOR_FAILED", "one or more checks failed",
			"set --host/UNIFI_HOST and UNIFI_API_KEY, then re-run `ufi doctor`")
	}
	return rt.Out.Emit(map[string]any{"ok": true, "checks": checks})
}

// --- schema -----------------------------------------------------------------

type SchemaCmd struct{}

func (c *SchemaCmd) Run(rt *Runtime) error {
	k, err := kong.New(&CLI{}, kong.Name("ufi"))
	if err != nil {
		return errs.New(errs.ExitGeneric, "SCHEMA_ERROR", err.Error(), "")
	}
	out := map[string]any{
		"tool":    "ufi",
		"version": version.String(),
		"conformance": map[string]any{
			"spec": "agent-cli-guidelines", "version": version.Spec, "level": "Full",
		},
		"commands":   nodeToMap(k.Model.Node),
		"exit_codes": errs.Table(),
		"safety": map[string]any{
			"allow_mutations": rt.Cfg.AllowMutations,
			"dry_run":         rt.Cfg.DryRun,
			"no_input":        rt.Cfg.NoInput,
		},
	}
	return rt.Out.EmitJSON(out) // schema is always JSON
}

func nodeToMap(n *kong.Node) map[string]any {
	m := map[string]any{"name": n.Name}
	if n.Help != "" {
		m["help"] = n.Help
	}
	var flags []map[string]any
	for _, f := range n.Flags {
		if f.Name == "help" {
			continue
		}
		fm := map[string]any{"name": f.Name}
		if f.Help != "" {
			fm["help"] = f.Help
		}
		if f.Default != "" {
			fm["default"] = f.Default
		}
		flags = append(flags, fm)
	}
	if len(flags) > 0 {
		m["flags"] = flags
	}
	var args []map[string]any
	for _, p := range n.Positional {
		args = append(args, map[string]any{"name": p.Name, "help": p.Help})
	}
	if len(args) > 0 {
		m["args"] = args
	}
	var subs []any
	for _, ch := range n.Children {
		subs = append(subs, nodeToMap(ch))
	}
	if len(subs) > 0 {
		m["subcommands"] = subs
	}
	return m
}

// --- agent ------------------------------------------------------------------

type AgentCmd struct{}

func (c *AgentCmd) Run(rt *Runtime) error {
	_, err := rt.Out.Stdout.Write([]byte(skill.Content))
	return err
}

// --- version ----------------------------------------------------------------

// VersionCmd prints the version, or with --check asks (pull-based, fail-silent) whether a
// newer release exists. Update awareness, never self-mutation (contract §11). The tool never
// auto-updates; it only reports the upgrade command for the human/package manager.

type VersionCmd struct {
	Check bool `help:"Check for a newer release (network, short timeout, fail-silent)."`
}

func (c *VersionCmd) Run(rt *Runtime) error {
	cur := version.String()
	if !c.Check {
		return rt.Out.Emit(map[string]any{"version": cur})
	}
	out := map[string]any{
		"current":         cur,
		"latest":          nil,
		"updateAvailable": false,
		"upgrade":         version.UpgradeHint(),
	}
	if latest, err := version.Latest(context.Background()); err == nil && latest != "" {
		out["latest"] = latest
		out["updateAvailable"] = version.UpdateAvailable(latest, cur)
	} else {
		out["note"] = "could not check for updates"
	}
	return rt.Out.Emit(out)
}
