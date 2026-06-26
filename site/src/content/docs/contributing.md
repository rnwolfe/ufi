---
title: Development
description: How to build, test, extend, and contribute to the ufi codebase.
---

This guide is for developers who want to fix bugs, add commands, or understand how ufi
is wired. It covers the repo layout, the build/test loop, the schema-snapshot gate, the
rules for adding a command, and the contribution workflow.

## Prerequisites

Go 1.25 or later. No CGO required — `CGO_ENABLED=0` produces a fully static binary.

```bash
git clone https://github.com/rnwolfe/ufi && cd ufi
go build ./...
go vet ./...
go test ./...
go run ./cmd/ufi --help
```

## Repository layout

```
cmd/ufi/
  main.go                   # os.Exit(cli.Run(...)); all logic is in internal/

internal/
  cli/
    root.go                 # kong grammar, global flags, Runtime, Guard, emitError
    helpers.go              # listEnvelope — the stable list envelope constructor
    runtime.go              # Runtime helper methods: siteList/siteGet/siteAction/siteDelete,
                            #   configWrite/readData, fenceItems/fenceValue, emitList
    device.go               # device noun (list, get, stats, restart, port-cycle)
    core.go                 # info, site, client, wifi, voucher nouns
    config.go               # network, firewall, acl, dns, traffic-list, apply (declarative config)
    cloud.go                # hidden stub — returns UNSUPPORTED (exit 11) + issue pointer
    misc.go                 # auth, doctor, schema, agent, version
    suggest.go              # "did you mean" typo suggestion
    cli_test.go             # offline contract tests (mutation gate, dry-run, hash, cloud stub…)
    schema_golden_test.go   # contract-stability snapshot gate
    integration_test.go     # mock-console integration tests (envelope shape, fencing, idempotent delete…)
    testdata/schema.json    # committed schema snapshot (the golden file)

  unifi/
    client.go               # HTTP client: X-API-KEY auth, self-signed TLS, offset→cursor pagination,
                            #   error classification, generic camelCase→snake_case conversion
    client_test.go          # toSnake, cursor round-trip, pagination, error-classification unit tests

  auth/
    auth.go                 # env → OS keyring → 0600-file credential resolution

  plan/
    plan.go                 # declarative-config plan persistence ($XDG_STATE_HOME/ufi/plans/<hash>.json)

  output/
    output.go               # Writer: stdout=data / stderr=warnings, --format, --select, --limit

  errs/
    errs.go                 # stable exit-code constants + CLIError type + MutationBlocked/NotFound constructors

  version/
    version.go              # version check against GitHub releases (fail-silent, UFI_NO_UPDATE_CHECK=1)

  skill/
    SKILL.md                # embedded agent contract — `ufi agent` prints this

integration-openapi.json    # console's own OpenAPI 3.1 spec (authoritative path/field/enum reference)
spec.md                     # single source of truth for the ufi contract
```

The `cloud.go` stub is intentionally hidden from `--help` and `schema`. The cloud surface is
deferred; see [Local-only scope](#local-only-scope) below.

## Build, vet, format, and test

Run these before every commit — CI enforces all four:

```bash
go build ./...              # must succeed
go vet ./...                # must be clean
gofmt -l .                  # must print nothing (no unformatted files)
go test ./...               # must pass (includes the schema-snapshot gate)
```

`go test ./...` runs three suites without any network access:

- **Offline contract tests** (`internal/cli/cli_test.go`): mutation gate, `--dry-run`,
  config preview hash, `apply` with unknown hash, `--write` alias, `UFI_HELP=agent`, cloud
  stub, schema structure, "did you mean", `version --check`.
- **Mock-console integration tests** (`internal/cli/integration_test.go`): a real
  `httptest.Server` emulates the Integration API base path
  (`/proxy/network/integration/v1`). These verify the list envelope shape, snake_case
  conversion, untrusted-text fencing, idempotent voucher delete, EMPTY exit code, and auth
  status.
- **`unifi` client unit tests** (`internal/unifi/client_test.go`): `toSnake`, cursor
  round-trip, offset-based pagination with a live cursor, and error-classification for
  zone-based-firewall (`UNSUPPORTED`, exit 11) and 401 (`AUTH`, exit 4).

## The schema-snapshot gate

`internal/cli/testdata/schema.json` is a committed golden file of the full `ufi schema`
output (minus the volatile `version` field). `TestSchemaGolden` diffs the live schema
against it on every `go test` run. If they diverge, the test fails with:

```
schema drift vs testdata/schema.json — review the diff; if intended, regenerate with UFI_UPDATE_GOLDEN=1
```

This is intentional: a silent rename or removal of a command, flag, or exit code is a
breaking change. The gate forces you to review the diff before landing it.

When you **intentionally** change the command surface, regenerate the snapshot and commit
the diff as part of your PR:

```bash
UFI_UPDATE_GOLDEN=1 go test ./internal/cli -run TestSchemaGolden
git diff internal/cli/testdata/schema.json   # review carefully
```

The `schema` output is also the source for `ufi schema` at runtime — agents depend on it
for self-description. Treat it like a public API. Output fields are **append-only**: adding
a field is non-breaking; renaming or removing one is breaking and needs discussion.

## `integration-openapi.json` — the authoritative field reference

`integration-openapi.json` is the UniFi console's own OpenAPI 3.1 spec (verified on
Network 10.4.57). It is the canonical reference for:

