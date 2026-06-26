---
title: Authentication
description: Generate a UniFi API key, store it securely with ufi auth login, understand credential precedence, and verify your setup with ufi doctor.
---

`ufi` authenticates to your console with a single **`X-API-KEY`** header — no OAuth, no
session cookie, no refresh token. The key is resolved from your environment or local storage
on every invocation and sent directly to the Integration API at
`https://{host}/proxy/network/integration/v1`.

## Generate a key in the UniFi UI

1. Open the UniFi Network application.
2. Go to **Settings → Control Plane → Integrations**.
3. Click **Create API Key**, give it a label, and copy the value.

The UI shows the key **once**. If you close the dialog without copying it, generate a new one.

:::caution[Full-admin access — treat like a root password]
A UniFi Integration API key **bypasses per-admin RBAC**. Anyone who holds the key has full
access to the Network application — equivalent to a local `admin` account. Store it in a
keyring or environment secret, never in plain text committed to a repo.

`ufi` is read-only by default to limit the blast radius of a compromised invocation, but the
key itself carries no scoping. See the [safety model](/safety-model/) for how `--allow-mutations`
and the reviewed-apply flow keep mutations intentional.
:::

## Store the key with `ufi auth login`

The recommended first-time setup:

```bash
# Set the console address (permanent env var or --host on every command):
export UNIFI_HOST=https://192.168.1.1

# Pipe the key in — it is validated before being stored:
printf %s "$KEY" | ufi auth login
```

`auth login` reads the key from **stdin**, trims whitespace, validates it against `GET /info`
on your console, then stores the key and host. On success it emits:

```json
{
  "ok": true,
  "console": "https://192.168.1.1",
  "application_version": "10.4.57",
  "stored": "keyring"
}
```

