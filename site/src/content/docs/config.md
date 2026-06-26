---
title: Editing config safely
description: How ufi's reviewed-artifact flow lets you write networks, firewall rules, ACLs, DNS policies, and traffic lists without ever executing a blind mutation.
---

Single-target actions like `device restart` or `client authorize` are gated by `--allow-mutations`
and accept a `--dry-run` preview. **Declarative config** — networks, firewall policies and zones,
ACL rules, DNS policies, and traffic-matching lists — raises the stakes higher: a misplaced
firewall rule can lock you out of the console itself. ufi handles this category differently.

Instead of executing the write and then asking "are you sure?", ufi uses a **reviewed-artifact**
flow:

1. **Write command** — ufi validates your input, resolves the full HTTP request, computes a stable
   content hash, and persists the plan to disk. **No network call is made.**
2. **Review** — you (or your agent) inspect the emitted plan. The hash pins exactly what will run.
3. **`ufi apply <hash> --allow-mutations`** — ufi loads the persisted plan and executes it
   byte-for-byte. No re-evaluation, no re-resolution.

This closes the **TOCTOU** (time-of-check / time-of-use) gap that a simple `--yes` confirmation
leaves open: what you reviewed is exactly what runs.

## The config surface

The following commands go through the reviewed-artifact flow. Read operations (`list`, `get`) are
always safe and execute immediately.

| Command group | Subcommands |
|---|---|
| `ufi network` | `list` `get` `create` `update` `delete` |
| `ufi firewall policy` | `list` `get` `create` `update` `delete` `reorder` |
| `ufi firewall zone` | `list` `get` `create` `update` `delete` |
| `ufi acl` | `list` `get` `create` `update` `delete` `reorder` |
| `ufi dns policy` | `list` `get` `create` `update` `delete` |
| `ufi traffic-list` | `list` `get` `create` `update` `delete` |

`create`, `update`, `delete`, and `reorder` are all config write operations — they require
`--allow-mutations` and go through the plan+hash flow. Running one without `--allow-mutations`
returns a `MUTATION_BLOCKED` error (exit 12) with a remediation hint. See [Safety model](/safety-model/).

## Providing config data with `--data`

Every config write command accepts `--data` to supply the JSON body. Three forms are accepted:

```bash
# 1. File path (strip the "@" prefix, or omit it — both work)
ufi network create --data @network.json
ufi network create --data network.json

# 2. Stdin (pass "-")
cat network.json | ufi network create --data -

# 3. Inline JSON (string that starts with "{" or "[")
ufi network update abc123 --data '{"name":"IoT","vlan_id":20}'
```

The value is **validated as JSON** before the plan is written. If the body is malformed, you get
a `USAGE` error (exit 2) immediately — nothing is persisted.

`--data` is not accepted for `delete` or `reorder`: delete takes only the resource ID as a
positional argument; reorder takes an ordered list of IDs.

## End-to-end example: create a firewall policy

### Step 1 — read the current state

Start with a read. This costs nothing and gives you the IDs you need.

```bash
ufi firewall zone list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    { "id": "z-lan",  "name": "LAN",  "network_ids": ["net-abc"] },
    { "id": "z-iot",  "name": "IoT",  "network_ids": ["net-def"] },
    { "id": "z-wan",  "name": "WAN",  "network_ids": [] }
  ],
  "count": 3,
  "nextCursor": null
}
```

### Step 2 — prepare the config body

Write the policy body to a file, or construct it inline. Here we block IoT from reaching LAN:

```bash
cat > block-iot.json << 'EOF'
{
  "name": "block-iot-to-lan",
  "action": "drop",
  "source_zone_id": "z-iot",
  "destination_zone_id": "z-lan",
  "enabled": true
}
EOF
```

### Step 3 — run the write command (produces a plan, no change yet)

```bash
ufi firewall policy create --data @block-iot.json --allow-mutations
```

```json
{
  "action": "firewall policy create",
  "method": "POST",
  "path": "firewall/policies",
  "hash": "a1b2c3d4e5f6",
  "plan": {
    "body": {
      "name": "block-iot-to-lan",
      "action": "drop",
      "source_zone_id": "z-iot",
      "destination_zone_id": "z-lan",
      "enabled": true
    }
  },
  "dry_run": true,
  "note": "preview only — run `ufi apply a1b2c3d4e5f6 --allow-mutations` to execute"
}
```

The plan is persisted to `$XDG_STATE_HOME/ufi/plans/a1b2c3d4e5f6.json`. Exit 0.

### Step 4 — review, then apply

```bash
# Optionally inspect the persisted plan file
cat ~/.local/state/ufi/plans/a1b2c3d4e5f6.json

# Execute exactly the reviewed plan
ufi apply a1b2c3d4e5f6 --allow-mutations
```

```json
{
  "ok": true,
  "hash": "a1b2c3d4e5f6",
  "op": "firewall policy create",
  "result": {
    "id": "fp-789xyz",
    "name": "block-iot-to-lan",
    "action": "drop",
    "source_zone_id": "z-iot",
    "destination_zone_id": "z-lan",
    "enabled": true
  }
}
```

The API response is snake_cased (camelCase fields from the API are normalized: `sourceZoneId` →
`source_zone_id`). See [Output & pagination](/output/).

## The plan file

Plans are persisted at:

```text
$XDG_STATE_HOME/ufi/plans/<hash>.json
```