- API paths and HTTP methods
- Request/response field names (camelCase as returned by the API)
- Enum values (device states, action types, etc.)

When the console's API behavior and `spec.md` ever disagree, trust `integration-openapi.json`
and update `spec.md`. Never invent a field or endpoint that isn't in the spec.

## Adding a command

Follow this checklist to keep the contract intact. CI will catch most misses through the
schema-snapshot gate and the test suite.

### 1. Wire the grammar in `root.go`

Add your noun or verb to the `CLI` struct and register it with kong. Follow the existing
noun-verb convention (`DeviceCmd → device list`, `device restart`, …). Read-only commands
live in the "read-first core" block; declarative-config commands go in the "declarative
config" block.

### 2. Implement in the appropriate file

| Kind | File |
|---|---|
| Read-only device/client/wifi/site/info | `core.go` or `device.go` |
| Declarative config (network, firewall, acl, dns, traffic-list) | `config.go` |
| Auth / doctor / schema / agent / version | `misc.go` |

### 3. Every mutation calls `Guard` first

The read-only-by-default gate is enforced by `rt.Guard(op)` in `internal/cli/root.go`. Call
it as the very first line of any state-changing `Run` method, before any API call:

```go
func (c *DeviceRestartCmd) Run(rt *Runtime) error {
    if err := rt.Guard("device restart"); err != nil {
        return err
    }
    // ... then honor --dry-run ...
}
```

A missing `Guard` call is a contract violation — the mutation gate is part of the
[safety model](/safety-model/).

### 4. Single-target mutations support `--dry-run`

After `Guard` passes, check `rt.Cfg.DryRun`. If set, emit a preview object and return
without calling the API:

```go
if rt.Cfg.DryRun {
    return rt.Out.Emit(map[string]any{
        "dry_run": true,
        "action":  "RESTART",
        "id":      c.ID,
    })
}
```

### 5. Declarative config goes through `configWrite` / `apply <hash>`

Never issue a direct write for declarative-config commands. Use the thin wrappers in
`internal/cli/config.go` (`configCreate`, `configUpdate`, `configDelete`, `configReorder`) —
they parse `--data`, build the request body, then delegate to `rt.configWrite` in
`internal/cli/runtime.go`, which computes a plan and 12-hex hash, persists it under
`$XDG_STATE_HOME/ufi/plans/<hash>.json`, and emits the plan summary. The operator then
runs `ufi apply <hash> --allow-mutations` to execute exactly that plan. This closes the
TOCTOU gap.

See [Config](/config/) for the full user-facing flow.

### 6. List commands emit the stable list envelope

Wrap results in `listEnvelope(items, count, nextCursor)` from `internal/cli/helpers.go`.
The shape `{ schemaVersion, items, count, nextCursor }` is a contract — agents parse it.
For most list commands use `rt.emitList(res)` (defined in `internal/cli/runtime.go`), which
handles the envelope and automatically sets `rt.ExitCode = errs.ExitEmpty` (exit 3) when
`res.Count == 0`, so the envelope is always emitted even on empty results:

```go
res, err := c.List(ctx, "/sites/"+site+"/"+subpath, rt.listOpts())
if err != nil {
    return err
}
return rt.emitList(res)
```

See [Output](/output/) for the full envelope spec.

### 7. Fence network-controlled free text

If a field in your response comes from network-controlled free text (device name, client
hostname/note, voucher name), wrap it when `rt.Fence` is true. See
`internal/cli/runtime.go` for the `fenceItems` and `fenceValue` helpers, or pass the
untrusted key names as variadic arguments to `rt.siteList` / `rt.siteGet`. Operator-set
fields (site name, network name) are NOT fenced. See [Agents](/agents/) for the
prompt-injection fencing details.

### 8. Errors use `internal/errs`

Return `errs.NotFound(kind, id)`, `errs.New(exit, code, msg, remediation)`, or a wrapped
upstream `*errs.CLIError`. Never `fmt.Errorf` alone for user-facing errors — structured
errors carry machine-readable codes and remediation hints. See [Errors](/errors/) and
[Exit Codes](/exit-codes/).

