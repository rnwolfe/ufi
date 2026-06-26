---
title: Flags & environment variables
description: Complete reference for every global flag and environment variable that ufi accepts.
---

Every flag in this table is **global** — it can appear anywhere on the command line and applies to
every subcommand. Most have an environment-variable equivalent you can set once in your shell or
`.env` file instead of repeating it on every invocation.

---

## Global flags

### Output

| Flag | Type | Default | Description |
|---|---|---|---|
| `--format` | `json` \| `plain` \| `tsv` | `plain` | Output format. `json` emits 2-space indented JSON with HTML-escaping disabled. `plain` is human-readable prose. `tsv` is tab-separated for shell pipelines. |
| `--json` | bool | `false` | Shorthand for `--format=json`. Useful for one-off agent calls without setting `--format`. |
| `--no-color` | bool | `false` | Disable ANSI color in plain output. Color is automatically disabled when stdout is not a TTY, when `NO_COLOR` is set, or when `--format` is not `plain`. |
| `--select` | string | _(all fields)_ | Comma-separated dot-path field projection applied **client-side** after the API responds. The API has no server-side projection; this trims the response locally. Example: `id,name,uptime_sec` or `network.vlan_id`. |
| `--limit` | int | `50` | Maximum items returned per list operation. The API is queried with this bound; raise it to fetch more in one call or combine with `--cursor`/`--page` to page through large sets. |
| `--cursor` | string | _(none)_ | Opaque pagination cursor. Copy the `nextCursor` value from a previous list response and pass it here to fetch the next page. Takes priority over `--page` when both are set. |
| `--page` | int | _(none)_ | 1-based page number. An alternative to `--cursor` when you want offset-style navigation. Ignored when `--cursor` is also set. |
| `--concise` | bool | `false` | Accepted for [Agent CLI Guidelines](https://aclig.dev) contract uniformity; currently a no-op (output is already concise by default). |
| `--detailed` | bool | `false` | Accepted for contract uniformity; reserved for richer output in a future release. Currently a no-op. |

**Pagination example** — walk through all devices two pages at a time:

```bash
# Page 1
ufi device list --json --limit 2

# Page 2 — copy nextCursor from the previous response
ufi device list --json --limit 2 --cursor "eyJvZmZzZXQiOjJ9"
```

```json
{
  "schemaVersion": 1,
  "items": [
    { "id": "abc123", "name": "sw-core", "model": "USW-Pro-48" }
  ],
  "count": 1,
  "nextCursor": null
}
```

**Field projection example** — return only `id` and `name`:

```bash
ufi device list --json --select id,name
```

---

### Prompt-injection fencing

These flags control whether network-controlled free text (device names, client hostnames, voucher
names, notes) is wrapped with `[UNTRUSTED_DATA_BEGIN]…[UNTRUSTED_DATA_END]` markers. The markers
exist so that an LLM consuming `ufi` output cannot be manipulated by an attacker who controls
those strings. See [/agents/](/agents/) for a full discussion.

| Flag | Type | Default | Description |
|---|---|---|---|
| `--no-fence` | bool | `false` | Disable untrusted-text fencing. By default, fencing is **on** whenever ufi is in agent mode (JSON output or non-TTY stdout). Use this only when your pipeline handles sanitisation itself. |
| `--wrap-untrusted` | bool | `false` | Force fencing on, even on a TTY in plain-text mode. Useful when you're scripting and piping output to a log or LLM. |

**Agent mode** is detected automatically: fencing is active when `--format=json` (or `--json`) is
set, or when stdout is not a terminal. `--no-fence` and `--wrap-untrusted` are explicit overrides
in opposite directions.

Operator-set fields — site names, network names — are **not** fenced; only fields that a
network-connected device or client can influence are treated as untrusted.

---

### Safety

| Flag | Type | Default | Description |
|---|---|---|---|
| `--allow-mutations` / `--write` | bool | `false` | Permit state-changing operations. Without this flag, every mutating command returns a structured `MUTATION_BLOCKED` error (exit 12). `--write` is an alias for ergonomic use. |
| `--dry-run` | bool | `false` | Preview a mutation without executing it. For simple single-target actions (e.g. `device restart`) it prints what would happen without making any change. Declarative config write commands (`network create`, `firewall policy update`, etc.) always produce a persisted plan at `$XDG_STATE_HOME/ufi/plans/<hash>.json` and always require `--allow-mutations`; `--dry-run` has no distinct effect for those commands. That hash is then passed to `ufi apply <hash> --allow-mutations` to execute the change later. |
| `--no-input` | bool | `false` | Never prompt for input. If a command needs interactive input and this flag is set, it fails immediately with exit 13 (`INPUT_REQUIRED`). Use this in CI/CD or agent loops where you cannot accept a blocking prompt. |

**Typical mutation flow:**

```bash
# 1. Produce the plan — config write commands always persist a plan (--allow-mutations required)
ufi network update net-001 --data ./patch.json --allow-mutations

# Output shows the plan hash, e.g. a1b2c3d4e5f6
# Plan saved to ~/.local/state/ufi/plans/a1b2c3d4e5f6.json

# 2. Review the plan file if desired
# cat ~/.local/state/ufi/plans/a1b2c3d4e5f6.json

# 3. Execute exactly that previewed plan
ufi apply a1b2c3d4e5f6 --allow-mutations
```

The hash is a stable 12-hex SHA-256 digest of the operation, method, API path, and body. If you
preview the same change twice you get the same hash, so `apply` is idempotent for identical
payloads. See [/safety-model/](/safety-model/) for the full reviewed-artifact flow.

**Blocked mutation error (JSON):**

```json
{
  "error": "network update net-001 is a mutating operation and is blocked by default",
  "code": "MUTATION_BLOCKED",
  "remediation": "re-run with --allow-mutations (add --dry-run to preview)"
}
```

---

### Connection

| Flag | Env var | Type | Default | Description |
|---|---|---|---|---|
| `--host` | `UNIFI_HOST` | string | _(none)_ | Base URL or IP of your UniFi console, e.g. `https://192.168.1.1`. Required for every API call; set it once in the env rather than repeating it. |
| `--site` | `UNIFI_SITE` | string | _(auto)_ | Site identifier: matches on id, name, or internal reference. When omitted and the console has exactly one site, that site is used. When omitted and multiple sites exist, ufi errors and asks you to pass `--site`. |
| `--insecure` | `UNIFI_INSECURE` | bool | `false` | Skip TLS certificate verification. UniFi consoles ship a self-signed certificate; this flag is necessary when you have not imported a custom cert. **Warns loudly on every invocation** — it is intentionally not silent. Off by default. |

```bash
# One-time env setup; then ufi works without flags
export UNIFI_HOST=https://192.168.1.1
export UNIFI_API_KEY=your-api-key

ufi device list --json
```

```bash
# Typical self-signed console
ufi doctor --json --insecure
```

---

## Environment variables

All env vars are read at startup. CLI flags take precedence over env vars for `--host`,
`--site`, and `--insecure` (kong folds the env vars into the flags during parse). API keys are
read directly from the environment in `auth.Resolve`, not via kong.

### Credential / connection variables

| Variable | Description |
|---|---|
| `UNIFI_HOST` | Console base URL or IP. Equivalent to `--host`. |
| `UNIFI_API_KEY` | Local API key (sent as `X-API-KEY` header). Takes precedence over keyring and credential file. Treat this like a root password — a UniFi API key grants full-admin access, bypassing per-admin RBAC. |
| `UNIFI_CLOUD_API_KEY` | Site Manager cloud API key. Stored alongside the local key but the cloud surface is currently deferred — `ufi cloud …` returns exit 11. Reserved for future use. |
| `UNIFI_SITE` | Site identifier. Equivalent to `--site`. |
| `UNIFI_INSECURE` | Set to `1` to skip TLS verification. Equivalent to `--insecure`. Warns loudly every invocation. |

**Credential precedence** (highest to lowest):

1. `UNIFI_API_KEY` / `UNIFI_CLOUD_API_KEY` environment variables
2. OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager)
3. 0600 credential file at `$XDG_CONFIG_HOME/ufi/credentials`

