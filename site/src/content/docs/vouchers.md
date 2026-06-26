---
title: Vouchers & guest access
description: Create and manage hotspot vouchers, and authorize or revoke guest-client network access from the CLI.
---

UniFi's Hotspot feature lets you hand guests a time-limited code — a *voucher* — that they enter in the captive-portal to get on the network. You can also authorize specific clients directly via the API, which is useful when you already know the client ID (for example, from a kiosk or a booking system).

Both operations are **mutations** and require `--allow-mutations`. Every mutating command also accepts `--dry-run` so you can see exactly what `ufi` will send before it touches the network.

---

## Prerequisites

- `ufi` installed and authenticated — see [/installation/](/installation/) and [/auth/](/auth/).
- A site with the Hotspot feature enabled on the UniFi console.
- Your console's API key exported or stored — see [/auth/](/auth/).

If your console has more than one site, pass `--site <name-or-id>` or set `UNIFI_SITE`. See [/getting-started/](/getting-started/) for details.

---

## Hotspot vouchers

### List vouchers

```bash
ufi voucher list --json
```

Output follows the standard [list envelope](/output/):

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000001",
      "name": "conference-2024",
      "code": "4861409510",
      "created_at": "2024-11-01T09:00:00Z",
      "time_limit_minutes": 480,
      "authorized_guest_limit": 1,
      "authorized_guest_count": 0,
      "activated_at": null,
      "expires_at": null,
      "expired": false
    },
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000002",
      "name": "lobby-guest",
      "code": "3920184756",
      "created_at": "2024-11-02T14:30:00Z",
      "time_limit_minutes": 60,
      "authorized_guest_limit": null,
      "authorized_guest_count": 1,
      "activated_at": "2024-11-02T15:10:00Z",
      "expires_at": "2024-11-02T16:10:00Z",
      "expired": false
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Fields are snake_cased from the API's camelCase (e.g. `timeLimitMinutes` → `time_limit_minutes`). An empty result exits with code 3 (`empty_results`) and emits the envelope with `count: 0` and `items: []`. See [/exit-codes/](/exit-codes/) and [/output/](/output/).

Use `--limit` and `--cursor` / `--page` to page through large voucher pools:

```bash
# First page of 20
ufi voucher list --json --limit 20

# Next page using the opaque cursor from a previous response
ufi voucher list --json --limit 20 --cursor "eyJvZmZzZXQiOjIwfQ=="

# Or navigate by page number (1-based)
ufi voucher list --json --limit 20 --page 3
```

Project down to just the fields you care about with `--select`:

```bash
ufi voucher list --json --select id,name,code,expired
```

`--select` is a client-side dot-path projection applied after the API responds; the API itself returns the full object either way. See [/output/](/output/).

---

### Create vouchers

`voucher create` requires two things: a **name** (a human-readable label duplicated across all generated vouchers) and a **time limit in minutes**. All other flags are optional.

```bash
# Minimal — one voucher, 4-hour session
ufi voucher create "hotel-guest" --minutes 240 --allow-mutations
```

#### Dry run first

Always check the preview before committing. The dry run shows exactly what will be POSTed and exits 0:

```bash
ufi voucher create "hotel-guest" --minutes 240 --dry-run --allow-mutations --json
```

```json
{
  "action": "voucher.create",
  "name": "hotel-guest",
  "minutes": 240,
  "count": 1,
  "dry_run": true
}
```

#### Real run

```bash
ufi voucher create "hotel-guest" --minutes 240 --allow-mutations --json
```

```json
{
  "ok": true,
  "action": "voucher.create",
  "name": "hotel-guest",
  "minutes": 240,
  "count": 1,
  "result": {
    "vouchers": [
      {
        "id": "a1b2c3d4-0000-0000-0000-000000000003",
        "name": "hotel-guest",
        "code": "7730291845",
        "created_at": "2024-11-03T10:00:00Z",
        "time_limit_minutes": 240,
        "authorized_guest_count": 0,
        "expired": false
      }
    ]
  }
}
```

The `code` field is the secret the guest types into the captive portal. Store it or hand it off to your booking system.

#### All create flags

