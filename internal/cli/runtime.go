package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rnwolfe/ufi/internal/errs"
	"github.com/rnwolfe/ufi/internal/plan"
	"github.com/rnwolfe/ufi/internal/unifi"
)

func itoa(n int) string { return strconv.Itoa(n) }

func (rt *Runtime) ctx() context.Context { return context.Background() }

// local builds a client for the Integration API, or a structured AUTH_REQUIRED error.
func (rt *Runtime) local() (*unifi.Client, error) {
	if rt.Creds.APIKey == "" {
		return nil, errs.New(errs.ExitAuth, "AUTH_REQUIRED", "no UniFi API key configured",
			"run `ufi auth login`, or set UNIFI_API_KEY and --host/UNIFI_HOST")
	}
	return unifi.NewLocal(rt.Creds.Host, rt.Creds.APIKey, unifi.Options{Insecure: rt.Creds.Insecure})
}

func (rt *Runtime) listOpts() unifi.ListOpts {
	return unifi.ListOpts{Limit: rt.Cfg.Limit, Cursor: rt.Cfg.Cursor, Page: rt.Cfg.Page}
}

func (rt *Runtime) emitList(res *unifi.ListResult) error {
	// A successful query that resolved to zero items exits EMPTY (3) after emitting the
	// envelope — an agent can branch on the code without parsing (contract §4).
	if res.Count == 0 {
		rt.ExitCode = errs.ExitEmpty
	}
	return rt.Out.Emit(listEnvelope(res.Items, res.Count, res.NextCursor))
}

// resolveSite returns the site id from --site (matching id, name, or reference), or the only
// site when --site is unset. Errors helpfully when ambiguous or not found.
func (rt *Runtime) resolveSite(ctx context.Context, c *unifi.Client) (string, error) {
	res, err := c.List(ctx, "/sites", unifi.ListOpts{Limit: 200})
	if err != nil {
		return "", err
	}
	want := strings.TrimSpace(rt.Cfg.SiteRef)
	if want == "" {
		switch len(res.Items) {
		case 1:
			return siteID(res.Items[0]), nil
		case 0:
			return "", errs.New(errs.ExitNotFound, "NOT_FOUND", "no sites found on this console", "verify the API key has access")
		default:
			return "", errs.New(errs.ExitUsage, "USAGE", "multiple sites; specify --site",
				"choices: "+strings.Join(siteNames(res.Items), ", "))
		}
	}
	for _, it := range res.Items {
		if siteMatches(it, want) {
			return siteID(it), nil
		}
	}
	return "", errs.New(errs.ExitNotFound, "NOT_FOUND", "site not found: "+want, "run `ufi site list`")
}

// siteList is the common read path: resolve client + site, list a site-scoped subpath, emit.
// untrusted names the free-text keys to fence in agent mode (contract §8).
func (rt *Runtime) siteList(subpath string, untrusted ...string) error {
	c, err := rt.local()
	if err != nil {
		return err
	}
	ctx := rt.ctx()
	site, err := rt.resolveSite(ctx, c)
	if err != nil {
		return err
	}
	res, err := c.List(ctx, "/sites/"+site+"/"+subpath, rt.listOpts())
	if err != nil {
		return err
	}
	rt.fenceItems(res.Items, untrusted)
	return rt.emitList(res)
}

// siteGet fetches one site-scoped resource and emits it.
func (rt *Runtime) siteGet(subpath string, untrusted ...string) error {
	c, err := rt.local()
	if err != nil {
		return err
	}
	ctx := rt.ctx()
	site, err := rt.resolveSite(ctx, c)
	if err != nil {
		return err
	}
	v, err := c.GetObject(ctx, "/sites/"+site+"/"+subpath)
	if err != nil {
		return err
	}
	rt.fenceValue(v, untrusted)
	return rt.Out.Emit(v)
}

// fencePrefix/fenceSuffix delimit network-controlled free text so a downstream agent does not
// execute instructions embedded in a device/client/SSID name (contract §8).
const (
	fencePrefix = "[UNTRUSTED_DATA_BEGIN] "
	fenceSuffix = " [UNTRUSTED_DATA_END]"
)

func (rt *Runtime) fenceItems(items []any, keys []string) {
	if !rt.Fence || len(keys) == 0 {
		return
	}
	for _, it := range items {
		rt.fenceValue(it, keys)
	}
}

