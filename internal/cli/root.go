// Package cli wires the kong grammar, the runtime, and the exit-code mapping.
// main() does nothing but os.Exit(cli.Run(...)) so every path is testable in-process.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/ufi/internal/auth"
	"github.com/rnwolfe/ufi/internal/errs"
	"github.com/rnwolfe/ufi/internal/output"
	"github.com/rnwolfe/ufi/internal/skill"
)

// CLI is the kong grammar. Global flags are the universal agent-CLI contract surface;
// subcommands follow noun-verb grammar.
type CLI struct {
	// Output (contract §1, §6)
	Format   string `enum:"json,plain,tsv" default:"plain" help:"Output format: json, plain, or tsv."`
	JSON     bool   `help:"Shorthand for --format=json."`
	NoColor  bool   `help:"Disable colored output."`
	Limit    int    `default:"50" help:"Maximum items to return for list operations."`
	Select   string `help:"Comma-separated dot-path field projection, e.g. id,name."`
	Cursor   string `help:"Opaque pagination cursor from a previous response's nextCursor."`
	Page     int    `help:"Page number (1-based), an alternative to --cursor; ignored when --cursor is set."`
	Concise  bool   `help:"Terser output (default; accepted for contract uniformity)."`
	Detailed bool   `help:"Richer output (reserved; accepted for contract uniformity)."`
	NoFence  bool   `name:"no-fence" help:"Disable untrusted-text fencing of network-controlled names/notes (on by default in agent mode)."`

	// Safety (contract §2)
	AllowMutations bool `aliases:"write" help:"Permit state-changing operations (off by default). Alias: --write."`
	DryRun         bool `help:"Print intended mutations without performing them."`
	WrapUntrusted  bool `name:"wrap-untrusted" help:"Force untrusted-text fencing on (already default in agent mode)."`
	NoInput        bool `help:"Never prompt; fail with exit 13 instead."`

	// UniFi connection
	Host     string `env:"UNIFI_HOST" help:"UniFi console base URL or IP, e.g. https://192.168.1.1."`
	SiteRef  string `name:"site" env:"UNIFI_SITE" help:"Site id, name, or reference (default: the only site)."`
	Insecure bool   `env:"UNIFI_INSECURE" help:"Skip TLS verification (consoles ship a self-signed cert). Off by default; warns when on."`
	UseCloud bool   `name:"cloud" help:"Route the read through the Site Manager cloud API (api.ui.com) instead of the local console."`

	// Commands — read-first core
	Info    InfoCmd    `cmd:"" help:"Show controller version and capabilities."`
	Site    SiteCmd    `cmd:"" help:"List sites on the console."`
	Device  DeviceCmd  `cmd:"" help:"Inspect and act on adopted devices."`
	Client  ClientCmd  `cmd:"" help:"Inspect and authorize connected clients."`
	Wifi    WifiCmd    `cmd:"" help:"Inspect WiFi broadcasts (SSIDs)."`
	Voucher VoucherCmd `cmd:"" help:"Manage hotspot vouchers."`

	// Commands — declarative config (preview → apply <hash>, contract §2)
	Network     NetworkCmd     `cmd:"" help:"Manage networks (VLAN/LAN)."`
	Firewall    FirewallCmd    `cmd:"" help:"Manage firewall zones and policies (Zone-Based Firewall)."`
	Acl         AclCmd         `cmd:"" help:"Manage switch ACL rules."`
	Dns         DnsCmd         `cmd:"" help:"Manage DNS policies."`
	TrafficList TrafficListCmd `cmd:"" name:"traffic-list" help:"Manage traffic-matching lists."`
	Apply       ApplyCmd       `cmd:"" help:"Execute a previously previewed config plan by its hash."`

	// Commands — cloud fleet (Site Manager)
	Cloud CloudCmd `cmd:"" help:"Cross-host fleet reads via the Site Manager cloud API."`

	// Commands — universal contract surface
	Auth    AuthCmd    `cmd:"" help:"Manage authentication (API keys)."`
	Doctor  DoctorCmd  `cmd:"" help:"Diagnose connectivity, TLS, key validity, and clock."`
	Schema  SchemaCmd  `cmd:"" help:"Print the machine-readable command schema (JSON)."`
	Agent   AgentCmd   `cmd:"" help:"Print the bundled agent SKILL.md."`
	Version VersionCmd `cmd:"" help:"Print the version."`
}

// Runtime is the per-invocation context bound into every command's Run method.
type Runtime struct {
	Cfg      *CLI
	Out      *output.Writer
	Creds    auth.Creds
	Stdin    io.Reader
	Fence    bool // wrap network-controlled free text as untrusted (contract §8)
	ExitCode int  // optional non-zero exit set by a command after emitting (e.g. EMPTY)
}

// Guard enforces the read-only-by-default mutation gate (contract §2).
func (rt *Runtime) Guard(op string) error {
	if rt.Cfg.AllowMutations {
		return nil
	}
	return errs.MutationBlocked(op)
}