| Flag | API field | Required | Notes |
|---|---|---|---|
| `<name>` (positional) | `name` | yes | Label duplicated across all vouchers in the batch |
| `--minutes N` | `timeLimitMinutes` | yes | 1–1 000 000 minutes |
| `--count N` | `count` | no (default: 1) | 1–1 000 vouchers per call |
| `--guests N` | `authorizedGuestLimit` | no | Max guests per voucher code (omit for unlimited) |
| `--data-mb N` | `dataUsageLimitMBytes` | no | 1–1 048 576 MB; omit for unlimited |

#### Batch example — 10 day-pass vouchers, 2 guests each

```bash
ufi voucher create "day-pass" \
  --minutes 1440 \
  --count 10 \
  --guests 2 \
  --data-mb 2048 \
  --allow-mutations \
  --json
```

This creates 10 vouchers in one API call. Each code can be shared by up to 2 guests, grants 24 hours of access, and caps total data at 2 GB. The timer starts when the *first* guest activates the voucher; any subsequent guests on the same code share the same expiry.

> **Note:** `--rx-kbps` / `--tx-kbps` rate limits are available on the `client authorize` command (see below) but the voucher creation API does not accept them.

#### Without `--allow-mutations`

If you forget the flag, `ufi` blocks the operation and exits 12:

```json
{
  "error": "voucher create is a mutating operation and is blocked by default",
  "code": "MUTATION_BLOCKED",
  "remediation": "re-run with --allow-mutations (add --dry-run to preview)"
}
```

See [/safety-model/](/safety-model/) for the full gate design.

---

### Delete a voucher

```bash
ufi voucher delete <id> --allow-mutations
```

Deletion is **idempotent**: if the voucher ID does not exist the command still exits 0 and reports `"existed": false`. This makes it safe to call from a cleanup loop without checking first.

#### Dry run

```bash
ufi voucher delete a1b2c3d4-0000-0000-0000-000000000001 --dry-run --allow-mutations --json
```

```json
{
  "dry_run": true,
  "action": "voucher delete",
  "id": "a1b2c3d4-0000-0000-0000-000000000001"
}
```

#### Real run — voucher existed

```bash
ufi voucher delete a1b2c3d4-0000-0000-0000-000000000001 --allow-mutations --json
```

```json
{
  "ok": true,
  "kind": "voucher",
  "id": "a1b2c3d4-0000-0000-0000-000000000001",
  "existed": true
}
```

#### Real run — voucher already gone (idempotent soft success)

```json
{
  "ok": true,
  "kind": "voucher",
  "id": "a1b2c3d4-0000-0000-0000-000000000001",
  "existed": false
}
```

---

## Guest client access

Rather than issuing a voucher code, you can authorize or revoke network access for a specific client directly — useful for automated workflows (room-check-in/out, event management, etc.).

You need the client's ID. Get it from `ufi client list`:

```bash
ufi client list --json --select id,name,mac_address
```

Only **guest clients** can be authorized this way. The client must be connected to a network configured as a guest network on the UniFi console.

### Authorize a guest client

```bash
ufi client authorize <client-id> --allow-mutations
```

#### Dry run

```bash
ufi client authorize 8f3e9a00-0000-0000-0000-000000000042 \
  --minutes 120 \
  --dry-run --allow-mutations --json
```

```json
{
  "action": "AUTHORIZE_GUEST_ACCESS",
  "id": "8f3e9a00-0000-0000-0000-000000000042",
  "minutes": 120,
  "dry_run": true
}
```

#### Real run

```bash
ufi client authorize 8f3e9a00-0000-0000-0000-000000000042 \
  --minutes 120 \
  --data-mb 500 \
  --allow-mutations --json
```

```json
{
  "ok": true,
  "action": "AUTHORIZE_GUEST_ACCESS",
  "id": "8f3e9a00-0000-0000-0000-000000000042",
  "minutes": 120,
  "result": {
    "action": "AUTHORIZE_GUEST_ACCESS",
    "granted_authorization": {
      "authorized_at": "2024-11-03T11:00:00Z",
      "authorization_method": "API",
      "expires_at": "2024-11-03T13:00:00Z",
      "data_usage_limit_m_bytes": 500,
      "usage": {
        "duration_sec": 0,
        "rx_bytes": 0,
        "tx_bytes": 0,
        "bytes": 0
      }
    }
  }
}
```

If the client was already authorized, the API cancels the existing session, creates a new one with the new limits, and resets traffic counters. The previous authorization appears as `revoked_authorization` in the result alongside `granted_authorization`.

#### All authorize flags