func (rt *Runtime) fenceValue(v any, keys []string) {
	if !rt.Fence || len(keys) == 0 {
		return
	}
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			m[k] = fencePrefix + s + fenceSuffix
		}
	}
}

// siteAction runs a low-stakes single-target action: gate → dry-run preview → POST and emit.
func (rt *Runtime) siteAction(op, subpath string, body map[string]any, preview map[string]any) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	if rt.Cfg.DryRun {
		preview["dry_run"] = true
		return rt.Out.Emit(preview)
	}
	c, err := rt.local()
	if err != nil {
		return err
	}
	ctx := rt.ctx()
	site, err := rt.resolveSite(ctx, c)
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(body)
	v, err := c.Send(ctx, http.MethodPost, "/sites/"+site+"/"+subpath, raw)
	if err != nil {
		return err
	}
	out := map[string]any{"ok": true}
	for k, val := range preview {
		out[k] = val
	}
	if v != nil {
		out["result"] = v
	}
	return rt.Out.Emit(out)
}

// siteDelete deletes a site-scoped resource idempotently: a NOT_FOUND is a soft success
// (contract §9). Gated; --dry-run previews.
func (rt *Runtime) siteDelete(op, subpath, kind, id string) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dry_run": true, "action": op, "id": id})
	}
	c, err := rt.local()
	if err != nil {
		return err
	}
	ctx := rt.ctx()
	site, err := rt.resolveSite(ctx, c)
	if err != nil {
		return err
	}
	_, err = c.Send(ctx, http.MethodDelete, "/sites/"+site+"/"+subpath, nil)
	existed := true
	if err != nil {
		var ce *errs.CLIError
		if ok := asCLIError(err, &ce); ok && ce.Code == "NOT_FOUND" {
			existed = false
		} else {
			return err
		}
	}
	return rt.Out.Emit(map[string]any{"ok": true, "kind": kind, "id": id, "existed": existed})
}

// configWrite previews a declarative config change (reviewed-artifact apply, contract §2):
// gate → persist a plan keyed by hash → emit the plan. No network at preview time; the site is
// resolved and the change executed by `ufi apply <hash>`. subpath is site-relative.
func (rt *Runtime) configWrite(op, method, subpath string, body []byte, summary map[string]any) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	p := plan.New(op, method, subpath, body, summary)
	if err := plan.Save(p); err != nil {
		return errs.New(errs.ExitConfig, "PLAN_SAVE_FAILED", err.Error(), "check $XDG_STATE_HOME/ufi is writable")
	}
	return rt.Out.Emit(map[string]any{
		"action":  op,
		"method":  method,
		"path":    subpath,
		"hash":    p.Hash,
		"plan":    summary,
		"dry_run": true,
		"note":    "preview only — run `ufi apply " + p.Hash + " --allow-mutations` to execute",
	})
}

// readData loads a config body from --data: '-' (stdin), inline JSON, or a file path (@file ok).
func (rt *Runtime) readData(data string) ([]byte, error) {
	var b []byte
	var err error
	switch {
	case data == "-":
		b, err = io.ReadAll(rt.Stdin)
	case strings.HasPrefix(strings.TrimSpace(data), "{"), strings.HasPrefix(strings.TrimSpace(data), "["):
		b = []byte(data)
	default:
		b, err = os.ReadFile(strings.TrimPrefix(data, "@"))
	}
	if err != nil {
		return nil, errs.New(errs.ExitConfig, "CONFIG", "could not read --data: "+err.Error(), "pass inline JSON, a file path, or '-' for stdin")
	}
	if !json.Valid(b) {
		return nil, errs.New(errs.ExitUsage, "USAGE", "--data is not valid JSON", "provide a JSON object body")
	}
	return b, nil
}

// --- site helpers -----------------------------------------------------------

func siteID(it any) string {
	if m, ok := it.(map[string]any); ok {
		if s, ok := m["id"].(string); ok {
			return s
		}
	}
	return ""
}

func siteNames(items []any) []string {
	var out []string
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			if s, ok := m["name"].(string); ok && s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func siteMatches(it any, want string) bool {
	m, ok := it.(map[string]any)
	if !ok {
		return false
	}
	for _, k := range []string{"id", "internal_reference", "name"} {
		if s, ok := m[k].(string); ok && strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}

// asCLIError unwraps to a *errs.CLIError without importing errors at every call site.
func asCLIError(err error, target **errs.CLIError) bool {
	if ce, ok := err.(*errs.CLIError); ok {
		*target = ce
		return true
	}
	return false
}
