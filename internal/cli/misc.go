package cli

import (
	"context"
	"io"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/ufi/internal/auth"
	"github.com/rnwolfe/ufi/internal/errs"
	"github.com/rnwolfe/ufi/internal/skill"
	"github.com/rnwolfe/ufi/internal/unifi"
	"github.com/rnwolfe/ufi/internal/version"
)

// --- auth -------------------------------------------------------------------
// Auth is API-key only (X-API-KEY): no OAuth, no session, no refresh (spec: Auth). Keys are read
// from stdin/env (never argv), stored in the OS keyring with a 0600-file fallback, and validated
// against GET /info (contract §7).

type AuthCmd struct {
	Status  AuthStatusCmd  `cmd:"" help:"Show which credentials are stored and whether they work."`
	Login   AuthLoginCmd   `cmd:"" help:"Store + validate an API key (piped on stdin, never argv)."`
	Logout  AuthLogoutCmd  `cmd:"" help:"Remove stored credentials (local only)."`
	Refresh AuthRefreshCmd `cmd:"" help:"No-op: API keys do not expire or refresh."`
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *Runtime) error {
	cr := rt.Creds
	out := map[string]any{
		"console":       cr.Host,
		"has_local_key": cr.APIKey != "",
		"has_cloud_key": cr.CloudAPIKey != "",
		"source":        cr.Source,
		"valid":         nil,
	}
	if bad, p := auth.InsecureFilePerms(); bad {
		out["warning"] = "credential file is group/other readable: " + p
	}
	var problem error
	if cr.APIKey == "" {
		problem = errs.New(errs.ExitAuth, "AUTH_REQUIRED", "no local API key configured",
			"pipe a key to `ufi auth login`, or set UNIFI_API_KEY and --host/UNIFI_HOST")
	} else if cr.Host != "" {
		if cl, err := unifi.NewLocal(cr.Host, cr.APIKey, unifi.Options{Insecure: cr.Insecure}); err == nil {
			if ver, err := cl.Validate(rt.ctx()); err == nil {
				out["valid"] = true
				out["application_version"] = ver
			} else {
				out["valid"] = false
				problem = err
			}
		}
	}
	_ = rt.Out.Emit(out)
	return problem
}

// AuthLoginCmd reads the API key from stdin (never argv) and validates+stores it.
type AuthLoginCmd struct {
	Cloud bool `name:"cloud-key" help:"Store a Site Manager cloud API key (from unifi.ui.com) instead of the local console key."`
}

func (c *AuthLoginCmd) Run(rt *Runtime) error {
	key, _ := io.ReadAll(rt.Stdin)
	k := strings.TrimSpace(string(key))
	if k == "" {
		if rt.Cfg.NoInput {
			return errs.InputRequired("API key on stdin")
		}
		return errs.New(errs.ExitUsage, "USAGE", "no API key on stdin",
			"pipe the key in, e.g.  printf %s \"$UNIFI_API_KEY\" | ufi auth login")
	}
	if c.Cloud {
		src, err := auth.SaveCloud(k)
		if err != nil {
			return errs.New(errs.ExitConfig, "STORE_FAILED", err.Error(), "check keyring/credential-file permissions")
		}
		return rt.Out.Emit(map[string]any{"ok": true, "scope": "cloud", "stored": src})
	}
	host := rt.Creds.Host
	if host == "" {
		return errs.New(errs.ExitConfig, "CONFIG", "no console host set", "pass --host or set UNIFI_HOST, e.g. https://192.168.1.1")
	}
	cl, err := unifi.NewLocal(host, k, unifi.Options{Insecure: rt.Creds.Insecure})
	if err != nil {
		return err
	}
	ver, err := cl.Validate(rt.ctx())
	if err != nil {
		return err // AUTH_REQUIRED / TRANSIENT etc. — the key/host didn't validate
	}
	src, err := auth.SaveLocal(host, k)
	if err != nil {
		return errs.New(errs.ExitConfig, "STORE_FAILED", err.Error(), "check keyring/credential-file permissions")
	}
	if bad, p := auth.InsecureFilePerms(); bad {
		rt.Out.Info("warning: credential file is group/other readable: %s (chmod 600 it)", p)
	}
	return rt.Out.Emit(map[string]any{"ok": true, "console": host, "application_version": ver, "stored": src})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rt *Runtime) error {
	removed, err := auth.Clear()
	if err != nil {
		return errs.New(errs.ExitConfig, "LOGOUT_FAILED", err.Error(), "remove the credential file manually")
	}
	rt.Out.Info("cleared local credentials only; the API key is still valid on the console until you revoke it there")
	return rt.Out.Emit(map[string]any{"ok": true, "cleared": removed})
}

type AuthRefreshCmd struct{}

func (c *AuthRefreshCmd) Run(rt *Runtime) error {
	rt.Out.Info("API keys do not expire or refresh; nothing to do")
	return rt.Out.Emit(map[string]any{"ok": true, "refreshed": false})
}

// --- doctor -----------------------------------------------------------------

type DoctorCmd struct{}

func (c *DoctorCmd) Run(rt *Runtime) error {
	cr := rt.Creds
	var checks []map[string]any
	add := func(name string, ok bool, detail, fix string) {
		ch := map[string]any{"name": name, "ok": ok, "detail": detail}
		if !ok && fix != "" {
			ch["fix"] = fix
		}
		checks = append(checks, ch)
	}

	hostOK := cr.Host != ""
	hostDetail := cr.Host
	if !hostOK {
		hostDetail = "not set"
	}
	add("host", hostOK, hostDetail, "pass --host or set UNIFI_HOST (e.g. https://192.168.1.1)")

	keyOK := cr.APIKey != ""
	keyDetail := "present (redacted), source=" + cr.Source
	if !keyOK {
		keyDetail = "not set"
	}
	add("api_key", keyOK, keyDetail, "pipe a key to `ufi auth login` or set UNIFI_API_KEY")

	if hostOK && keyOK {
		if cl, err := unifi.NewLocal(cr.Host, cr.APIKey, unifi.Options{Insecure: cr.Insecure}); err == nil {
			if ver, err := cl.Validate(rt.ctx()); err == nil {
				add("connectivity", true, "console reachable, key valid, version "+ver, "")
			} else {
				add("connectivity", false, err.Error(), "verify host/key; add --insecure for a self-signed console cert")
			}
		}
	} else {
		add("connectivity", false, "skipped — host/key missing", "configure host + key first")
	}

	if bad, p := auth.InsecureFilePerms(); bad {
		add("cred_perms", false, "credential file is group/other readable: "+p, "chmod 600 "+p)
	}

	allOK := true
	for _, ch := range checks {
		if ok, _ := ch["ok"].(bool); !ok {
			allOK = false
		}
	}
	if !allOK {
		_ = rt.Out.Emit(map[string]any{"ok": false, "checks": checks})
		return errs.New(errs.ExitConfig, "DOCTOR_FAILED", "one or more checks failed", "see each failing check's fix")
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
