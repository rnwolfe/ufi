---
title: Reviewed config — apply <hash>
description: How ufi handles high-stakes declarative config — preview produces a plan + hash, and apply <hash> runs exactly that plan, closing the time-of-check/time-of-use gap.
---

Single-target actions (`device restart`, `client authorize`, `voucher create`) are gated by a
simple `--allow-mutations` flag. But **declarative config** — networks, firewall, ACLs, DNS, and
traffic rules — is high-stakes: a wrong firewall rule can lock you out of your own console. So
ufi never executes these writes directly. Instead it uses a **reviewed-artifact** flow:

1. **Preview** with `--dry-run` (the default for any config write) — ufi resolves the request(s),
   computes a stable **content hash**, prints the plan, and persists it under
   `$XDG_STATE_HOME/ufi/plans/<hash>.json`.
2. **Apply** with `ufi apply <hash> --allow-mutations` — ufi runs **exactly** that persisted
   plan, nothing else.

This closes the **time-of-check / time-of-use (TOCTOU)** gap that a blind `--yes` confirmation
opens: what you reviewed is byte-for-byte what runs.

## The flow

```bash
# 1. preview — emits the plan and a hash (no change is made)
ufi firewall policy create --data @rule.json --dry-run
# → plan:
#   + firewall policy "block-iot-to-lan"  (zone IoT → zone LAN, drop)
#   hash: a1b2c3d4e5

# 2. apply exactly that reviewed plan
ufi apply a1b2c3d4e5 --allow-mutations
# → applied. { "schemaVersion": 1, "applied": 1, "hash": "a1b2c3d4e5" }
```

Applying a **stale or unknown** hash is a usage error (exit `2`) — there's no ambiguity about
what would run.

## What goes through apply

The reviewed flow covers create / update / delete / reorder on:

- `ufi network …`
- `ufi firewall policy …` and `ufi firewall zone …`
- `ufi acl …`
- `ufi dns policy …`
- `ufi traffic-list …`

```bash
ufi network create --data @net.json --dry-run        # → plan + hash
ufi apply <hash> --allow-mutations
```

## Zone-Based Firewall requirement

Firewall commands require **Zone-Based Firewall** enabled on the console. On a console where
ZBF is off, the command returns a structured error rather than a confusing API failure:

```json
{ "error": "Zone-Based Firewall is not enabled on this console",
  "code": "UNSUPPORTED",          // exit 11
  "remediation": "Enable Zone-Based Firewall in the UniFi UI, then retry" }
```

An agent should surface this to the human (it needs a UI toggle) rather than treating it as a
bug or retrying.

## Why not just `--yes`?

A blind confirmation evaluates the change at confirm-time, then re-fetches and re-resolves it at
run-time — between those two moments the console state (or the agent's intent) can drift. The
hash pins the **resolved request** at preview time, so `apply` is deterministic: it executes the
reviewed artifact or nothing.