// helpDescription is the example-led root help (contract §5): runnable invocations first.
const helpDescription = `An agent-friendly CLI for Ubiquiti UniFi Network over the official API.
Read-only by default; state-changing commands require --allow-mutations.

Examples:
  ufi doctor --json                         # check host/key/connectivity
  ufi device list --json --select id,name   # adopted devices (bounded, projected)
  ufi client list --json --limit 20         # connected clients (paged; use --cursor/--page)
  ufi device restart <id> --allow-mutations # a gated single-target action
  ufi network update <id> --data @net.json --allow-mutations   # prints a plan + hash
  ufi apply <hash> --allow-mutations        # execute exactly that previewed plan

Auth: set UNIFI_HOST and UNIFI_API_KEY (or pipe a key to ` + "`ufi auth login`" + `).
Run ` + "`ufi agent`" + ` for the full embedded usage contract, or ` + "`ufi schema`" + ` for JSON.`

// Run parses args and dispatches, returning the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// Terse agent help mode (contract §5): UFI_HELP=agent + a help request prints the
	// machine-skimmable embedded contract instead of the full kong help.
	if os.Getenv("UFI_HELP") == "agent" && wantsHelp(args) {
		_, _ = io.WriteString(stdout, skill.Content)
		return errs.ExitOK
	}

	var cfg CLI
	helpShown := false
	parser, err := kong.New(&cfg,
		kong.Name("ufi"),
		kong.Description(helpDescription),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) { helpShown = true }), // --help/--version: we control exit
	)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return errs.ExitGeneric
	}

	kctx, perr := parser.Parse(args)
	if helpShown {
		return errs.ExitOK
	}
	if perr != nil {
		return handleParseError(stderr, args, perr)
	}

	if cfg.JSON {
		cfg.Format = "json"
	}
	rt := newRuntime(&cfg, stdin, stdout, stderr)

	// Loud warning when TLS verification is disabled (contract: never silent).
	if cfg.Insecure {
		rt.Out.Info("warning: TLS verification disabled (--insecure / UNIFI_INSECURE); the console's identity is not verified")
	}

	if err := kctx.Run(rt); err != nil {
		return emitError(rt, err)
	}
	return rt.ExitCode // 0 unless a command set an override (e.g. EMPTY)
}

// wantsHelp reports whether args is a help invocation (no args, or -h/--help present).
func wantsHelp(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

func newRuntime(cfg *CLI, stdin io.Reader, stdout, stderr io.Writer) *Runtime {
	format := output.Format(cfg.Format)
	color := !cfg.NoColor && os.Getenv("NO_COLOR") == "" && isTTY(stdout) && format == output.FormatPlain
	var sel []string
	if cfg.Select != "" {
		sel = strings.Split(cfg.Select, ",")
	}
	w := &output.Writer{
		Stdout: stdout, Stderr: stderr,
		Format: format, Color: color, Limit: cfg.Limit, Select: sel,
	}
	creds := auth.Resolve(cfg.Host, cfg.Insecure)
	// Agent mode = JSON output or a non-TTY stdout; fence untrusted text there by default.
	// --wrap-untrusted forces it on; --no-fence forces it off.
	agent := format == output.FormatJSON || !isTTY(stdout)
	fence := !cfg.NoFence && (agent || cfg.WrapUntrusted)
	return &Runtime{Cfg: cfg, Out: w, Creds: creds, Stdin: stdin, Fence: fence}
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// emitError prints a structured error to stderr and returns its exit code (contract §3).
func emitError(rt *Runtime, err error) int {
	var ce *errs.CLIError
	if !errors.As(err, &ce) {
		ce = errs.New(errs.ExitGeneric, "INTERNAL", err.Error(), "")
	}
	if rt.Out.Format == output.FormatJSON {
		enc := json.NewEncoder(rt.Out.Stderr)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"error":       ce.Message,
			"code":        ce.Code,
			"remediation": ce.Remediation,
		})
	} else {
		fmt.Fprintf(rt.Out.Stderr, "error: %s\n", ce.Message)
		if ce.Code != "" {
			fmt.Fprintf(rt.Out.Stderr, "  code: %s\n", ce.Code)
		}
		if ce.Remediation != "" {
			fmt.Fprintf(rt.Out.Stderr, "  fix:  %s\n", ce.Remediation)
		}
	}
	return ce.Exit
}

// handleParseError reports usage errors. kong already appends its own "did you mean" for close
// misspellings, so we only add one when kong didn't, and never second-guess a token that is
// already a valid command (or an unknown flag).
func handleParseError(stderr io.Writer, args []string, err error) int {
	msg := err.Error()
	fmt.Fprintf(stderr, "error: %s\n", msg)
	if strings.Contains(msg, "did you mean") {
		return errs.ExitUsage
	}
	commands := []string{
		"info", "site", "device", "client", "wifi", "voucher",
		"network", "firewall", "acl", "dns", "traffic-list", "apply",
		"cloud", "auth", "doctor", "schema", "agent", "version",
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue // unknown flag — don't suggest a command for it
		}
		if contains(commands, a) {
			break // already a valid command; nothing to suggest
		}
		if s, ok := closest(a, commands); ok {
			fmt.Fprintf(stderr, "  did you mean %q?\n", s)
		}
		break
	}
	return errs.ExitUsage
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