The OS keyring is preferred on desktop hosts. On headless agents where no keyring daemon is
available, ufi degrades gracefully to the file — it never blocks on a passphrase prompt. See
[/auth/](/auth/) for the full credential lifecycle.

### Agent / tooling variables

| Variable | Description |
|---|---|
| `UFI_HELP` | When set to `agent`, a help request (`ufi`, `ufi --help`, `ufi -h`) prints the embedded `SKILL.md` machine contract instead of the full kong help text. Useful for bootstrapping an agent that needs to know ufi's capabilities without human-formatted output. |
| `UFI_NO_UPDATE_CHECK` | Set to `1` to silence the human-readable update hint that appears when `ufi version` (without `--check`) detects a newer release is available. Has no effect on `ufi version --check`, which always performs the check explicitly. |
| `UFI_RELEASES_URL` | Override the GitHub Releases API URL used by `ufi version --check`. Must be `https://` (any host) or `http://localhost` / `http://127.0.0.1` / `http://[::1]` — any other scheme or host is silently ignored and the default GitHub URL is used. Intended for test harnesses; not useful in production. |
| `NO_COLOR` | Standard [`NO_COLOR`](https://no-color.org/) convention. When set (any value), disables ANSI color in plain output. Equivalent to `--no-color`. |

### XDG base directory variables

ufi respects the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/).

