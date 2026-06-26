---
title: Quickstart
description: Install ufi, explore its contract offline, then point it at your UniFi console for bounded JSON reads and your first gated mutation.
---

`ufi` is a CLI for **Ubiquiti UniFi Network** built on Ubiquiti's **official local Network
Integration API** — no reverse-engineered controller endpoints, no fragile legacy sessions. It
reads devices, clients, WiFi, firewall rules, DNS policies, and more as bounded, structured JSON.
It is **read-only by default**: every state-changing command requires an explicit
`--allow-mutations` flag, and high-stakes declarative config goes through a `plan + hash → apply`
flow that closes the time-of-check/time-of-use gap.

## Install

Pick the method that fits your environment. All options produce a single static binary.

```bash
# Homebrew (macOS / Linux) — recommended; gets upgrades through brew
brew install rnwolfe/tap/ufi

# Go (any platform; static binary, no CGO required)
go install github.com/rnwolfe/ufi/cmd/ufi@latest

# Shell script (macOS / Linux) — downloads a release binary, verifies its SHA-256, installs to ~/.local/bin
curl -fsSL https://uficli.sh/install.sh | sh
```

Prebuilt binaries for linux/macOS/windows on amd64/arm64 are also available on the
[Releases page](https://github.com/rnwolfe/ufi/releases), each with checksums, an SBOM, and
build-provenance attestation.

For a full walkthrough of all install methods, see [Installation](/installation/).

## Learn the contract offline

You do not need a console to explore `ufi`. The binary describes itself completely:

```bash
ufi --help
```

The `--help` output leads with runnable examples — reads first, mutations clearly flagged.

```bash
ufi agent
```

`ufi agent` prints the embedded `SKILL.md`: a concise machine-readable guide to the full command
surface, output format, safety gates, and error codes. Agents can load this at startup to
understand ufi's contract without reading the docs.

```bash
ufi schema | jq '{conformance, exit_codes}'
```

`ufi schema` emits a machine-readable JSON document: the full command tree (flags, args,
subcommands), the stable exit-code table, and the live safety state (`allow_mutations`,
`dry_run`, `no_input`). The `conformance` block lets an agent verify the spec level from the
binary itself:

```json
{
  "conformance": {
    "level": "Full",
    "spec": "agent-cli-guidelines",
    "version": "0.4.0"
  },
  "exit_codes": {
    "auth_required": 4,
    "cancelled": 130,
    "config_error": 10,
    "empty_results": 3,
    "generic_error": 1,
    "input_required": 13,
    "mutation_blocked": 12,
    "not_found": 5,
    "ok": 0,
    "permission": 6,
    "rate_limited": 7,
    "retryable": 8,
    "unsupported": 11,
    "usage": 2
  }
}
```

If you prefer the terse machine-contract variant instead of full kong help, set
`UFI_HELP=agent` and run `ufi --help` — it prints `SKILL.md` directly:

```bash
UFI_HELP=agent ufi --help
```

## Point it at your console

### Generate an API key

Open the UniFi UI and go to **Settings → Control Plane → Integrations → Create API Key**. Copy
the key — you will not see it again after closing that dialog.

A UniFi API key is effectively **full admin**: it bypasses per-admin RBAC. Treat it like a root
password. See [Authentication](/auth/) for the full threat model, keyring storage, and credential
precedence.

### Set the environment

```bash
export UNIFI_HOST=https://192.168.1.1   # IP or hostname of your console
export UNIFI_API_KEY=<your-key>         # env is the simplest path; see /auth/ for the keyring
```

If your console has a self-signed certificate (most local consoles do), add `--insecure` or set
`UNIFI_INSECURE=1`. ufi warns loudly every invocation when TLS verification is disabled — this
is intentional and cannot be silenced without removing the flag.

```bash
export UNIFI_INSECURE=1   # skip TLS verification — warns on every call
```

Alternatively, store the key in the OS keyring so it persists across shells:

```bash
printf '%s' "$UNIFI_API_KEY" | ufi auth login
```

`ufi auth login` reads the key from stdin (never from argv, which leaks to `ps` and shell
history), validates it against the console, and stores it in the OS keyring with a `0600` file
fallback on headless hosts.

### Run diagnostics

Before issuing any reads, confirm that ufi can reach the console:

```bash
ufi doctor --json
```

`doctor` checks the host configuration, key presence, connectivity, and credential file
permissions, and prints a `fix` for each failing check:

```json
{
  "ok": true,
  "checks": [
    { "name": "host",         "ok": true, "detail": "https://192.168.1.1" },
    { "name": "api_key",      "ok": true, "detail": "present (redacted), source=keyring" },
    { "name": "connectivity", "ok": true, "detail": "console reachable, key valid, version 10.4.57" }
  ]
}
```

If any check fails, the exit code is `10` (`config_error`) and each failing item includes a
`fix` field.

## Your first reads

### List adopted devices

```bash
ufi device list --json --select id,name,model,state --limit 5
```

All list commands return the stable list envelope: `{ schemaVersion, items, count, nextCursor }`.
Fields are snake_cased from the API's camelCase (`macAddress` → `mac_address`,
`uptimeSec` → `uptime_sec`):

