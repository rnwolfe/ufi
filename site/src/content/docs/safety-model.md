---
title: The safety model
description: How ufi keeps your network safe by default — read-only by default, a single mutation gate, dry-run previews, and a reviewed-artifact apply flow for declarative config changes.
---

Every command in ufi is safe to run without thinking twice — unless you explicitly say otherwise. This page explains the layered safety model so you understand exactly what you're opting into when you add `--allow-mutations`.

## Read-only by default

Out of the box, ufi will never touch your network. List commands, get commands, `ufi doctor`, `ufi auth status` — all of them read data and exit. Any command that would change state is blocked until you pass `--allow-mutations` (alias: `--write`).

If you forget, you get a structured error instead of a surprise:

```bash
ufi device restart abc123
```

```json
{
  "error": "device restart is a mutating operation and is blocked by default",
  "code": "MUTATION_BLOCKED",
  "remediation": "re-run with --allow-mutations (add --dry-run to preview)"
}
```

Exit code **12** (`mutation_blocked`). The error goes to stderr; stdout stays clean. See [Exit codes](/exit-codes/) for the full table.

This behaviour is enforced by a single `Guard` call at the top of every mutation path:

```go
// Guard enforces the read-only-by-default mutation gate.
func (rt *Runtime) Guard(op string) error {
    if rt.Cfg.AllowMutations {
        return nil
    }
    return errs.MutationBlocked(op)
}
```

There are no exceptions, no `--force` escape hatches, and no ambient state that unlocks mutations. You always have to mean it.

## The single gate: `--allow-mutations`

`--allow-mutations` (alias `--write`) is a **global flag** — it lives at the root level, not per-subcommand. You can put it anywhere in the invocation:

```bash
ufi --allow-mutations device restart abc123
ufi device restart abc123 --allow-mutations   # same thing
ufi device restart abc123 --write             # alias form
```

There is no separate flag per operation type. One flag, one intent.

:::caution[A UniFi API key is effectively full-admin]
The API key bypasses per-admin RBAC on the console — it has full read-write access regardless of which admin account it was created under. Treat it like a root password. See [Auth](/auth/) for how ufi stores and protects it.
:::

## `--dry-run` previews

Every mutating command accepts `--dry-run`. When set, the command prints what it _would_ do without touching the network. `--dry-run` still requires `--allow-mutations` — the guard runs first — so the dry-run itself tells you whether you've opted in correctly:

```bash
ufi device restart abc123 --allow-mutations --dry-run
```

```json
{
  "action": "RESTART",
  "id": "abc123",
  "dry_run": true
}
```

Exit code **0**. No network call was made.

## Two tiers of mutation

Not all mutations are equal. ufi divides them into two tiers with different safety mechanisms.

### Tier 1 — Low-stakes single-target actions

These operations act on one identified resource right now. The blast radius is bounded: you're doing one thing to one device, client, or voucher.

| Command | Operation |
|---|---|
| `device restart <id>` | Restart an adopted device |
| `device port-cycle <id> <port>` | Power-cycle a PoE port |
| `client authorize <id>` | Authorize guest access (with optional limits) |
| `client unauthorize <id>` | Remove guest authorization |
| `voucher create <name> --minutes N` | Create a hotspot voucher |
| `voucher delete <id>` | Delete a hotspot voucher |

The flow for all of these is: **gate → dry-run preview → execute → emit result**.

```bash
# Preview a client authorization before committing
ufi client authorize c1d2e3f4 \
  --minutes 60 --data-mb 500 \
  --allow-mutations --dry-run
```

```json
{
  "action": "AUTHORIZE_GUEST_ACCESS",
  "id": "c1d2e3f4",
  "minutes": 60,
  "dry_run": true
}
```

```bash
# Execute it
ufi client authorize c1d2e3f4 \
  --minutes 60 --data-mb 500 \
  --allow-mutations
```

```json
{
  "ok": true,
  "action": "AUTHORIZE_GUEST_ACCESS",
  "id": "c1d2e3f4",
  "minutes": 60
}
```

See [Commands](/commands/) for the full flag reference for each action.

### Tier 2 — Declarative config changes (reviewed-artifact flow)

Config mutations — `network`, `firewall policy`, `firewall zone`, `acl`, `dns policy`, `traffic-list` create/update/delete/reorder — use a more conservative two-step flow. The reason: these changes restructure the network and may be hard to roll back. Immediate execution would mean the thing you reviewed at the keyboard is not exactly the thing that ran later (a classic TOCTOU — time-of-check / time-of-use gap).

The solution is the **reviewed-artifact apply flow**:

1. Run the config write command with `--allow-mutations`. Instead of executing, ufi computes a **12-hex content hash** over the operation, HTTP method, path, and body, then saves the plan to `$XDG_STATE_HOME/ufi/plans/<hash>.json`. You get the plan and the hash back on stdout.
2. Inspect the plan. Share it for review. Put it in a PR. Sleep on it.
3. Run `ufi apply <hash> --allow-mutations` to execute exactly that saved plan. The hash guarantees that what runs is exactly what you previewed — no drift.

```bash
# Step 1: write a network update plan
ufi network update 64a1f3b2 \
  --data '{"name":"IoT VLAN","vlan_id":30}' \
  --allow-mutations
```

```json
{
  "action": "network update",
  "method": "PUT",
  "path": "networks/64a1f3b2",
  "hash": "c3d7e1a04f82",
  "plan": {
    "id": "64a1f3b2",
    "body": { "name": "IoT VLAN", "vlan_id": 30 }
  },
  "dry_run": true,
  "note": "preview only — run `ufi apply c3d7e1a04f82 --allow-mutations` to execute"
}
```

The response carries `"dry_run": true` because no network request has been made yet.

```bash
# Step 2: review the persisted plan file directly (optional)
cat "$XDG_STATE_HOME/ufi/plans/c3d7e1a04f82.json"
```

```json
{
  "hash": "c3d7e1a04f82",
  "op": "network update",
  "method": "PUT",
  "path": "networks/64a1f3b2",
  "body": { "name": "IoT VLAN", "vlan_id": 30 },
  "summary": { "id": "64a1f3b2", "body": { "name": "IoT VLAN", "vlan_id": 30 } },
  "created_at": "2026-06-26T14:30:00Z"
}
```

```bash
# Step 3: execute exactly that plan
ufi apply c3d7e1a04f82 --allow-mutations
```

```json
{
  "ok": true,
  "hash": "c3d7e1a04f82",
  "op": "network update",
  "result": {
    "id": "64a1f3b2",
    "name": "IoT VLAN",
    "vlan_id": 30
  }
}
```

### Why the hash closes the TOCTOU gap

Without a reviewed artifact, an agent or operator could generate a plan, have it modified by a concurrent process or a typo in a re-run, and execute something different from what was approved. The hash is computed over the operation name, HTTP method, URL path, and request body:

```
sha256(op + "\n" + method + "\n" + path + "\n" + body)[:12]
```

Re-running the same config command with the same inputs produces the same hash — so the saved plan is stable. Any change to the body or target produces a different hash, which means a different (non-existent) plan file, which makes `ufi apply` fail with `PLAN_NOT_FOUND`. You cannot accidentally apply a modified plan under the old hash.

:::tip[`--dry-run` on `ufi apply` itself]
`ufi apply <hash> --allow-mutations --dry-run` loads the persisted plan and prints it without executing — a final pre-flight check before the real run.
:::

### Where plans are stored

Plans live at:

```
$XDG_STATE_HOME/ufi/plans/<hash>.json
```

If `XDG_STATE_HOME` is unset, ufi falls back to `~/.local/state/ufi/plans/`. Files are written with mode `0600`. There is no automatic expiry; old plans persist until you remove them manually.

If `$XDG_STATE_HOME/ufi` is not writable, the config write command returns a `PLAN_SAVE_FAILED` error (exit 10).

## Idempotent deletes

`voucher delete` (tier 1) and all declarative `delete` commands (tier 2 via `ufi apply`) are idempotent: if the target resource does not exist, the operation is treated as a **soft success** rather than an error. The response includes `"existed": false` so you can see what happened, but the exit code is still **0**.

```bash
ufi voucher delete nonexistent-id --allow-mutations
```

```json
{
  "ok": true,
  "kind": "voucher",
  "id": "nonexistent-id",
  "existed": false
}
```

This makes delete operations safe to retry and safe to use in scripts that might run more than once.

## Summary

| Scenario | Behaviour | Exit code |
|---|---|---|
| Read command (any) | Always allowed, no gate | 0 (or 3 for empty) |
| Mutating command, no `--allow-mutations` | `MUTATION_BLOCKED` error to stderr | 12 |
| Tier 1 action with `--dry-run` | Prints preview, no network call | 0 |
| Tier 1 action with `--allow-mutations` | Executes immediately | 0 |
| Tier 2 config write with `--allow-mutations` | Saves plan, prints hash; no network call yet | 0 |
| `ufi apply <hash> --allow-mutations` | Executes exactly the saved plan | 0 |
| `ufi apply <hash>` (no `--allow-mutations`) | `MUTATION_BLOCKED` error | 12 |
| `ufi apply <unknown-hash>` | `PLAN_NOT_FOUND` error | 2 |
| Idempotent delete on missing resource | Soft success, `"existed": false` | 0 |

For the full command reference, see [Commands](/commands/). For the declarative config surface in detail — including `--data` input formats and reorder operations — see [Config](/config/). For all exit codes, see [Exit codes](/exit-codes/).