The `stored` field tells you whether the key landed in the OS keyring (`"keyring"`) or the
fallback credential file (`"file"`). Either is fine; see [Credential storage](#credential-storage)
below.

:::note[Key is never passed as an argument]
`ufi auth login` reads the key exclusively from stdin. Passing it as a flag or positional
argument would expose it to `ps`, `/proc`, shell history, and agent command logs. Always pipe
it in.
:::

### Why validate before storing?

`auth login` hits the live console before saving anything. If the key or host is wrong, you
get a structured error on stderr and nothing is stored — no silent bad-credential state.

## Credential precedence

On every invocation, `ufi` resolves credentials in this order (first match wins):

| Priority | Source | Notes |
|----------|--------|-------|
| 1 | `UNIFI_API_KEY` env var | Overrides any stored key |
| 2 | OS keyring | macOS Keychain / Linux Secret Service / Windows Credential Manager |
| 3 | `$XDG_CONFIG_HOME/ufi/credentials` | `0600` file, JSON; path is `~/.config/ufi/credentials` when `XDG_CONFIG_HOME` is unset |

The host is resolved the same way: `--host` flag or `UNIFI_HOST` env var override whatever
`auth login` stored.

In CI and containers, set `UNIFI_API_KEY` and `UNIFI_HOST` directly — no keyring, no file,
no prior `auth login` step needed:

```bash
export UNIFI_HOST=https://192.168.1.1
export UNIFI_API_KEY=my-key
ufi device list --json
```

## Credential storage

`ufi auth login` writes to the **OS keyring** (macOS Keychain, Linux Secret Service via
libsecret, Windows Credential Manager) when one is available. On headless hosts with no
keyring backend, it falls back to a **`0600` JSON file** at
`$XDG_CONFIG_HOME/ufi/credentials` (typically `~/.config/ufi/credentials`).

The file fallback is intentional — ufi never prompts for a keyring passphrase, which would
deadlock a non-interactive agent. The keyring backends used are OS-native only.

`ufi doctor` and `ufi auth status` warn if the credential file exists with permissions looser
than `0600`:

```text
warning: credential file is group/other readable: /home/you/.config/ufi/credentials
```

Fix it with:

```bash
chmod 600 ~/.config/ufi/credentials
```

## `ufi auth status`

Check whether a key is configured and whether it works against the live console:

```bash
ufi auth status --json
```

Example — key valid:

```json
{
  "console": "https://192.168.1.1",
  "has_local_key": true,
  "has_cloud_key": false,
  "source": "keyring",
  "valid": true,
  "application_version": "10.4.57"
}
```

Example — key stored but console unreachable:

```json
{
  "console": "https://192.168.1.1",
  "has_local_key": true,
  "has_cloud_key": false,
  "source": "file",
  "valid": false,
  "reason": "connection refused"
}
```

Key behaviours to note:

- `valid: false` with a `reason` exits **0** — the key is stored and `auth status` ran
  successfully; the connectivity issue is surfaced as data, not a process failure. This lets
  an agent distinguish "no key at all" from "console temporarily unreachable".
- The stored key is **never printed**. `auth status` only reports whether it is present and
  valid.
- If no key is configured at all, `auth status` returns `AUTH_REQUIRED` (exit 4) with a
  remediation hint — see [AUTH\_REQUIRED](#auth_required-exit-4) below.

## `ufi auth logout`

Remove locally stored credentials:

```bash
ufi auth logout
```

```json
{
  "ok": true,
  "cleared": true
}
```

`auth logout` removes the key from **both** the OS keyring and the credential file. It does
**not** revoke the key on the console — it remains valid until you delete it in
**Settings → Control Plane → Integrations → API Keys**. If you suspect a key has leaked,
revoke it on the console first, then run `auth logout` to clean up locally.

## `ufi auth refresh`

```bash
ufi auth refresh
```

This is a no-op: UniFi Integration API keys do not expire and have no refresh flow.

```json
{
  "ok": true,
  "refreshed": false
}
```

It exists so scripts and agents that call `auth refresh` as a routine step do not need
a special case for `ufi`.

## AUTH\_REQUIRED (exit 4)

Any command that needs the API — including `auth status` when no key is stored — returns a
structured error on stderr and exits 4:

```json
{
  "error": "no UniFi API key configured",
  "code": "AUTH_REQUIRED",
  "remediation": "run `ufi auth login`, or set UNIFI_API_KEY and --host/UNIFI_HOST"
}
```

Exit 4 is distinct and stable, so an agent can branch on it without parsing the message text.
See [Exit codes](/exit-codes/) for the full table.

## Self-signed console certificates

UniFi consoles ship a **self-signed TLS certificate**. By default, `ufi` rejects it and exits
with a TLS verification error. Use `--insecure` or `UNIFI_INSECURE=1` to skip verification:

```bash
# flag (per-command):
ufi device list --insecure --json

# env var (session-wide):
export UNIFI_INSECURE=1
ufi device list --json
```

When `--insecure` is active, `ufi` **warns loudly on every invocation** to stderr:

```text
warning: TLS verification disabled (--insecure / UNIFI_INSECURE); the console's identity is not verified
```

This is intentional: `--insecure` is an escape hatch, not a recommended steady state.
Consider installing your console's certificate in your system trust store instead.

## `ufi doctor`

Run a full pre-flight diagnostic before doing any real work:

```bash
ufi doctor --json
```

Example output (all checks passing):

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

Example output (key missing):

```json
{
  "ok": false,
  "checks": [
    { "name": "host",         "ok": true,  "detail": "https://192.168.1.1" },
    { "name": "api_key",      "ok": false, "detail": "not set",
      "fix": "pipe a key to `ufi auth login` or set UNIFI_API_KEY" },
    { "name": "connectivity", "ok": false, "detail": "skipped — host/key missing",
      "fix": "configure host + key first" }
  ]
}
```

When any check fails, `ufi doctor` exits with `CONFIG_ERROR` (exit 10) and each failing
check carries a `fix` field with the next action. When the credential file has loose
permissions, a `cred_perms` check is added automatically with the `chmod 600` fix.

`ufi doctor` is the fastest path to a working setup — run it whenever you change hosts, rotate
a key, or move to a new machine.

## What to do if a key leaks

1. **Revoke it on the console** (Settings → Control Plane → Integrations → API Keys). Local
   `auth logout` does not revoke the key — only the console can do that.
2. Generate a replacement key.
3. Run `ufi auth logout` to clear the old local copy.
4. Run `printf %s "$NEW_KEY" | ufi auth login` with the new key.
5. Run `ufi doctor --json` to verify the new key works.

Never paste a real key into a GitHub issue. See the
[SECURITY.md](https://github.com/rnwolfe/ufi/blob/main/SECURITY.md) for the private
vulnerability reporting process.

## Related pages

- [Getting started](/getting-started/) — install ufi and run the first command.
- [Safety model](/safety-model/) — how `--allow-mutations`, `--dry-run`, and the
  reviewed-apply flow bound what an agent can do with a valid key.
- [Flags & environment variables](/flags-env/) — `UNIFI_HOST`, `UNIFI_SITE`, `UNIFI_INSECURE`,
  and other env vars.
- [Exit codes](/exit-codes/) — the full stable exit-code table, including exit 4
  (`AUTH_REQUIRED`).
- [For agents](/agents/) — the full agent contract, including credential handling and
  `ufi schema`.
