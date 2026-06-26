---
title: Exit codes
description: The full table of ufi exit codes, what each one means, and how to branch on them in scripts and agent loops.
---

Every ufi process exits with a number that tells you — and your scripts — exactly what happened,
without parsing a message string. The codes are a stable, append-only contract: new codes can be
added but existing ones never change meaning or value.

Get the live table at any time (always current with the binary you have installed):

```bash
ufi schema | jq .exit_codes
```

```json
{
  "ok": 0,
  "generic_error": 1,
  "usage": 2,
  "empty_results": 3,
  "auth_required": 4,
  "not_found": 5,
  "permission": 6,
  "rate_limited": 7,
  "retryable": 8,
  "config_error": 10,
  "unsupported": 11,
  "mutation_blocked": 12,
  "input_required": 13,
  "cancelled": 130
}
```

## The table

| Code | Machine string | Meaning |
|------|----------------|---------|
| 0 | `ok` | Command completed successfully. |
| 1 | `generic_error` | Unexpected error with no more specific code. |
| 2 | `usage` | Bad arguments, unknown flags, or malformed input. |
| 3 | `empty_results` | Query succeeded but returned zero items. |
| 4 | `auth_required` | No API key is configured, or the key was rejected. |
| 5 | `not_found` | A named resource does not exist on the console. |
| 6 | `permission` | API key is valid but lacks the required scope. |
| 7 | `rate_limited` | The console is throttling requests. |
| 8 | `retryable` | Transient network or server error; safe to retry. |
| 10 | `config_error` | Local ufi configuration is broken or unwritable. |
| 11 | `unsupported` | Feature not available on this console or surface. |
| 12 | `mutation_blocked` | State-changing op blocked because `--allow-mutations` was not passed. |
| 13 | `input_required` | A required value is missing and `--no-input` prevents prompting. |
| 130 | `cancelled` | Process received SIGINT (Ctrl-C). |

:::note[Exit code 9 is intentionally absent]
There is no code 9. The set is append-only — gaps let future codes be added without renumbering.
:::

## Each code in detail

### 0 — ok

The command ran to completion and produced its output. For list commands this includes successful
pagination; for mutations it means the change was applied (or the dry-run preview was printed).

```bash
ufi device list
echo $?   # 0
```

### 1 — generic_error