| Variable | Default (if unset) | Used for |
|---|---|---|
| `XDG_CONFIG_HOME` | `$HOME/.config` | Credential file: `$XDG_CONFIG_HOME/ufi/credentials` (mode 0600). |
| `XDG_STATE_HOME` | `$HOME/.local/state` | Plan files: `$XDG_STATE_HOME/ufi/plans/<hash>.json` (mode 0600, dir 0700). These are the reviewed artifacts produced by `--dry-run` on config commands, consumed by `ufi apply <hash>`. |

**Credential file location:**

```bash
# Default
~/.config/ufi/credentials

# Custom XDG_CONFIG_HOME
XDG_CONFIG_HOME=/opt/ufi-state ufi auth login
# writes to /opt/ufi-state/ufi/credentials
```

**Plan directory location:**

```bash
# Default — populated by config --dry-run, read by ufi apply
ls ~/.local/state/ufi/plans/
# a1b2c3d4e5f6.json
# d7e8f9a0b1c2.json
```

---

## Flag quick-reference

The table below groups all flags by category for fast lookup.

| Flag | Short form | Env var | Category |
|---|---|---|---|
| `--format` | — | — | Output |
| `--json` | — | — | Output (shorthand) |
| `--no-color` | — | `NO_COLOR` | Output |
| `--select` | — | — | Output |
| `--limit` | — | — | Pagination |
| `--cursor` | — | — | Pagination |
| `--page` | — | — | Pagination |
| `--concise` | — | — | Output (contract) |
| `--detailed` | — | — | Output (contract) |
| `--no-fence` | — | — | Fencing |
| `--wrap-untrusted` | — | — | Fencing |
| `--allow-mutations` | `--write` | — | Safety |
| `--dry-run` | — | — | Safety |
| `--no-input` | — | — | Safety |
| `--host` | — | `UNIFI_HOST` | Connection |
| `--site` | — | `UNIFI_SITE` | Connection |
| `--insecure` | — | `UNIFI_INSECURE` | Connection |

---

## Related pages

- [/auth/](/auth/) — storing, validating, and rotating API keys
- [/safety-model/](/safety-model/) — the read-only default, `--allow-mutations`, `--dry-run`, and the reviewed-artifact apply flow in detail
- [/output/](/output/) — the list envelope, field projection, pagination, and output formats
- [/agents/](/agents/) — prompt-injection fencing, `UFI_HELP=agent`, `ufi schema`, and the agent contract
- [/exit-codes/](/exit-codes/) — every exit code and its meaning
- [/errors/](/errors/) — structured error format (`error`, `code`, `remediation`)
- [/config/](/config/) — declarative config commands and the `--dry-run` → `apply` flow
