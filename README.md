<div align="center">

# ufi

**An agent-friendly CLI for Ubiquiti UniFi Network.**

Capable UniFi CLIs exist — but they're built for humans, or speak the brittle reverse-engineered
controller API. `ufi` is built for **AI agents** on Ubiquiti's **official** local Network
Integration API, with the [Agent CLI Guidelines](https://aclig.dev/) baked in: read-only by
default, a single `--allow-mutations` gate (plus a reviewed-artifact `apply <hash>` flow for
config), machine-discoverable `schema`, structured errors + stable exit codes, bounded JSON, and
prompt-injection fencing — all in one static binary.

[![ci](https://github.com/rnwolfe/ufi/actions/workflows/ci.yml/badge.svg)](https://github.com/rnwolfe/ufi/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/rnwolfe/ufi?sort=semver)](https://github.com/rnwolfe/ufi/releases/latest)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![Agent CLI Guidelines v0.4.0: Full](https://aclig.dev/badge/agent-cli-guidelines-full.svg)](https://aclig.dev/conformance/)

![ufi demo](./demo/ufi.gif)

</div>

## Why ufi

- **Read-only by default; mutations are gated.** Listing devices, clients, WiFi, firewall, DNS —
  all safe. State-changing commands need an explicit `--allow-mutations`; a blocked one returns a
  structured `MUTATION_BLOCKED` (exit 12), so an agent can tell "ask permission" from "real error."
- **High-stakes config is reviewed before it runs.** `network`/`firewall`/`acl`/`dns` writes don't
  execute directly: `--dry-run` emits a plan + hash, and `ufi apply <hash>` runs *exactly* that
  plan — closing the time-of-check/time-of-use gap a blind confirmation opens.
- **Built for agents, not just humans.** `--json` everywhere, `schema --json` (with a conformance
  block), an embedded `agent` guide, `{error,code,remediation}` errors, a stable exit-code table,
  and `--limit`/`--select`/`--cursor` so responses stay within an agent's context budget.
- **Prompt-injection fenced.** A guest can name their device `Ignore previous instructions…`.
  ufi wraps network-controlled names/notes `[UNTRUSTED_DATA_BEGIN]…[UNTRUSTED_DATA_END]` by default.
- **Official API only.** Ubiquiti's versioned Integration API — no reverse-engineered endpoints,
  so it stays stable as UniFi updates. Validated against UniFi Network 10.4.57.

## Install

```bash
# Homebrew (macOS / Linux) — recommended
brew install rnwolfe/tap/ufi

# Go (any platform; static, no CGO)
go install github.com/rnwolfe/ufi/cmd/ufi@latest

# Shell script (macOS / Linux) — downloads a release binary, verifies its SHA-256, installs to ~/.local/bin
curl -fsSL https://uficli.sh/install.sh | sh
```

Or grab a prebuilt binary (linux/macOS/windows, amd64/arm64) from
[Releases](https://github.com/rnwolfe/ufi/releases) — each ships with checksums, an SBOM, and
build-provenance attestation.

## Quickstart

No console needed for the first look — `ufi` describes itself offline:

```bash
ufi --help
ufi agent                                  # the embedded agent guide (SKILL.md)
ufi schema | jq '{conformance, exit_codes}' # machine-readable contract
```

Then point it at your console. Generate an API key in the UniFi UI
(**Settings → Control Plane → Integrations → Create API Key**), then:

```bash
export UNIFI_HOST=https://192.168.1.1
export UNIFI_API_KEY=…          # or: printf %s "$KEY" | ufi auth login  (stored in the OS keyring)
# self-signed console cert? add --insecure (or UNIFI_INSECURE=1)

ufi doctor --json                          # host / key / connectivity / TLS / perms
ufi device list --json --select id,name,model,state
ufi client list --json --limit 20
ufi device restart <id> --allow-mutations  # mutations are opt-in
```

## Authentication

`X-API-KEY` — no OAuth, no session, no refresh. The key is read from **stdin** (`ufi auth login`)
or the `UNIFI_API_KEY` env var, **never argv** (argv leaks to `ps`/`/proc`/shell history), and
stored in the OS keyring with a `0600` file fallback on headless hosts.

- `ufi auth login` — pipe the key on stdin; validates it against `GET /info` and stores it.
- `ufi auth status --json` — reports which key is stored and whether it works (never prints it).
- `ufi auth logout` — removes the **local** copy only (revoke on the console to invalidate it).
- `ufi doctor --json` — full diagnostics with a fix for each failure.

A UniFi API key is effectively **full-admin** (it bypasses per-admin RBAC) — treat it like a root
password. A missing/invalid key returns `AUTH_REQUIRED` (exit 4) naming the login command.

## Commands

```bash
ufi info                       # controller version
ufi site list                  # sites on the console
ufi device list|get|stats <id> # adopted devices + latest statistics
ufi device restart <id>            # (mutation) reboot a device
ufi device port-cycle <id> <port>  # (mutation) PoE power-cycle a switch port
ufi client list|get <id>       # connected clients
ufi client authorize|unauthorize <id>   # (mutation) guest access
ufi wifi list|get <id>         # WiFi broadcasts (SSIDs)
ufi voucher list                   # hotspot vouchers
ufi voucher create <name> --minutes N   # (mutation) generate vouchers
ufi voucher delete <id>                 # (mutation, idempotent)
ufi network|firewall|acl|dns|traffic-list …   # declarative config (preview → apply <hash>)
ufi apply <hash>               # (mutation) execute a previewed config plan
ufi auth login|status|logout|refresh
ufi doctor                     # diagnostics
ufi schema                     # machine-readable command tree, flags, exit codes, safety
ufi agent                      # print the embedded agent guide (SKILL.md)
ufi version [--check]          # installed version; --check pulls GitHub for the latest release
```

> ufi never auto-updates. On the human path it prints a once-a-day upgrade hint to stderr
> (silent for agents; disable with `UFI_NO_UPDATE_CHECK=1`).

Global flags: `--json`/`--format json|plain|tsv`, `--select a,b.c`, `--limit N`, `--cursor`/`--page`,
`--allow-mutations`/`--write`, `--dry-run`, `--host`, `--site`, `--insecure`, `--no-fence`, `--no-input`.

> **Local-only for now.** The Site Manager **cloud** surface (`ufi cloud …`) is deferred — it
> returns `UNSUPPORTED` with a pointer to open an issue. Want it? [File one](https://github.com/rnwolfe/ufi/issues).

## Exit codes

`0` ok · `2` usage · `3` empty results · `4` auth required · `5` not found · `6` permission ·
`7` rate limited · `8` retryable · `10` config · `11` unsupported (needs a console feature, e.g.
Zone-Based Firewall) · `12` mutation blocked · `13` input required · `130` cancelled.
Full table: `ufi schema`.

## Conformance

ufi conforms to the [Agent CLI Guidelines](https://aclig.dev/) **v0.4.0** at the **Full** level.
The contract version is machine-verifiable from the binary — `ufi schema` emits a `conformance`
block, so an agent can confirm the standard it was built against without trusting this README:

```json
{
  "conformance": {
    "level": "Full",
    "spec": "agent-cli-guidelines",
    "version": "0.4.0"
  }
}
```

## Safety

- **Read-only by default**; every mutation is behind `--allow-mutations`, and declarative config
  goes through the reviewed-artifact `apply <hash>` flow.
- **Untrusted text fenced** by default in agent mode (`--no-fence` to disable).
- **Secrets** via stdin/env → OS keyring (`0600` file fallback); never argv; redacted in output.

See [SECURITY.md](./SECURITY.md) for the secret-handling threat model.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) and [AGENTS.md](./AGENTS.md). Build/test:
`go build ./... && go vet ./... && go test ./...`.

## License

[MIT](./LICENSE) © Ryan Wolfe