An unexpected internal error that does not fit any specific code. The structured error on stderr
will include `"code": "INTERNAL"` or `"API_ERROR"` along with details. If you see this
consistently, [open an issue](https://github.com/rnwolfe/ufi/issues).

### 2 — usage

The invocation itself was wrong: missing a required argument, an unrecognised flag, an `--cursor`
that cannot be decoded, invalid `--data` JSON, or `ufi apply` given an unknown hash. The
remediation in the error object points at what to fix.

```bash
ufi device list --cursor not-base64
```

```json
{
  "error": "invalid --cursor",
  "code": "USAGE",
  "remediation": "use the nextCursor from a prior response"
}
```

Exit code: **2**

### 3 — empty_results

The query reached the console, authenticated, and paged successfully — but the result set is
empty. The full list envelope is still emitted on stdout so your pipeline does not have to handle
a missing document:

```bash
ufi voucher list
```

```json
{
  "schemaVersion": 1,
  "items": [],
  "count": 0,
  "nextCursor": null
}
```

Exit code: **3** — check `$?` before processing `items`, not the other way around. See
[Output](/output/) for the envelope shape.

### 4 — auth_required

No API key is configured, or the key supplied was rejected by the console (HTTP 401). `ufi auth
status` exits **0** even when the key is invalid (it folds validity into the response object);
exit 4 only fires when there is no key at all. To fix:

```bash
printf '%s' 'YOUR_KEY_HERE' | ufi auth login
# or set the environment variable:
export UNIFI_API_KEY=YOUR_KEY_HERE
```

See [Auth](/auth/) for the full storage and precedence rules.

### 5 — not_found

A resource id or reference you supplied does not exist on the console. Run the corresponding list
command to find valid ids:

```bash
ufi device list | jq '.items[].id'
```

`voucher delete` is the notable exception: a missing voucher is treated as a **soft success**
(exit 0) because deletes are idempotent by design. See [Safety model](/safety-model/).

### 6 — permission

The API key is valid and authenticated, but the console refused the operation because the key
lacks the required scope (HTTP 403). A UniFi API key is effectively full-admin, so this usually
means the key was scoped at creation time to a subset of endpoints. Generate a new, unrestricted
key on the console and re-run `ufi auth login`.

### 7 — rate_limited

The console returned HTTP 429. The structured error includes the remediation from any
`Retry-After` header the console sent:

```json
{
  "error": "too many requests",
  "code": "RATE_LIMITED",
  "remediation": "rate limited; retry after 30s"
}
```

Back off for at least the indicated duration before retrying. In an agent loop, treat this as a
signal to pause — not to abandon the task.

### 8 — retryable

A transient failure: the console was unreachable, the connection was reset, or the console
returned a 5xx. ufi already retried GET requests up to three times (with a short exponential
back-off) before surfacing this code. For non-GET requests (mutations), it surfaces immediately
because re-sending a mutation blindly is unsafe. Safe to retry the full command once the console
is reachable.

```json
{
  "error": "dial tcp 192.168.1.1:443: connect: connection refused",
  "code": "TRANSIENT",
  "remediation": "the console was unreachable; check --host/network and retry"
}
```

### 10 — config_error

Something is wrong with ufi's local configuration — the host is not set, the credential store is
not writable, or a plan file could not be saved. This is a local problem, not a network or API
problem. Common cases:

- `--host` / `UNIFI_HOST` is missing when running `ufi auth login`.
- `$XDG_STATE_HOME/ufi/plans/` is not writable (plan save fails for config write commands).
- The credential file could not be written during `auth login` or cleared during `auth logout`.

Fix the underlying configuration, then retry.

### 11 — unsupported

The operation is not available on this console, site, or surface. Two common triggers:

**Zone-Based Firewall not enabled.** `firewall policy` and `firewall zone` commands require ZBF
to be configured on the console. When it is not, the console returns a 400 with code
`api.firewall.zone-based-firewall-not-configured` and ufi maps it to:

```json
{
  "error": "zone-based firewall is not configured",
  "code": "UNSUPPORTED",
  "remediation": "enable Zone-Based Firewall on the console (Settings → Security) to use firewall commands"
}
```

**Cloud surface.** `ufi cloud …` is a hidden stub. It always exits 11 with an `UNSUPPORTED`
error pointing to the [issue tracker](https://github.com/rnwolfe/ufi/issues). ufi is local-only;
the cloud surface is deferred. Do not attempt to work around this — there is no hidden path.

### 12 — mutation_blocked

The command would change state, and `--allow-mutations` was not passed. This is the read-only
safety gate. It is intentional — not a bug to retry or suppress:

```bash
ufi device restart abc123
```

```json
{
  "error": "device restart is a mutating operation and is blocked by default",
  "code": "MUTATION_BLOCKED",
  "remediation": "re-run with --allow-mutations (add --dry-run to preview)"
}
```

In an agent loop, treat exit 12 as a signal to ask the human for explicit permission rather than
retrying automatically. See [Safety model](/safety-model/) for the full mutation gate design.

### 13 — input_required

`--no-input` was set (guaranteeing ufi never blocks on a TTY read), but a required value was not
supplied via flags or arguments. The remediation tells you exactly what is missing:

```json
{
  "error": "API key on stdin is required",
  "code": "INPUT_REQUIRED",
  "remediation": "pass it as a flag/argument (running with --no-input, so prompts are disabled)"
}
```

Always pass `--no-input` in agent and CI contexts, and supply all required values explicitly.

### 130 — cancelled

The process received SIGINT (Ctrl-C) and exited cleanly. No partial mutation is committed (ufi
does not buffer partial writes). This mirrors the POSIX convention for interrupted processes
(`128 + signal 2`).

## Branching in shell scripts

Exit codes are most useful when you branch on them without parsing message strings:

```bash
#!/usr/bin/env bash
set -euo pipefail

result=$(ufi device list --json 2>/tmp/ufi-err)
code=$?

case $code in
  0)  echo "devices: $(echo "$result" | jq '.count')" ;;
  3)  echo "no devices found" ;;
  4)  echo "not authenticated — run: ufi auth login" ;;
  7)  echo "rate limited — backing off" ; sleep 30 ;;
  8)  echo "console unreachable — retrying later" ;;
  12) echo "mutation blocked — human approval required" ;;
  *)  echo "unexpected error $code:" ; cat /tmp/ufi-err ; exit $code ;;
esac
```

## Branching in an agent loop

An agent should branch on the numeric exit code, not the human-readable message. Key decision
points:

| Exit code | Recommended agent action |
|-----------|--------------------------|
| 0 | Proceed. Parse stdout. |
| 3 | The list is empty — this is expected, not an error. Proceed with empty-list logic. |
| 4 | Surface to the human: credentials must be configured before continuing. |
| 5 | The id does not exist. Offer to list valid ids and let the human choose. |
| 7 | Pause for the Retry-After duration, then retry the same command. |
| 8 | Console is temporarily unreachable. Retry after a short delay. |
| 11 | Surface to the human: a prerequisite (e.g. ZBF) needs to be enabled on the console. |
| 12 | **Do not retry.** Ask the human for explicit permission to pass `--allow-mutations`. |
| 13 | A required value is missing. Ask the human to supply it. |
| 130 | The human cancelled. Stop the task. |
| 1, 6, 10 | Parse `stderr` for the structured error and surface the remediation to the human. |

For the full structured-error shape (`{error, code, remediation}`), see [Errors](/errors/). For
how `ufi schema` returns the live exit-code table alongside the command tree and safety state, see
[For agents](/agents/).
