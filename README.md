# ufi

An **agent-friendly CLI for Ubiquiti UniFi Network**, built on Ubiquiti's **official** APIs only
— the local **Network Integration API** and the **Site Manager** cloud API. No reverse-engineered
legacy controller endpoints, so it stays stable as Ubiquiti versions the API.

`ufi` is engineered to the [Agent CLI Guidelines](https://aclig.dev): read-only by default,
machine-discoverable (`schema --json`), structured errors + stable exit codes, bounded token
output, default-deny mutations with `--dry-run` previews, and prompt-injection fencing of
network-supplied text — all in a single static binary.

> **Status: scaffold.** The command surface and agent-CLI contract are complete and tested; the
> UniFi API calls are being wired (see `AGENTS.md`). Not yet released.

## Install

```sh
go install github.com/rnwolfe/ufi/cmd/ufi@latest
# or (once released) brew install rnwolfe/tap/ufi
```

## Quick start

```sh
export UNIFI_HOST=https://192.168.1.1
export UNIFI_API_KEY=…        # console → Settings → Control Plane → Integrations → API
# self-signed cert? add --insecure (or UNIFI_INSECURE=1)

ufi doctor --json             # check host/key/connectivity
ufi device list --json        # adopted devices (stable list envelope)
ufi client list --json        # connected clients
ufi device restart <id> --allow-mutations   # mutations are opt-in
```

## Why another UniFi CLI?

Capable UniFi CLIs and MCP servers exist, but they're built for humans or a different transport.
`ufi`'s wedge is the **agent contract** as a single binary: a stable, append-only JSON output
schema; `schema --json` self-description (with a machine-readable conformance block);
default-deny `--allow-mutations` + `--dry-run`; a reviewed-artifact `apply <hash>` flow for
high-stakes config; rigorous exit codes; and official-API-only stability.

## Safety model

- **Read-only by default.** State-changing commands need `--allow-mutations`; blocked ones return
  exit `12` + `{"code":"MUTATION_BLOCKED"}`.
- **Preview everything.** `--dry-run` prints the intended change and does nothing.
- **Declarative config** (`network`, `firewall`, `acl`, `dns`, `traffic-list`) previews a plan +
  hash; `ufi apply <hash>` executes exactly that plan.
- **Credentials** live in the OS keyring (never argv); a UniFi API key is full-admin — guard it.

## Agent usage

`ufi agent` prints the embedded usage contract; `ufi schema` dumps the full command tree, every
flag, the exit-code table, and live safety state. See [`SKILL.md`](internal/skill/SKILL.md).

## License

MIT (see `LICENSE`, added at publish).
