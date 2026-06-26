---
title: Authentication & security
description: How ufi authenticates to a UniFi console (X-API-KEY), stores the key, the doctor command, and the secret-handling threat model.
---

`ufi` authenticates to your console with a single **`X-API-KEY`** header — no OAuth, no
session, no refresh. The key is read from **stdin** (`ufi auth login`) or the `UNIFI_API_KEY`
env var, **never argv** (argv leaks to `ps` / `/proc` / shell history), and stored in the OS
keyring with a `0600` file fallback on headless hosts.

## Generate a key in the console

In the UniFi UI: **Settings → Control Plane → Integrations → Create API Key**. Copy the key
once — the UI won't show it again.

A UniFi API key is effectively **full admin**: it bypasses per-admin RBAC, so treat it like a
root password. `doctor` warns if a key is sitting unencrypted in a plain file.

## Logging in

```bash
# pipe the key on stdin — it is validated against the console before being stored:
printf %s "$KEY" | ufi auth login

# or provide it via env (overrides the keyring); useful in CI / containers:
export UNIFI_HOST=https://192.168.1.1
export UNIFI_API_KEY=…
```

`ufi auth login` validates the key against `GET /info` and stores it in the OS keyring.

## Host, site & TLS

```bash
export UNIFI_HOST=https://192.168.1.1   # or --host
export UNIFI_SITE=default               # or --site (most consoles have one site)
export UNIFI_INSECURE=1                 # or --insecure: accept the console's self-signed cert
```

Most UniFi consoles ship a **self-signed** certificate, so verification fails by default. The
`--insecure` / `UNIFI_INSECURE=1` escape hatch is **off by default** and warns loudly when used.

## status / logout

```bash
ufi auth status --json   # reports which key is stored and whether it works — never prints it
ufi auth logout          # removes the LOCAL copy only
```

`auth logout` does **not** revoke the key server-side — **rotate/revoke it on the console** to
invalidate it.

## doctor

```bash
ufi doctor --json
```

Runs the full diagnostic — host reachability, key validity, live connectivity, TLS (and whether
`--insecure` is in play), permissions, and clock skew — emitting `checks[]{name, ok, detail}`
with a fix under each failure. A missing/invalid key returns `AUTH_REQUIRED` (exit 4) naming the
login command.

## Secret-handling threat model

- **Storage:** OS keyring (Keychain / Secret Service / Credential Manager) with a `0600` file
  fallback under `$XDG_CONFIG_HOME/ufi/credentials`. `doctor` warns on loose perms or a key left
  in an env var on a shared host.
- **Never via argv** — the key comes from stdin/env; argv leaks to `ps` / `/proc` / history /
  agent logs.
- **Redaction** — `auth status` reports validity only; ufi never logs the key, and redacts it
  from `--verbose` and error output.
- **Full-admin blast radius** — because a UniFi key bypasses RBAC, the read-only-by-default +
  `--allow-mutations` gating is the primary guardrail against an agent making unintended changes.

See [SECURITY.md](https://github.com/rnwolfe/ufi/blob/main/SECURITY.md) for the reporting process
and the leaked-key runbook.
