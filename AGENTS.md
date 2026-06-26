# AGENTS.md — ufi

Agent-focused CLI for **Ubiquiti UniFi Network** over the **official** APIs (local Integration
API + Site Manager cloud). Go + [kong](https://github.com/alecthomas/kong). Built by the
agent-cli-factory; conforms to the Agent CLI Guidelines (see `internal/skill/SKILL.md`).

## Build / test / run
```sh
go build ./...                  # build all packages
go vet ./...                    # static checks
go test ./...                   # contract tests (internal/cli/cli_test.go)
gofmt -l .                      # must print nothing (formatting gate)
go run ./cmd/ufi schema         # the machine-readable contract
go run ./cmd/ufi --help         # example-led help
```

## Layout (contract surface — do not weaken)
- `cmd/ufi/main.go` — `os.Exit(cli.Run(...))`; all logic is in-process + testable.
- `internal/cli/root.go` — kong grammar (global flags + command tree), `Runtime`, the
  `Guard` mutation gate, structured `emitError`, "did you mean".
- `internal/cli/helpers.go` — list envelope, the gated-action / idempotent-delete /
  config-preview (`apply <hash>`) helpers.
- `internal/cli/{device,core,config,cloud}.go` — the noun-verb command surface.
- `internal/cli/misc.go` — `auth`/`doctor`/`schema`/`agent`/`version` skeletons.
- `internal/output/`, `internal/errs/` — output discipline + exit-code table. **Don't touch.**
- `internal/skill/SKILL.md` — embedded in the binary (`ufi agent` prints it).
- `internal/store/` — **placeholder target** (local JSON). Replaced by the real client in cli-implement.
- `spec.md` — single source of truth. `integration-openapi.json` — the console's own OpenAPI 3.1
  spec (verified on UniFi Network 10.4.57), the authoritative endpoint/field/enum reference.

## Status: SCAFFOLD (cli-implement is next)
The contract surface is complete and green; the **UniFi logic is not wired yet**. Reads emit
placeholder envelopes/objects; single-target mutations gate + preview under `--dry-run` and
return `NOT_IMPLEMENTED` on real execution; config writes emit a plan + hash but don't persist.

### What cli-implement must wire (read `spec.md` + `integration-openapi.json` first)
- Replace `internal/store/` with a UniFi HTTP client (`internal/unifi/`): `X-API-KEY` header,
  base `https://{host}/proxy/network/integration/v1/`, self-signed-TLS handling (`--insecure`),
  offset/limit/filter pagination (`{offset,limit,count,totalCount,data}` → `nextCursor`).
- `internal/auth/` with the **OS keyring** (env → keyring → `0600` file; OS-native backends only,
  no passphrase-prompt backend) + `auth login/status/logout`; redact keys everywhere.
- Map upstream errors `{statusCode,statusName,code,message,...}` → the exit-code table (incl. the
  Zone-Based-Firewall `400 api.firewall.zone-based-firewall-not-configured` → `unsupported`).
- Implement the Site Manager cloud client (`ufi cloud …`) with `Retry-After` / 429 backoff.
- Persist config plans under `$XDG_STATE_HOME/ufi/plans/<hash>.json` and make `ufi apply <hash>` execute them.
- camelCase wire → snake_case output mapping (see `spec.md` "Wire vs. output naming").
- Wrap network-controlled free text (device/client/SSID/voucher names/notes) as untrusted (§8).

## Conventions
- stdout = data, stderr = everything else. JSON is 2-space, HTML-escaping off.
- Every mutation calls `rt.Guard(op)` FIRST, then honors `--dry-run`.
- Output fields are an **append-only** contract; the `schema --json` shape is a CI gate.
- Never put secrets in argv; never log API keys.
