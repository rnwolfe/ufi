# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-26

### Added
- Initial release of **ufi**, an agent-focused CLI for Ubiquiti UniFi Network over the
  official local **Network Integration API** (`X-API-KEY`), built to the Agent CLI
  Guidelines (Full conformance).
- **Reads:** `info`, `site list`, `device list|get|stats`, `client list|get`, `wifi list|get`,
  `voucher list`, `network list|get`, `firewall policy|zone list|get`, `acl list|get`,
  `dns policy list|get`, `traffic-list list|get`.
- **Gated mutations:** `device restart`, `device port-cycle`, `client authorize|unauthorize`,
  `voucher create|delete` — read-only by default, behind `--allow-mutations` with `--dry-run`.
- **Declarative config** (`network`/`firewall`/`acl`/`dns`/`traffic-list` create|update|delete|
  reorder) via a reviewed-artifact flow: `--dry-run` emits a plan + hash, `ufi apply <hash>`
  executes exactly that plan.
- **Agent contract:** machine-discoverable `schema --json` (with a conformance block), embedded
  `SKILL.md` via `ufi agent`, stable exit codes, structured `{error, code, remediation}`,
  bounded output with opaque-cursor pagination (`--limit`/`--cursor`/`--page`/`--select`),
  and prompt-injection fencing of network-controlled names/notes (`--no-fence` to disable).
- **Auth:** `auth login|status|logout|refresh` with OS-keyring storage (env → keyring → `0600`
  file), `doctor`, and a self-signed-TLS escape hatch (`--insecure` / `UNIFI_INSECURE`).
- Zone-Based-Firewall-not-configured maps to a clear `UNSUPPORTED` (exit 11).

### Notes
- The Site Manager **cloud** surface is deferred; `ufi cloud …` returns `UNSUPPORTED` with a
  pointer to open an issue. Local-only for now.
