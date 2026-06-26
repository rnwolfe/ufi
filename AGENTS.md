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
- `internal/cli/misc.go` — `auth`/`doctor`/`schema`/`agent`/`version`.
- `internal/output/`, `internal/errs/` — output discipline + exit-code table.
- `internal/skill/SKILL.md` — embedded in the binary (`ufi agent` prints it).
- `internal/unifi/` — the API client (Integration + Site Manager): `X-API-KEY`, self-signed-TLS
  escape hatch, offset→opaque-cursor pagination, error classification, generic snake_casing.
- `internal/auth/` — env → OS keyring → 0600-file credential resolution.
- `internal/plan/` — config-plan persistence for `ufi apply <hash>`.
- `spec.md` — single source of truth. `integration-openapi.json` — the console's own OpenAPI 3.1
  spec (verified on UniFi Network 10.4.57), the authoritative endpoint/field/enum reference.

## Status: IMPLEMENTED (local API validated live on Network 10.4.57)
Reads, single-target mutations, declarative-config preview/apply, auth (keyring), doctor, and
untrusted-text fencing are wired and validated against real hardware. Tests: offline contract +
mock-console integration + `unifi` unit tests, with a schema-golden CI gate (`go test ./...`).

### Known follow-ups
- **Site Manager cloud** (`ufi cloud …`) is **deferred/hidden** — local-only build. The `cloud`
  command is a hidden stub (`cloud.go`) that returns `UNSUPPORTED` + an issue-tracker pointer; it
  is omitted from `--help` and `schema` (nodeToMap skips hidden nodes). The client groundwork
  (`unifi.NewCloud`, `CloudBase`) remains for re-enabling once a cloud key path is validated.
- Server-side `--filter` (RSQL) is plumbed in the client (`unifi.ListOpts.Filter`) but not yet
  exposed as a flag — agents over-fetch then client-filter for now.
- Config `--data` bodies are passed through opaquely (validated only as "is it JSON"); the
  console validates the rest and returns structured errors.

## Conventions
- stdout = data, stderr = everything else. JSON is 2-space, HTML-escaping off.
- Every mutation calls `rt.Guard(op)` FIRST, then honors `--dry-run`.
- Output fields are an **append-only** contract; the `schema --json` shape is a CI gate.
- Never put secrets in argv; never log API keys.

## Web presence freshness (keep docs + landing + cards in sync)

The site lives in `site/` (Astro + Starlight). **One shared token source** —
`site/src/styles/tokens.css` (the NOC-console theme) — styles BOTH the landing
(`site/src/pages/index.astro`) and the docs (via `customCss` in `site/astro.config.mjs`). Never
hand-copy tokens.

When the **value proposition, command surface, flags, exit codes, or brand** change, in the
**same PR**:

1. Update the affected docs page under `site/src/content/docs/`.
2. Update the **landing** copy/examples in `site/src/pages/index.astro` (hero, the "other way vs
   guidelines way" comparisons, exit-code table) and the README's mirrored sections.
3. **Regenerate OG cards** if any page title/headline changed: `cd site && node scripts/gen-og.mjs`
   (keep the page list in `scripts/gen-og.mjs` and the `known` list in `src/components/Head.astro`
   in sync). If the brand/tagline/headline command changed, also regenerate the GitHub social card:
   `cd site && node scripts/gen-social.mjs` → `public/social-card.png` + `.github/social-preview.png`
   (re-upload in repo Settings → Social preview).
4. Rebuild (`cd site && pnpm build`) so `llms.txt` regenerates, and keep the embedded
   `internal/skill/SKILL.md` aligned (it's the agent contract shipped in the binary).

Build/preview the site: `cd site && pnpm build` / `pnpm dev` (binds 0.0.0.0). Deploy is Vercel
(`uficli.sh`, pending domain purchase; live preview at the project's `*.vercel.app` alias).

## Stages
`cli-plan` → `cli-scaffold` → `cli-implement` (done) → `cli-publish` (this web presence + hygiene).