If `XDG_STATE_HOME` is unset, the default is `~/.local/state/ufi/plans/`. Each file is written
`0600`. A plan contains the operation name, HTTP method, site-relative path, the serialized body,
a human-readable summary, and a `created_at` timestamp.

The **hash** is a stable 12-character hex digest over the operation name, HTTP method, path, and
body — so re-running the same write command on the same input yields the same hash. Re-running
with different input yields a different hash; both plans are kept on disk.

Referencing an unknown or deleted hash returns `PLAN_NOT_FOUND` (exit 2):

```json
{
  "error": "no persisted plan for hash deadbeef1234",
  "code": "PLAN_NOT_FOUND",
  "remediation": "re-run the config command with --dry-run to produce a plan"
}
```

## Previewing apply itself

You can pass `--dry-run` to `ufi apply` to verify what a hash resolves to without executing it:

```bash
ufi apply a1b2c3d4e5f6 --dry-run
```

```json
{
  "dry_run": true,
  "hash": "a1b2c3d4e5f6",
  "op": "firewall policy create",
  "method": "POST",
  "path": "firewall/policies",
  "plan": {
    "body": { "name": "block-iot-to-lan", "action": "drop", "..." : "..." }
  }
}
```

## Reordering

`firewall policy reorder` and `acl reorder` take an ordered list of IDs as positional arguments
rather than `--data`. They also go through the plan+hash flow:

```bash
# List current policies to get their IDs in the order you want
ufi firewall policy list --json | jq '[.items[].id]'

# Specify the desired order
ufi firewall policy reorder fp-111 fp-789xyz fp-222 --allow-mutations
```

```json
{
  "action": "firewall policy reorder",
  "method": "PUT",
  "path": "firewall/policies/ordering",
  "hash": "b2c3d4e5f601",
  "plan": {
    "order": ["fp-111", "fp-789xyz", "fp-222"]
  },
  "dry_run": true,
  "note": "preview only — run `ufi apply b2c3d4e5f601 --allow-mutations` to execute"
}
```

```bash
ufi apply b2c3d4e5f601 --allow-mutations
```

## Zone-Based Firewall requirement

`ufi firewall policy` and `ufi firewall zone` commands require **Zone-Based Firewall** to be
enabled on the console. This is a UniFi console setting, not a ufi setting.

On a console where ZBF is disabled, the API returns `400` with the code
`api.firewall.zone-based-firewall-not-configured`. ufi classifies this as `UNSUPPORTED` (exit 11)
and adds a concrete remediation:

```json
{
  "error": "Zone-Based Firewall is not enabled on this console",
  "code": "UNSUPPORTED",
  "remediation": "enable Zone-Based Firewall on the console (Settings → Security) to use firewall commands"
}
```

Exit 11 means "this feature requires a prerequisite that is not met" — it is not a transient
error and should not be retried. An agent should surface the remediation message to the operator;
a human needs to toggle the setting in the UniFi UI before firewall commands will work.

Read operations (`firewall policy list`, `firewall zone get`, etc.) return the same error when
ZBF is off — the API does not expose the firewall surface at all until the feature is enabled.

## Deleting a config resource

Delete also goes through the plan+hash flow:

```bash
ufi firewall policy delete fp-789xyz --allow-mutations
```

```json
{
  "action": "firewall policy delete",
  "method": "DELETE",
  "path": "firewall/policies/fp-789xyz",
  "hash": "c3d4e5f60112",
  "plan": {
    "id": "fp-789xyz"
  },
  "dry_run": true,
  "note": "preview only — run `ufi apply c3d4e5f60112 --allow-mutations` to execute"
}
```

```bash
ufi apply c3d4e5f60112 --allow-mutations
```

```json
{
  "ok": true,
  "hash": "c3d4e5f60112",
  "op": "firewall policy delete"
}
```

If the resource no longer exists when `apply` runs, the API returns a `NOT_FOUND` (exit 5).
Unlike `voucher delete` (which is idempotent by design), config deletes report the upstream
result faithfully.

## Quick reference: exit codes for config operations

| Exit code | Meaning | When it occurs |
|---|---|---|
| `0` | OK | Write command produced and persisted a plan; or `apply` succeeded |
| `2` | Usage / plan not found | `--data` is not valid JSON; unknown hash passed to `apply` |
| `4` | Auth required | No API key configured |
| `5` | Not found | Resource ID does not exist (on `apply` for update/delete) |
| `10` | Config error | Plan directory not writable; plan file unreadable |
| `11` | Unsupported | ZBF not enabled (firewall commands); or other feature not configured |
| `12` | Mutation blocked | Ran a write command without `--allow-mutations` |

See [Exit codes](/exit-codes/) for the full table.

## Agent usage notes

In an agent workflow the write+apply split maps naturally to a **propose / confirm / execute**
pattern:

1. Call the config write command → emit the plan to the operator.
2. The operator reviews and confirms.
3. Call `ufi apply <hash> --allow-mutations` → execute exactly the reviewed plan.

The hash is stable across sessions: if the operator is reviewing asynchronously, the plan file
remains on disk until you delete it. You can always `ufi apply <hash> --dry-run` to echo it back
without side effects.

A `MUTATION_BLOCKED` response (exit 12) from a config write command means `--allow-mutations` was
not passed — no plan is computed or persisted. Re-run with the flag to produce the plan and the
apply hint.

For more on the safety model and how agents should handle structured errors, see [For agents](/agents/)
and [Safety model](/safety-model/).
