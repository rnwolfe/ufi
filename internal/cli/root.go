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

	"github.com/rnwolfe/ufi/internal/errs"
	"github.com/rnwolfe/ufi/internal/output"
	"github.com/rnwolfe/ufi/internal/store"
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
	Concise  bool   `help:"Terser output (default)."`
	Detailed bool   `help:"Richer output."`

	// Safety (contract §2)
	AllowMutations bool `help:"Permit state-changing operations (off by default)."`
	DryRun         bool `help:"Print intended mutations without performing them."`
	Yes            bool `help:"Assume yes for confirmations (scripting)."`
	Force          bool `help:"Bypass safety checks."`
	NoInput        bool `help:"Never prompt; fail with exit 13 instead."`

	// UniFi connection
	Host     string `env:"UNIFI_HOST" help:"UniFi console base URL or IP, e.g. https://192.168.1.1."`
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
	Cfg   *CLI
	Out   *output.Writer
	Store *store.Store
	Stdin io.Reader
}

// Guard enforces the read-only-by-default mutation gate (contract §2).
func (rt *Runtime) Guard(op string) error {
	if rt.Cfg.AllowMutations {
		return nil
	}
	return errs.MutationBlocked(op)
}

// Run parses args and dispatches, returning the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cfg CLI
	helpShown := false
	parser, err := kong.New(&cfg,
		kong.Name("ufi"),
		kong.Description("An agent-friendly CLI for Ubiquiti UniFi Network (official API). Read-only by default; mutations require --allow-mutations."),
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
	return errs.ExitOK
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
	return &Runtime{Cfg: cfg, Out: w, Store: store.New(store.DefaultPath()), Stdin: stdin}
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

// handleParseError reports usage errors and offers a "did you mean" suggestion.
func handleParseError(stderr io.Writer, args []string, err error) int {
	fmt.Fprintf(stderr, "error: %s\n", err)
	commands := []string{
		"info", "site", "device", "client", "wifi", "voucher",
		"network", "firewall", "acl", "dns", "traffic-list", "apply",
		"cloud", "auth", "doctor", "schema", "agent", "version",
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if s, ok := closest(a, commands); ok {
			fmt.Fprintf(stderr, "  did you mean %q?\n", s)
		}
		break
	}
	return errs.ExitUsage
}
