# Security Policy

`ufi` is a credential-handling CLI: it stores a UniFi **API key** (effectively full-admin to
your console) and is designed to be driven autonomously by LLM agents. Security is a
first-class concern.

## Supported versions

| Version | Supported |
|---|---|
| latest `0.x` | ✅ |
| older | ❌ (upgrade to latest) |

Until `1.0`, only the latest release receives fixes.

## Reporting a vulnerability

**Do not open a public issue for security reports.** Use GitHub **Private Vulnerability
Reporting**: the repo's **Security → Report a vulnerability** tab. If that is unavailable,
email the maintainer (`rn.wolfe@gmail.com`) with `ufi security` in the subject.

- **Acknowledgement SLA:** within ~48 hours.
- **Coordinated disclosure:** we'll agree on a fix timeline and a disclosure date; please give
  us a reasonable window before any public detail.
- **Safe harbor:** good-faith research that respects user privacy and avoids data destruction
  will not be pursued.
- Include a minimal reproducible PoC and the affected version (`ufi version`).

## Secret-handling threat model

What ufi stores, where, and how it tries to fail safe:

- **The key is full-admin.** A UniFi Integration API key bypasses per-admin RBAC and grants
  full access to the Network application. Treat it like a root password. ufi never weakens
  this; it just handles the key carefully.
- **Storage.** The key goes to the OS keyring (macOS Keychain / Linux Secret Service / Windows
  Credential Manager). The fallback is a `0600` file under `$XDG_CONFIG_HOME/ufi/credentials`
  on headless hosts with no keyring backend (ufi degrades gracefully — it never blocks on a
  passphrase prompt). `ufi doctor` and `ufi auth status` warn if the file's perms are looser
  than `0600`.
- **Never via argv.** The key is read from **stdin** (`ufi auth login`) or the `UNIFI_API_KEY`
  env var, never passed as a flag — argv leaks to `ps`, `/proc`, shell history, and an agent's
  own command log.
- **Redaction.** `auth status` reports validity only; it never prints the stored key. ufi does
  not log credentials.
- **Read-only by default; mutations are gated.** No state-changing operation runs without an
  explicit `--allow-mutations`. High-stakes declarative config (firewall/network/DNS/ACL) uses
  a reviewed-artifact flow: `--dry-run` emits a plan + hash and `ufi apply <hash>` executes
  only that exact plan, closing the time-of-check/time-of-use gap a blind confirmation opens.
  The worst-case blast radius of a compromised invocation is bounded by whether
  `--allow-mutations` is in play.
- **Untrusted upstream content.** Device, client, and SSID names/notes are attacker-
  controllable (a guest can name a device `Ignore previous instructions…`). ufi fences them as
  `[UNTRUSTED_DATA_BEGIN] … [UNTRUSTED_DATA_END]` by default in agent mode (`--no-fence` to
  disable) to reduce prompt-injection risk for the driving agent.
- **TLS.** UniFi consoles ship a self-signed cert; `--insecure` / `UNIFI_INSECURE=1` disables
  verification and **warns loudly** every invocation. It is off by default.
- **Official API only.** ufi speaks Ubiquiti's versioned Integration API — no reverse-
  engineered endpoints, no credential scraping, no evasion.

## If a key leaks

1. **Revoke it on the console** (Settings → Control Plane → Integrations → API Keys) and
   create a new one — `ufi auth logout` only removes the **local** copy, it does not revoke
   the key server-side.
2. Re-run `ufi auth login` with the new key.
3. Never paste a real key into an issue; redact it and rotate first.