| Flag | API field | Notes |
|---|---|---|
| `--minutes N` | `timeLimitMinutes` | 1–1 000 000; omit to use the site default |
| `--data-mb N` | `dataUsageLimitMBytes` | 1–1 048 576 MB; omit for unlimited |
| `--rx-kbps N` | `rxRateLimitKbps` | Download rate cap, 2–100 000 kbps |
| `--tx-kbps N` | `txRateLimitKbps` | Upload rate cap, 2–100 000 kbps |

All flags are optional. With none set the site's default time limit applies and there are no data or rate caps.

---

### Unauthorize a guest client

Revoke access and immediately disconnect the client:

```bash
ufi client unauthorize <client-id> --allow-mutations
```

#### Dry run

```bash
ufi client unauthorize 8f3e9a00-0000-0000-0000-000000000042 \
  --dry-run --allow-mutations --json
```

```json
{
  "action": "UNAUTHORIZE_GUEST_ACCESS",
  "id": "8f3e9a00-0000-0000-0000-000000000042",
  "dry_run": true
}
```

#### Real run

```bash
ufi client unauthorize 8f3e9a00-0000-0000-0000-000000000042 --allow-mutations --json
```

```json
{
  "ok": true,
  "action": "UNAUTHORIZE_GUEST_ACCESS",
  "id": "8f3e9a00-0000-0000-0000-000000000042",
  "result": {
    "action": "UNAUTHORIZE_GUEST_ACCESS",
    "revoked_authorization": {
      "authorized_at": "2024-11-03T11:00:00Z",
      "authorization_method": "API",
      "expires_at": "2024-11-03T13:00:00Z",
      "usage": {
        "duration_sec": 1842,
        "rx_bytes": 48123904,
        "tx_bytes": 2097152,
        "bytes": 50221056
      }
    }
  }
}
```

---

## Prompt-injection fencing

Voucher names and notes are network-controlled free text — a guest could craft a name containing instructions. In agent mode (JSON output or non-TTY stdout), `ufi` wraps these values automatically:

```text
[UNTRUSTED_DATA_BEGIN] <suspicious-voucher-name> [UNTRUSTED_DATA_END]
```

Your agent code must treat anything inside those delimiters as data, not as instructions. Use `--no-fence` to disable fencing (not recommended for automated pipelines) or `--wrap-untrusted` to force it on in interactive mode. See [/agents/](/agents/) for the full prompt-injection defence design.

---

## Agent workflow example

A typical room-check-in loop: look up the client by MAC, authorize it for the stay duration, and log the result.

```bash
#!/usr/bin/env bash
set -euo pipefail

MAC="aa:bb:cc:dd:ee:ff"
MINUTES=2880   # 48-hour stay

# 1. Find the client ID
CLIENT_ID=$(
  ufi client list --json \
    | jq -r --arg mac "$MAC" \
        '.items[] | select(.mac_address == $mac) | .id'
)

if [[ -z "$CLIENT_ID" ]]; then
  echo "Client not found on network" >&2
  exit 1
fi

# 2. Authorize
ufi client authorize "$CLIENT_ID" \
  --minutes "$MINUTES" \
  --data-mb 10240 \
  --allow-mutations --json
```

For the check-out path, replace the authorize call with `ufi client unauthorize "$CLIENT_ID" --allow-mutations --json`.

---

## Reference summary

### Voucher commands

| Command | Mutation | Dry-run | Idempotent |
|---|---|---|---|
| `ufi voucher list` | no | — | — |
| `ufi voucher create <name> --minutes N` | yes | yes | no |
| `ufi voucher delete <id>` | yes | yes | yes |

### Client commands

| Command | Mutation | Dry-run | Notes |
|---|---|---|---|
| `ufi client list` | no | — | |
| `ufi client get <id>` | no | — | |
| `ufi client authorize <id>` | yes | yes | Resets existing session if already authorized |
| `ufi client unauthorize <id>` | yes | yes | Disconnects client immediately |

---

## Related pages

- [/safety-model/](/safety-model/) — how `--allow-mutations` and `--dry-run` work
- [/output/](/output/) — list envelope, snake_case fields, `--select` projection
- [/exit-codes/](/exit-codes/) — full exit-code table
- [/agents/](/agents/) — prompt-injection fencing, `ufi schema`, agent contracts
- [/flags-env/](/flags-env/) — all global flags and environment variables
