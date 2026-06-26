# Contributing to ufi

Thanks for your interest! ufi is a Go CLI built to the [Agent CLI
Guidelines](https://aclig.dev/) — contributions should preserve that contract: read-only by
default, a single `--allow-mutations` gate (plus the reviewed-artifact `apply <hash>` flow for
config), structured errors + stable exit codes, bounded JSON output, prompt-injection fencing,
and stdin/keyring secrets (never argv).

## Setup

```bash
git clone https://github.com/rnwolfe/ufi && cd ufi
go build ./...        # build
go vet ./...          # vet
go test ./...         # tests (incl. the schema-snapshot gate)
go run ./cmd/ufi --help
```

Go 1.25+. No CGO required (`CGO_ENABLED=0` builds statically).

## Tests

- Offline contract tests, a mock-console integration test, and `unifi` client unit tests run
  with `go test ./...`.
- The **schema snapshot is a CI gate** — if you intentionally change the command tree, flags,
  or exit codes, regenerate it and review the diff:
  `UFI_UPDATE_GOLDEN=1 go test ./internal/cli -run TestSchemaGolden`.
- Endpoint behavior should be covered by an `httptest` fixture (see `internal/unifi/*_test.go`
  and `internal/cli/integration_test.go`); the committed `integration-openapi.json` (the
  console's own OpenAPI 3.1 spec) is the authoritative reference for paths/fields/enums.

## Conventional Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) — they drive the changelog
and version bumps. Examples: `feat(cli): …`, `fix(client): …`, `docs: …`, `chore: …`.

Sign off your commits (DCO): `git commit -s`. There is no CLA.

## Pull requests

- Keep the contract intact; if you add a command/flag, update `schema`, the embedded
  `SKILL.md`, the docs, and the landing copy in the **same PR** (see `AGENTS.md`).
- Output fields are **append-only** — renames/removals are breaking and need discussion.
- Every mutation must call the `Guard` gate first and honor `--dry-run`; declarative config
  goes through the `apply <hash>` preview, never a direct write.
- `go build ./... && go vet ./... && go test ./...` must pass; `gofmt -l` must be clean.
- Don't commit build artifacts, `dist/`, `node_modules/`, or `.vercel/`.

## Scope

ufi targets Ubiquiti's **official** APIs only. The local Network Integration API is fully
wired; the Site Manager **cloud** surface is intentionally deferred (the `cloud` command is a
hidden stub). Please open an issue to discuss before adding cloud or any legacy/unofficial-API
behavior.
