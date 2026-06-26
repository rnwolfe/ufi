<!-- Use a Conventional Commit title, e.g. "feat(cli): add X" -->

## What & why

<!-- What does this change and why? Link any issue. -->

## Checklist

- [ ] `go build ./... && go vet ./... && go test ./...` pass; `gofmt -l` clean
- [ ] Conventional Commit title; commits signed off (`git commit -s`)
- [ ] If the command tree / flags / exit codes changed: schema golden regenerated
      (`UFI_UPDATE_GOLDEN=1 go test ./internal/cli -run TestSchemaGolden`) and the embedded
      `SKILL.md` + docs + landing copy updated in this PR
- [ ] Output schema changes are append-only (no field renames/removals)
- [ ] Mutations call the `Guard` gate first and honor `--dry-run`; config goes through `apply <hash>`
- [ ] New/changed endpoint behavior covered by an `httptest` fixture
- [ ] CHANGELOG `[Unreleased]` updated