```json
{
  "schemaVersion": 1,
  "items": [
    { "id": "a1b2c3d4e5f6", "model": "USW-Pro-24-POE", "name": "[UNTRUSTED_DATA_BEGIN] main-switch [UNTRUSTED_DATA_END]", "state": "ONLINE" },
    { "id": "b2c3d4e5f6a1", "model": "U6-LR",          "name": "[UNTRUSTED_DATA_BEGIN] ap-upstairs [UNTRUSTED_DATA_END]",  "state": "ONLINE" }
  ],
  "count": 2,
  "nextCursor": null
}
```

The `[UNTRUSTED_DATA_BEGIN] … [UNTRUSTED_DATA_END]` fencing around device names is automatic in
agent mode (JSON output or non-TTY stdout). Device names are network-controlled free text; a
guest could set one to `Ignore previous instructions…`. The fencing prevents a downstream LLM
from treating it as a prompt. Pass `--no-fence` to disable it, or `--wrap-untrusted` to force
it on even in human-readable output. See [Agents](/agents/) for details.

#### Pagination

`--limit` defaults to 50. When more results exist, `nextCursor` is an opaque base64 string.
Pass it back with `--cursor` to retrieve the next page:

```bash
ufi device list --json --limit 2
# nextCursor: "eyJvZmZzZXQiOjJ9"

ufi device list --json --limit 2 --cursor eyJvZmZzZXQiOjJ9
```

Or use 1-based `--page` as a shorthand:

```bash
ufi device list --json --limit 2 --page 2
```

An empty result (no items matching the query) emits the envelope with `"count": 0` and exits
with code `3` (`empty_results`). A script can branch on the exit code without parsing JSON.

#### Field projection

`--select` is a comma-separated list of dot-path field names. Projection happens client-side
after the API response is received; the API has no projection of its own.

```bash
ufi device list --json --select id,name,state
```

### List connected clients

```bash
ufi client list --json --limit 20
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "c3d4e5f6a1b2",
      "mac_address": "aa:bb:cc:dd:ee:ff",
      "hostname": "[UNTRUSTED_DATA_BEGIN] ryans-laptop [UNTRUSTED_DATA_END]",
      "ip_address": "192.168.10.42",
      "uptime_sec": 3612,
      "type": "WIRED"
    }
  ],
  "count": 1,
  "nextCursor": null
}
```

### Check auth status

```bash
ufi auth status --json
```

Returns which credentials are stored and whether they pass a live validation against the console.
Exit code is `0` even when the console is unreachable (`valid: false` with a `reason` field) —
so an agent does not treat an unreachable console as a retryable command failure. Exit code `4`
(`auth_required`) is only returned when no key is configured at all.

## Your first mutation

Mutations are opt-in. Running a state-changing command without `--allow-mutations` returns a
structured error — exit `12` (`mutation_blocked`) — not a generic failure. An agent can
distinguish "ask for permission" from "real error" by checking the exit code:

```bash
ufi device restart a1b2c3d4e5f6
# stderr → {"error": "device restart is a mutating operation and is blocked by default",
#            "code":  "MUTATION_BLOCKED",
#            "remediation": "re-run with --allow-mutations (add --dry-run to preview)"}
# exit 12
```

Preview the action first with `--dry-run`:

```bash
ufi device restart a1b2c3d4e5f6 --allow-mutations --dry-run
```

```json
{
  "action": "RESTART",
  "id": "a1b2c3d4e5f6",
  "dry_run": true
}
```

When the preview looks right, drop `--dry-run` to execute:

```bash
ufi device restart a1b2c3d4e5f6 --allow-mutations
```

