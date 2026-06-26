---
title: Getting started
description: Install ufi, learn its contract offline, then point it at your UniFi console for a first bounded JSON result.
---

`ufi` is an **agent-friendly** CLI for **Ubiquiti UniFi Network**, built on Ubiquiti's
**official local Integration API** — no reverse-engineered controller endpoints. It lists
devices, clients, WiFi, firewall, and DNS as bounded, structured JSON. It is **read-only by
default**: every state-changing command is gated behind `--allow-mutations`, and high-stakes
config goes through a reviewed `apply <hash>` flow.

## Install

```bash
# Homebrew (macOS / Linux) — recommended
brew install rnwolfe/tap/ufi

# Go (any platform; static binary, no CGO)
go install github.com/rnwolfe/ufi/cmd/ufi@latest

# Shell script (macOS / Linux) — verifies the release SHA-256 before installing to ~/.local/bin
curl -fsSL https://uficli.sh/install.sh | sh
```

Or download a prebuilt binary from [Releases](https://github.com/rnwolfe/ufi/releases)
(checksums + SBOM + build-provenance attestation included).

## Learn the contract offline (no console)

`ufi` describes itself with no network and no console — having the binary is enough:

```bash
ufi --help
ufi agent                                    # the embedded agent guide (SKILL.md)
ufi schema | jq '{conformance, exit_codes}'  # machine-readable command tree + safety
```

## Point it at your console

Generate an API key in the UniFi UI
(**Settings → Control Plane → Integrations → Create API Key**), then:

```bash
export UNIFI_HOST=https://192.168.1.1
export UNIFI_API_KEY=…          # or: printf %s "$KEY" | ufi auth login  (stored in the OS keyring)
# self-signed console cert? add --insecure (or set UNIFI_INSECURE=1)

ufi doctor --json                            # host / key / connectivity / TLS / clock
ufi device list --json --select id,name,model,state
ufi client list --json --limit 20
```

A UniFi API key is effectively **full admin** (it bypasses per-admin RBAC) — treat it like a
root password. See [Authentication & security](/auth) for storage and the threat model.

## Your first mutation

Mutations are opt-in. A blocked one is a *signal*, not a failure — exit `12`,
`{"code":"MUTATION_BLOCKED"}` — so an agent can tell "ask permission" from "real error":

```bash
ufi device restart <id>                       # → MUTATION_BLOCKED (exit 12)
ufi device restart <id> --allow-mutations     # actually reboots the device
```

## Next steps

- [Authentication & security](/auth) — API key, the console UI path, keyring, `doctor`.
- [Reviewed config — apply &lt;hash&gt;](/config) — the preview → hash → apply flow.
- [Command reference](/commands) — every command, flag, and exit code.
- [For agents](/agents) — the contract an LLM agent relies on.