### 9. Update the schema snapshot, SKILL.md, docs, and site in the same PR

This is the most commonly missed step. When your PR changes any command, flag, exit code,
or value proposition:

1. Regenerate the schema golden file:
   `UFI_UPDATE_GOLDEN=1 go test ./internal/cli -run TestSchemaGolden`
2. Update `internal/skill/SKILL.md` (the embedded agent contract that `ufi agent` prints).
3. Update or add the relevant doc page under `site/src/content/docs/`.
4. Update `site/src/pages/index.astro` and `README.md` if the command surface or
   headline examples changed.
5. Rebuild the site: `cd site && pnpm build` (regenerates `llms.txt`).
6. Regenerate OG cards if any page title changed:
   `cd site && node scripts/gen-og.mjs`

The site lives in `site/` (Astro + Starlight). One shared token source
(`site/src/styles/tokens.css`) styles both the landing page and the docs — never
hand-copy tokens.

### 10. Add tests

- Offline contract test in `cli_test.go` for gate behavior, exit codes, and output shape.
- Mock-console integration test in `integration_test.go` for real HTTP round-trips using
  `mockConsole(t)`.
- Unit test in `internal/unifi/client_test.go` for new API client behavior or error
  classification.

## Output contract

All output routing flows through `internal/output.Writer`:

- **stdout** — data only (list envelopes, single-object responses, plan previews).
- **stderr** — everything else: warnings, error objects, the `--insecure` TLS warning.
- **JSON** — 2-space indented, HTML-escaping off (`SetEscapeHTML(false)` — so URLs survive).
- **Field names** — always snake_case. The `unifi` client converts camelCase API fields
  generically: `macAddress → mac_address`, `uptimeSec → uptime_sec`, etc.

Output fields are **append-only**. Adding a new field to an existing command is safe and
does not require a schema-version bump. Renaming or removing a field is breaking.

The `schemaVersion` field on the list envelope is bumped only for breaking output changes,
not for additive ones.

## Error classification

`internal/unifi/client.go` classifies upstream UniFi error responses
(`{statusCode, statusName, code, message}`) into `*errs.CLIError` values with the
appropriate exit code. When adding a new command that hits a new endpoint, check whether
the console can return any novel error shapes and add classification coverage in
`client_test.go`.

Notable classifications:

| UniFi response | ufi code | exit |
|---|---|---|
| HTTP 401 | `AUTH_REQUIRED` | 4 |
| HTTP 404 | `NOT_FOUND` | 5 |
| HTTP 429 | `RATE_LIMITED` | 7 |
| `api.firewall.zone-based-firewall-not-configured` | `UNSUPPORTED` | 11 |
| Blocked without `--allow-mutations` | `MUTATION_BLOCKED` | 12 |

See [Exit Codes](/exit-codes/) and [Errors](/errors/) for the full table.

## Local-only scope

ufi targets Ubiquiti's **official** local Network Integration API
(`https://{host}/proxy/network/integration/v1`). The Site Manager cloud surface is
intentionally deferred — `cloud.go` is a hidden stub that returns `UNSUPPORTED` (exit 11)
with a pointer to the issue tracker. It is absent from `--help` and `ufi schema` output
(`nodeToMap` skips hidden nodes), and `TestCloudStubUnsupported` in `cli_test.go` guards
this contract.

Please open an issue before adding cloud, legacy, or unofficial-API behavior. The client
groundwork (`unifi.NewCloud`, `CloudBase`) exists in the codebase for future re-enabling
once a cloud key path is validated.

## Conventional Commits and DCO

Use [Conventional Commits](https://www.conventionalcommits.org/) — they drive the
changelog and version bumps:

```
feat(cli): add traffic-list reorder command
fix(client): retry on 503 before returning RETRYABLE
docs: document the apply flow
chore: bump go.mod to 1.25
```

Sign off every commit (DCO — no CLA):

```bash
git commit -s -m "feat(cli): add wifi broadcast get command"
```

## Pull request checklist

Before opening a PR:

- [ ] `go build ./... && go vet ./... && gofmt -l . && go test ./...` all pass
- [ ] Schema golden regenerated if the command surface changed
- [ ] `internal/skill/SKILL.md` updated if the agent contract changed
- [ ] Docs page updated or added under `site/src/content/docs/`
- [ ] Landing and README updated if the command surface or examples changed
- [ ] Site rebuilt (`cd site && pnpm build`)
- [ ] No build artifacts, `dist/`, `node_modules/`, or `.vercel/` committed
- [ ] Secrets never appear in argv or logs

Related pages: [Safety Model](/safety-model/) · [Output](/output/) · [Exit Codes](/exit-codes/) · [Errors](/errors/) · [Agents](/agents/) · [Config](/config/)