```json
{
  "ok": true,
  "action": "RESTART",
  "id": "a1b2c3d4e5f6"
}
```

Other low-stakes single-target actions follow the same pattern:

```bash
# Power-cycle port 5 on a PoE switch
ufi device port-cycle a1b2c3d4e5f6 5 --allow-mutations --dry-run

# Authorize a guest client for 60 minutes with a 500 MB data cap
ufi client authorize c3d4e5f6a1b2 --minutes 60 --data-mb 500 --allow-mutations --dry-run

# Revoke a guest client's access
ufi client unauthorize c3d4e5f6a1b2 --allow-mutations
```

### High-stakes config: the plan + hash → apply flow

Declarative config commands (`network`, `firewall policy|zone`, `acl`, `dns policy`,
`traffic-list`) never execute immediately, even with `--allow-mutations`. Instead they compute a
plan, stamp a 12-hex content hash, and write the plan file to
`$XDG_STATE_HOME/ufi/plans/<hash>.json`. You then run `ufi apply <hash> --allow-mutations` to
execute exactly that plan — no re-computation, no race between preview and execution:

```bash
# 1. Preview the change (requires --allow-mutations to compute + persist the plan)
ufi network update <network-id> --data @net.json --allow-mutations
```

```json
{
  "action": "network update",
  "method": "PUT",
  "path": "networks/<network-id>",
  "hash": "a3f9c21b04e7",
  "plan": { "id": "<network-id>", "body": { "name": "IoT", "vlanId": 20 } },
  "dry_run": true,
  "note": "preview only — run `ufi apply a3f9c21b04e7 --allow-mutations` to execute"
}
```

```bash
# 2. Execute the exact previewed plan by its hash
ufi apply a3f9c21b04e7 --allow-mutations
```

The `--data` flag accepts a file path (with or without a leading `@`), `-` for stdin, or inline
JSON:

```bash
ufi network create --data @net.json --allow-mutations
ufi network create --data '{"name":"IoT","vlanId":20}' --allow-mutations
cat net.json | ufi network create --data - --allow-mutations
```

See [Declarative config](/config/) for the full reviewed-artifact flow and all config nouns.

## Structured errors

Every error goes to stderr as `{ error, code, remediation }`. The process exits with the
matching code from the stable table. A script never needs to parse error text:

```json
{
  "error": "no UniFi API key configured",
  "code": "AUTH_REQUIRED",
  "remediation": "run `ufi auth login`, or set UNIFI_API_KEY and --host/UNIFI_HOST"
}
```

Key exit codes at a glance:

| Code | Meaning |
|------|---------|
| `0`  | OK |
| `3`  | Empty result set (envelope still emitted) |
| `4`  | Auth required — no key configured |
| `5`  | Resource not found |
| `11` | Unsupported — needs a console feature (e.g. Zone-Based Firewall not enabled) |
| `12` | Mutation blocked — rerun with `--allow-mutations` |

The full table is in `ufi schema` → `exit_codes`, or at [Exit codes](/exit-codes/).

## Non-interactive use

When running in a pipeline or an agent that must never stall on a prompt, pass `--no-input`.
Commands that would otherwise prompt fail immediately with exit `13` (`input_required`) instead:

```bash
ufi auth login --no-input   # exit 13 — pipe the key on stdin instead
```

## Check for updates

ufi never auto-updates. On the human path it prints a once-a-day upgrade hint to stderr (silent
when output is JSON or non-TTY). To check explicitly:

```bash
ufi version --check --json
```

```json
{
  "current": "0.3.1",
  "latest": "0.4.0",
  "updateAvailable": true,
  "upgrade": "go install github.com/rnwolfe/ufi/cmd/ufi@latest"
}
```

Set `UFI_NO_UPDATE_CHECK=1` to silence the hint entirely.

## What's next

- [Installation](/installation/) — all install methods, shell completions, verifying checksums.
- [Authentication](/auth/) — key precedence (env → keyring → file), `auth login/status/logout`, `doctor`.
- [Safety model](/safety-model/) — read-only default, `--allow-mutations`, `--dry-run`, plan + hash flow, prompt-injection fencing.
- [Command reference](/commands/) — every command, flag, and subcommand.
- [Flags & environment](/flags-env/) — global flags and all `UNIFI_*` variables.
- [Exit codes](/exit-codes/) — the full stable table.
- [For agents](/agents/) — `ufi agent`, `ufi schema`, fencing, `--no-input`, and the Agent CLI Guidelines conformance block.
