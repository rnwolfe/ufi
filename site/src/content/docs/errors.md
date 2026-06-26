---
title: Errors & troubleshooting
description: Understand the structured error shape, how upstream UniFi HTTP errors map to exit codes, and how to fix the most common problems using ufi doctor.
---

Every error `ufi` emits follows the same contract: a structured JSON object on **stderr**,
a human-readable message on the same stream, and a stable **exit code** that a script or
agent can branch on without parsing text. This page explains the shape, the classification
rules, and the most common situations you will hit.

See [Exit codes](/exit-codes/) for the full table. See [Output format](/output/) for how
stdout and stderr are separated.

## The structured error shape

When `ufi` encounters an error it prints a single JSON object to **stderr**:

```json
{
  "error": "device abc123 not found",
  "code": "NOT_FOUND",
  "remediation": "list available devices to find a valid id"
}
```

| Field | Type | Description |
|---|---|---|
| `error` | string | Human-readable message. Never use this for branching — it can change. |
| `code` | string | Machine-readable symbolic name. **Stable.** Branch on this. |
| `remediation` | string | Concrete next step. May be empty for internal errors. |

The process also exits with the mapped exit code (e.g. `NOT_FOUND` → exit 5). All codes
are listed in [Exit codes](/exit-codes/).

### Reading errors in a pipeline

```bash
# Capture stderr separately and parse the code field:
result=$(ufi device list 2>/tmp/ufi-err.json)
code=$(jq -r '.code // empty' /tmp/ufi-err.json)

case "$code" in
  AUTH_REQUIRED) echo "no key configured" ;;
  RATE_LIMITED)  echo "back off and retry" ;;
  *) echo "other: $code" ;;
esac
```

Agents can rely on the exit code alone if they don't need the code string:

```bash
ufi client list --site "Home" || exit_code=$?
# exit_code=3 means empty results, not a failure
```

---

## How upstream UniFi errors are classified

`ufi` talks to the UniFi Integration API at
`https://{host}/proxy/network/integration/v1`. The API returns standard HTTP status codes
and, for non-2xx responses, an error envelope:

```json
{
  "statusCode": 404,
  "statusName": "NOT_FOUND",
  "code": "api.device.not-found",
  "message": "Device not found",
  "requestId": "..."
}
```

`ufi` classifies the HTTP status into a `CLIError` according to this table:

