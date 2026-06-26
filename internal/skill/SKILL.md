---
name: ufi
description: Drive ufi, an agent-friendly CLI for Ubiquiti UniFi Network via the official local Integration API. Read-only by default; mutations require --allow-mutations.
---

# ufi

An agent-focused CLI for **Ubiquiti UniFi Network**, built on Ubiquiti's **official local
Integration API** only (no reverse-engineered/legacy controller API). Safe to explore:
**read-only by default**, never prompts, structured output.

Talks to your console at `--host` / `UNIFI_HOST` (e.g. `https://192.168.1.1`) with an
`X-API-KEY`. (The Site Manager *cloud* surface is not shipped in this build; `ufi cloud …`
returns a pointer to open an issue.)

## First moves
- `ufi schema` — machine-readable command tree, exit codes, conformance, and live safety state.
- `ufi --help` — example-led help.
- `ufi auth status --json` — which credentials are configured.
- `ufi doctor --json` — host/key/connectivity checks.
- `ufi version --check --json` — `{current, latest, updateAvailable, upgrade}` (fail-silent). Never self-updates.

## Auth (API key only)
`X-API-KEY` — no OAuth, no session, no refresh. Generate the key once in the console UI
(Settings → Control Plane → Integrations → API), then provide it via env (never argv):
- `UNIFI_HOST` (or `--host`), `UNIFI_API_KEY` for the local API.
- `UNIFI_INSECURE=1` (or `--insecure`) to accept the console's self-signed cert (warns loudly).
A UniFi API key is effectively **full admin** — treat it like a root password.

## Output
- `--format json` (or `--json`) for structured output; `--format tsv` for columns.
- Lists return an envelope `{ schemaVersion, items, count, nextCursor }`; page with `--cursor`.
- `--select id,name` projects fields; `--limit N` bounds list size (default 50).
- Data → stdout; notes/warnings/errors → stderr. Free text controlled by network devices/clients
  (device/client/SSID/voucher names, notes) is fenced `[UNTRUSTED_DATA_BEGIN] … [UNTRUSTED_DATA_END]`
  in agent mode; disable with `--no-fence`.

## Reading
- `ufi info` · `ufi site list`
- `ufi device list` · `ufi device get <id>` · `ufi device stats <id>`
- `ufi client list` · `ufi client get <id>`
- `ufi wifi list` · `ufi wifi get <id>` · `ufi voucher list`
- `ufi network list|get` · `ufi firewall policy list` · `ufi firewall zone list`
  · `ufi acl list` · `ufi dns policy list` · `ufi traffic-list list`

## Mutating (gated)
Mutations are blocked unless you pass `--allow-mutations`. A blocked mutation returns exit
code **12** and `{"code":"MUTATION_BLOCKED"}`. Preview first with `--dry-run`.

Low-stakes single-target actions:
- `ufi device restart <id> --allow-mutations`
- `ufi device port-cycle <id> <port> --allow-mutations`
- `ufi client authorize <id> --minutes 60 --allow-mutations` · `ufi client unauthorize <id> --allow-mutations`
- `ufi voucher create "guest-pass" --minutes 1440 --count 5 --allow-mutations` (name + --minutes required) · `ufi voucher delete <id> --allow-mutations` (idempotent)

High-stakes **declarative config** uses a reviewed-artifact flow (preview → apply by hash):
- `ufi network create --data @net.json --allow-mutations` → prints a `plan` + `hash`.
- `ufi apply <hash> --allow-mutations` → executes exactly that previewed plan.
- Same for `firewall policy|zone`, `acl`, `dns policy`, `traffic-list` create/update/delete/reorder.
- Firewall commands require **Zone-Based Firewall** enabled on the console (otherwise → `unsupported`).

## Errors & exit codes
Structured `{error, code, remediation}` on stderr. Key codes: 0 ok, 2 usage, 3 empty,
4 auth_required, 5 not_found, 6 permission, 7 rate_limited, 8 retryable, 10 config,
**11 unsupported** (needs a console feature/legacy API), 12 mutation_blocked, 13 input_required.
Full table: `ufi schema`.

## Non-interactive use
Pass `--no-input` to guarantee the tool never prompts (it fails with exit 13 instead).