| HTTP status | `code` | Exit | Notes |
|---|---|---|---|
| 401 Unauthorized | `AUTH_REQUIRED` | 4 | Bad or missing API key. |
| 403 Forbidden | `PERMISSION_DENIED` | 6 | Key present but lacks scope. |
| 404 Not Found | `NOT_FOUND` | 5 | Resource id does not exist. |
| 429 Too Many Requests | `RATE_LIMITED` | 7 | Includes `Retry-After` seconds in remediation when the header is present. |
| 400 Bad Request (ZBF) | `UNSUPPORTED` | 11 | Only for `api.firewall.zone-based-firewall-not-configured` — see [Zone-Based Firewall](#zone-based-firewall-not-enabled). |
| 400 Bad Request (other) | `BAD_REQUEST` | 2 | Upstream code surfaced in remediation. |
| 5xx Server Error | `TRANSIENT` | 8 | Idempotent GETs are retried up to 3 times with exponential back-off before failing. |
| Network failure | `TRANSIENT` | 8 | Console unreachable; GETs retried up to 3 times. |

For `RATE_LIMITED`, the remediation message includes the exact retry window when the API
provides a `Retry-After` response header:

```json
{
  "error": "rate limit exceeded",
  "code": "RATE_LIMITED",
  "remediation": "rate limited; retry after 30s"
}
```

---

## Common situations and fixes

### No host configured

**Symptom** — `ufi` exits 10 with:

```json
{
  "error": "no UniFi host configured",
  "code": "CONFIG",
  "remediation": "set --host or UNIFI_HOST (e.g. https://192.168.1.1)"
}
```

**Fix** — Set the host in the environment or pass it on every command:

```bash
export UNIFI_HOST=https://192.168.1.1
# or per-command:
ufi --host https://192.168.1.1 device list
```

`ufi auth login` also requires a host to be set before it will store or validate a key.

---

### No API key configured

**Symptom** — `ufi` exits 4 with:

```json
{
  "error": "no UniFi API key configured",
  "code": "AUTH_REQUIRED",
  "remediation": "run `ufi auth login`, or set UNIFI_API_KEY and --host/UNIFI_HOST"
}
```

**Fix** — Store a key via `ufi auth login` (recommended) or set the environment variable:

```bash
# Recommended: store in keyring / 0600 credential file
printf %s "$MY_KEY" | ufi auth login

# Or for one-off use / CI:
export UNIFI_API_KEY=your-key-here
```

The key is read from (in precedence order): `UNIFI_API_KEY` env var → OS keyring →
`$XDG_CONFIG_HOME/ufi/credentials`. See [Authentication](/auth/) for the full precedence
chain and storage details.

---

### Bad or expired key (401)

**Symptom** — exit 4, `AUTH_REQUIRED`:

```json
{
  "error": "Unauthorized",
  "code": "AUTH_REQUIRED",
  "remediation": "run `ufi auth login` or set UNIFI_API_KEY"
}
```

UniFi API keys don't expire on a schedule, but they can be revoked from the console UI
(**Settings → Control Plane → Integrations**). If you recently regenerated the key there,
update your stored credentials:

```bash
printf %s "$NEW_KEY" | ufi auth login
```

`ufi auth logout` removes only the locally-stored credentials — the key remains active on
the console until you revoke it there.

---

### Self-signed TLS certificate

**Symptom** — The console ships a self-signed certificate. Without `--insecure`, `ufi`
exits 8 (`TRANSIENT`) with a message like:

```json
{
  "error": "tls: failed to verify certificate: ...",
  "code": "TRANSIENT",
  "remediation": "the console was unreachable; check --host/network and retry"
}
```

**Fix** — Pass `--insecure` or set `UNIFI_INSECURE=1`:

```bash
ufi --insecure device list
# or permanently via env:
export UNIFI_INSECURE=1
```

:::caution[Security warning]
`--insecure` disables TLS verification and **warns loudly on every invocation**. It is
appropriate for a self-hosted console on a trusted LAN, but you should not use it over
untrusted networks. Consider installing a valid certificate on your console instead.
:::

---

### Multiple sites — `--site` required

**Symptom** — When your console has more than one site and `--site` is not set, `ufi` exits 2:

```json
{
  "error": "multiple sites; specify --site",
  "code": "USAGE",
  "remediation": "choices: Default, Office, Warehouse"
}
```

**Fix** — Pass `--site` with the site name, id, or internal reference:

```bash
ufi --site Default device list
ufi --site "abc123def456" client list
```

You can also set it permanently with `UNIFI_SITE`. When the console has exactly one site,
`ufi` uses it automatically. Run `ufi site list` to see all available sites and their ids.

---

### Zone-Based Firewall not enabled

**Symptom** — `firewall policy` and `firewall zone` commands exit 11 (`UNSUPPORTED`):

```json
{
  "error": "zone-based firewall is not configured",
  "code": "UNSUPPORTED",
  "remediation": "enable Zone-Based Firewall on the console (Settings → Security) to use firewall commands"
}
```

The UniFi API returns HTTP 400 with code `api.firewall.zone-based-firewall-not-configured`
when ZBF is disabled. `ufi` maps this specific 400 to `UNSUPPORTED` (exit 11) rather than
`BAD_REQUEST` (exit 2) so agents can distinguish "feature not available" from "bad input".

**Fix** — Enable Zone-Based Firewall on the console: **Settings → Security → Zone-Based
Firewall → Enable**. Once enabled, `firewall policy list` and `firewall zone list` will work.

---

### Mutation blocked (read-only mode)

**Symptom** — Any state-changing command exits 12 when `--allow-mutations` is absent:

```json
{
  "error": "RESTART is a mutating operation and is blocked by default",
  "code": "MUTATION_BLOCKED",
  "remediation": "re-run with --allow-mutations (add --dry-run to preview)"
}
```

**Fix** — Add `--allow-mutations` (alias `--write`). Use `--dry-run` first to see what
would happen:

```bash
# Preview the restart:
ufi device restart abc123 --allow-mutations --dry-run

# Execute:
ufi device restart abc123 --allow-mutations
```

`ufi` is read-only by default to limit the blast radius of automation errors. See the
[safety model](/safety-model/) for the full gate and the reviewed-artifact apply flow for
declarative config changes.

---

### Rate limiting

**Symptom** — Exit 7, `RATE_LIMITED`. The remediation includes the back-off window when the
API provides one:

```json
{
  "error": "rate limit exceeded",
  "code": "RATE_LIMITED",
  "remediation": "rate limited; retry after 30s"
}
```

**Fix** — Wait the indicated number of seconds and retry. For agent pipelines, treat exit 7
as a signal to pause and retry rather than a hard failure:

```bash
while true; do
  ufi device list && break
  [ $? -eq 7 ] || exit 1   # only retry on RATE_LIMITED
  sleep 30
done
```

---

### Transient / unreachable console

**Symptom** — Exit 8, `TRANSIENT`. This covers network failures and 5xx responses from the
console:

```json
{
  "error": "dial tcp 192.168.1.1:443: connect: connection refused",
  "code": "TRANSIENT",
  "remediation": "the console was unreachable; check --host/network and retry"
}
```

`ufi` already retries idempotent GET requests up to 3 times with exponential back-off
(150 ms, 600 ms, 1350 ms) before surfacing the error. Mutating requests are **not** retried
automatically to avoid double-executing state changes.

**Fix** — Verify the console is reachable and the `--host` / `UNIFI_HOST` value is correct,
then retry. If you see this during normal operation, check your network path to the console.

---

### Cloud commands (deferred)

**Symptom** — `ufi cloud …` exits 11 (`UNSUPPORTED`):

```json
{
  "error": "the Site Manager (cloud) surface is not available in this build — ufi is local-only for now",
  "code": "UNSUPPORTED",
  "remediation": "if you want cloud (api.ui.com) support, please open an issue: https://github.com/rnwolfe/ufi/issues"
}
```

`ufi` targets the local Integration API only. The cloud (Site Manager) surface is a hidden
stub that returns a structured error. Track progress at the linked issue.

---

## Diagnose with `ufi doctor`

`ufi doctor` runs a set of preflight checks and returns a structured report. Run it first
when something is not working:

```bash
ufi doctor
```

Example output when everything is healthy:

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

Example output when checks fail (exit 10, `DOCTOR_FAILED`):

```json
{
  "ok": false,
  "checks": [
    { "name": "host",         "ok": false, "detail": "not set",
      "fix": "pass --host or set UNIFI_HOST (e.g. https://192.168.1.1)" },
    { "name": "api_key",      "ok": false, "detail": "not set",
      "fix": "pipe a key to `ufi auth login` or set UNIFI_API_KEY" },
    { "name": "connectivity", "ok": false, "detail": "skipped — host/key missing",
      "fix": "configure host + key first" }
  ]
}
```

Each failing check includes a `fix` field with the concrete remediation step. `ufi doctor`
exits 0 when all checks pass, and exits 10 (`config_error`) when any check fails.

The connectivity check performs a live authenticated `GET /info` request against the
console. If the credential file has insecure permissions (group- or other-readable), a
`cred_perms` check is added automatically with the path and a `chmod 600` fix.

---

## Schema-drift errors

If the UniFi API returns a response that `ufi` cannot parse, you will see `SCHEMA_DRIFT`
(exit 1):

```json
{
  "error": "could not parse response: ...",
  "code": "SCHEMA_DRIFT",
  "remediation": ""
}
```

This typically means the UniFi Network application was updated and the API changed in a way
`ufi` doesn't expect yet. [Open an issue](https://github.com/rnwolfe/ufi/issues) with the
`ufi version` output and the command that failed.

---

## Quick-reference: codes and exits

| Code | Exit | Typical cause |
|---|---|---|
| `AUTH_REQUIRED` | 4 | No key, or 401 from the API. |
| `PERMISSION_DENIED` | 6 | Key present, 403 from the API. |
| `NOT_FOUND` | 5 | Resource id does not exist (404). |
| `RATE_LIMITED` | 7 | 429 from the API. |
| `TRANSIENT` | 8 | Network failure or 5xx; already retried on GETs. |
| `MUTATION_BLOCKED` | 12 | State-changing op without `--allow-mutations`. |
| `UNSUPPORTED` | 11 | ZBF not enabled, or cloud command attempted. |
| `BAD_REQUEST` | 2 | Bad arguments or other 400 from the API. |
| `CONFIG` | 10 | Missing or invalid host / credential storage failure. |
| `INPUT_REQUIRED` | 13 | `--no-input` set but a prompt was needed. |
| `DOCTOR_FAILED` | 10 | One or more `ufi doctor` checks failed. |
| `SCHEMA_DRIFT` | 1 | API response could not be parsed. |
| `TRANSIENT` | 8 | Console unreachable or 5xx server error. |

The full authoritative table is in [Exit codes](/exit-codes/) and is also machine-readable
via `ufi schema` (the `exit_codes` field).
