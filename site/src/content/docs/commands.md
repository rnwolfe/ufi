---
title: Command reference
description: Exhaustive noun-verb command tree for ufi — every subcommand, its arguments, read/mutation status, and a one-line description.
---

Every `ufi` invocation follows a **noun verb** pattern, optionally followed by positional arguments and flags. This page is the dry reference: one section per noun group, covering all subcommands, their arguments, whether they require `--allow-mutations`, and what they do.

For global flags that apply to every command, see [Flags and environment variables](/flags-env/). For the mutation gate and the `--dry-run` → `ufi apply <hash>` flow, see [Safety model](/safety-model/).

---

## Reading this reference

| Column | Meaning |
|---|---|
| **Command** | Full invocation — positional args shown in `<angle brackets>` |
| **Type** | `read` — safe by default; `mutation` — blocked without `--allow-mutations`; `config mutation` — blocked and additionally goes through the plan → apply flow |
| **Description** | What it does |

Commands marked **mutation** emit a `MUTATION_BLOCKED` structured error (exit 12) if `--allow-mutations` is absent. Commands marked **config mutation** additionally do not execute immediately — they produce a persisted plan + 12-hex hash that you later execute with `ufi apply <hash> --allow-mutations`. See [Safety model](/safety-model/).

---

## Info and site

### `ufi info`

| Command | Type | Description |
|---|---|---|
| `ufi info` | read | Return controller version and capabilities (`GET /info`). |

### `ufi site`

| Command | Type | Description |
|---|---|---|
| `ufi site list` | read | List all sites on the console. |

**Example**

```bash
ufi site list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    { "id": "default", "name": "Default", "internal_reference": "default" }
  ],
  "count": 1,
  "nextCursor": null
}
```

---

## Device

The `device` noun operates on adopted devices scoped to a site. Pass `--site` or set `UNIFI_SITE` when the console has more than one site; if unset and there is exactly one site it is used automatically.

| Command | Type | Description |
|---|---|---|
| `ufi device list` | read | List all adopted devices for the active site. |
| `ufi device get <id>` | read | Get full detail for one device by its id. |
| `ufi device stats <id>` | read | Fetch the latest statistics snapshot for a device. |
| `ufi device restart <id>` | **mutation** | Send a `RESTART` action to the device. Supports `--dry-run`. |
| `ufi device port-cycle <id> <port>` | **mutation** | Power-cycle a switch port by its port index (`portIdx`). Supports `--dry-run`. |

**Arguments**

| Argument | Used by | Description |
|---|---|---|
| `<id>` | `get`, `stats`, `restart`, `port-cycle` | Device id (from `ufi device list`). |
| `<port>` | `port-cycle` | Switch port index (integer). |

**Examples**

```bash
# List devices, project to id + name only
ufi device list --json --select id,name

# Inspect a single device
ufi device get a1b2c3d4e5f6 --json

# Preview a restart without executing
ufi device restart a1b2c3d4e5f6 --allow-mutations --dry-run

# Actually restart
ufi device restart a1b2c3d4e5f6 --allow-mutations

# Power-cycle port 3 on a switch
ufi device port-cycle a1b2c3d4e5f6 3 --allow-mutations
```

---

## Client

The `client` noun covers all connected network clients (wired and wireless). Free-text fields — `name`, `hostname`, `note` — are wrapped in `[UNTRUSTED_DATA_BEGIN]…[UNTRUSTED_DATA_END]` by default in agent/JSON mode to prevent prompt injection. See [Safety model](/safety-model/).

| Command | Type | Description |
|---|---|---|
| `ufi client list` | read | List connected clients for the active site. |
| `ufi client get <id>` | read | Get full detail for one client by its id. |
| `ufi client authorize <id>` | **mutation** | Authorize a guest client (action `AUTHORIZE_GUEST_ACCESS`). Supports `--dry-run`. |
| `ufi client unauthorize <id>` | **mutation** | Revoke a guest client's network access (action `UNAUTHORIZE_GUEST_ACCESS`). Supports `--dry-run`. |

**Arguments and flags for `client authorize`**

| Argument / Flag | Required | Description |
|---|---|---|
| `<id>` | yes | Client id. |
| `--minutes <n>` | no | Time limit in minutes (`timeLimitMinutes`). Omit for no cap. |
| `--data-mb <n>` | no | Data usage cap in MB (`dataUsageLimitMBytes`). |
| `--rx-kbps <n>` | no | Download rate limit in kbps (`rxRateLimitKbps`). |
| `--tx-kbps <n>` | no | Upload rate limit in kbps (`txRateLimitKbps`). |

All bandwidth/time flags are optional; omitting them authorizes the client with no explicit caps, subject to the hotspot policy.

**Examples**

```bash
# List clients (paged; default limit 50)
ufi client list --json --limit 20

# Authorize a guest for 60 minutes with a 500 MB cap
ufi client authorize c0ffee112233 \
  --minutes 60 --data-mb 500 \
  --allow-mutations

# Revoke a guest
ufi client unauthorize c0ffee112233 --allow-mutations
```

---

## WiFi

The `wifi` noun lists and inspects WiFi broadcasts (SSIDs). SSID names are operator-set and are not fenced as untrusted.

| Command | Type | Description |
|---|---|---|
| `ufi wifi list` | read | List WiFi broadcasts (SSIDs) for the active site. |
| `ufi wifi get <id>` | read | Get full detail for one WiFi broadcast by its id. |

**Examples**

```bash
ufi wifi list --json
ufi wifi get 7f8a9b0c1d2e --json
```

---

## Voucher

The `voucher` noun manages hotspot vouchers. Both `<name>` and `--minutes` are required by the UniFi API when creating a voucher. `voucher delete` is idempotent: if the target id does not exist the command succeeds silently.

| Command | Type | Description |
|---|---|---|
| `ufi voucher list` | read | List hotspot vouchers for the active site. |
| `ufi voucher create <name>` | **mutation** | Generate one or more vouchers. `--minutes` is required. Supports `--dry-run`. |
| `ufi voucher delete <id>` | **mutation** | Delete a voucher by id (idempotent — missing id is a soft success). Supports `--dry-run`. |

**Arguments and flags for `voucher create`**

| Argument / Flag | Required | Description |
|---|---|---|
| `<name>` | **yes** | Voucher note/name (`name` field). |
| `--minutes <n>` | **yes** | Time limit per voucher in minutes (`timeLimitMinutes`). |
| `--count <n>` | no (default 1) | How many vouchers to generate. |
| `--guests <n>` | no | Authorized guests per voucher (`authorizedGuestLimit`). |
| `--data-mb <n>` | no | Data usage cap per voucher in MB (`dataUsageLimitMBytes`). |

**Examples**

```bash
# Preview before creating
ufi voucher create "lobby-day-pass" --minutes 480 --count 5 \
  --allow-mutations --dry-run

# Create 5 eight-hour vouchers
ufi voucher create "lobby-day-pass" --minutes 480 --count 5 \
  --allow-mutations

# Delete (safe to run even if the voucher is already gone)
ufi voucher delete v0uch3r1d --allow-mutations
```

See [Vouchers](/vouchers/) for a complete workflow guide.

---

## Declarative config commands

The following nouns manage configuration objects: `network`, `firewall`, `acl`, `dns`, `traffic-list`. All write subcommands (`create`, `update`, `delete`, `reorder`) are **config mutations**. They do not execute immediately. Instead they:

1. Validate `--data` as JSON.
2. Compute a plan + 12-character hex hash.
3. Persist the plan at `$XDG_STATE_HOME/ufi/plans/<hash>.json`.
4. Print the plan summary and hash to stdout.

You then execute the plan with `ufi apply <hash> --allow-mutations`.

Config write commands always produce a persisted plan — that is their entire purpose. `--dry-run` has no distinct effect for config write commands; they always save the plan and print the hash. Use `--dry-run` on simple single-target mutations (e.g. `device restart`) to preview without executing. See [Safety model](/safety-model/) for the full reviewed-artifact rationale.

**The `--data` flag** (required on `create` and `update`) accepts:
- A file path: `--data ./net.json`
- Standard input: `--data -`
- Inline JSON: `--data '{"name":"IoT","vlanId":30}'`

The value is validated as JSON before any plan is written.

### `ufi network`

Manages VLAN/LAN networks.

| Command | Type | Description |
|---|---|---|
| `ufi network list` | read | List networks for the active site. |
| `ufi network get <id>` | read | Get one network by id. |
| `ufi network create --data <body>` | **config mutation** | Create a network; produces a plan + hash. Requires `--allow-mutations`. |
| `ufi network update <id> --data <body>` | **config mutation** | Replace a network's configuration; produces a plan + hash. Requires `--allow-mutations`. |
| `ufi network delete <id>` | **config mutation** | Delete a network; produces a plan + hash. Requires `--allow-mutations`. |

**Example**

```bash
# Produce a plan for a new IoT VLAN (nothing executes yet)
ufi network create \
  --data '{"name":"IoT","vlanId":30,"purpose":"vlan"}' \
  --allow-mutations
# → prints: plan hash a3f7c912b804

# Execute exactly that plan
ufi apply a3f7c912b804 --allow-mutations
```

### `ufi firewall`

Manages Zone-Based Firewall (ZBF) policies and zones. ZBF must be enabled on the console; if it is not, read commands return `UNSUPPORTED` (exit 11) with a remediation message pointing to the UI toggle.

#### `ufi firewall policy`

| Command | Type | Description |
|---|---|---|
| `ufi firewall policy list` | read | List firewall policies. |
| `ufi firewall policy get <id>` | read | Get one firewall policy by id. |
| `ufi firewall policy create --data <body>` | **config mutation** | Create a firewall policy; produces a plan + hash. |
| `ufi firewall policy update <id> --data <body>` | **config mutation** | Update a firewall policy; produces a plan + hash. |
| `ufi firewall policy delete <id>` | **config mutation** | Delete a firewall policy; produces a plan + hash. |
| `ufi firewall policy reorder <id>...` | **config mutation** | Reorder policies; positional ids define the desired order. Produces a plan + hash. |

**Arguments for `reorder`**: one or more policy ids as positional arguments in the desired order.

#### `ufi firewall zone`

| Command | Type | Description |
|---|---|---|
| `ufi firewall zone list` | read | List firewall zones. |
| `ufi firewall zone get <id>` | read | Get one firewall zone by id. |
| `ufi firewall zone create --data <body>` | **config mutation** | Create a firewall zone; produces a plan + hash. |
| `ufi firewall zone update <id> --data <body>` | **config mutation** | Update a firewall zone; produces a plan + hash. |
| `ufi firewall zone delete <id>` | **config mutation** | Delete a firewall zone; produces a plan + hash. |

### `ufi acl`

Manages switch ACL rules.

| Command | Type | Description |
|---|---|---|
| `ufi acl list` | read | List ACL rules for the active site. |
| `ufi acl get <id>` | read | Get one ACL rule by id. |
| `ufi acl create --data <body>` | **config mutation** | Create an ACL rule; produces a plan + hash. |
| `ufi acl update <id> --data <body>` | **config mutation** | Update an ACL rule; produces a plan + hash. |
| `ufi acl delete <id>` | **config mutation** | Delete an ACL rule; produces a plan + hash. |
| `ufi acl reorder <id>...` | **config mutation** | Reorder ACL rules; positional ids define the desired order. Produces a plan + hash. |

### `ufi dns policy`

Manages DNS policies.

| Command | Type | Description |
|---|---|---|
| `ufi dns policy list` | read | List DNS policies. |
| `ufi dns policy get <id>` | read | Get one DNS policy by id. |
| `ufi dns policy create --data <body>` | **config mutation** | Create a DNS policy; produces a plan + hash. |
| `ufi dns policy update <id> --data <body>` | **config mutation** | Update a DNS policy; produces a plan + hash. |
| `ufi dns policy delete <id>` | **config mutation** | Delete a DNS policy; produces a plan + hash. |

### `ufi traffic-list`

Manages traffic-matching lists, which can be referenced by firewall policies.

| Command | Type | Description |
|---|---|---|
| `ufi traffic-list list` | read | List traffic-matching lists. |
| `ufi traffic-list get <id>` | read | Get one traffic-matching list by id. |
| `ufi traffic-list create --data <body>` | **config mutation** | Create a traffic-matching list; produces a plan + hash. |
| `ufi traffic-list update <id> --data <body>` | **config mutation** | Update a traffic-matching list; produces a plan + hash. |
| `ufi traffic-list delete <id>` | **config mutation** | Delete a traffic-matching list; produces a plan + hash. |

---

## `ufi apply`

`ufi apply` is a top-level command (not a subcommand of a noun). It loads a previously persisted config plan by its hash and executes it — ensuring the operator has already reviewed the exact payload before any change is made.

| Command | Type | Description |
|---|---|---|
| `ufi apply <hash>` | **mutation** | Execute the persisted plan identified by `<hash>`. Requires `--allow-mutations`. |

**Arguments**

| Argument | Description |
|---|---|
| `<hash>` | The 12-character hex hash printed by a prior config write command. |

Plans are stored at `$XDG_STATE_HOME/ufi/plans/<hash>.json`. Running `ufi apply <hash> --dry-run --allow-mutations` shows the stored plan without executing it — useful to confirm what you're about to run.

```bash
# Inspect the plan without executing
ufi apply a3f7c912b804 --allow-mutations --dry-run

# Execute
ufi apply a3f7c912b804 --allow-mutations
```

```json
{
  "ok": true,
  "hash": "a3f7c912b804",
  "op": "network create",
  "result": { "id": "64e1a2b3c4d5", "name": "IoT", "vlan_id": 30 }
}
```

If the hash is not found, `ufi apply` exits 2 (`usage`) with error code `PLAN_NOT_FOUND` and tells you to re-run the originating config command. If the plan file cannot be read, it exits 10 (`config_error`) with `PLAN_READ_FAILED`. See [Safety model](/safety-model/) for the full TOCTOU rationale.

---

## Auth

| Command | Type | Description |
|---|---|---|
| `ufi auth status` | read | Show which credentials are stored and validate them against the console. Exits 0 even when the console is unreachable (reports `valid: false` + reason). Exits 4 (`auth_required`) only when no API key is configured at all. |
| `ufi auth login` | read | Read an API key from **stdin** (never argv), validate it against `GET /info`, and store it in the OS keyring or a `0600` file. Requires `--host` / `UNIFI_HOST`. |
| `ufi auth logout` | read | Remove locally stored credentials. The key remains active on the console until revoked there. |
| `ufi auth refresh` | read | No-op. API keys do not expire or refresh; this subcommand exists for tooling parity only. |

**Notes**

- Keys are read from stdin to avoid shell-history exposure: `printf '%s' "$MY_KEY" | ufi auth login --host https://192.168.1.1`
- The key is never printed by any command.
- `auth logout` clears only the local store (keyring or `$XDG_CONFIG_HOME/ufi/credentials`); it does not invalidate the key on the UniFi console.
- A UniFi API key is effectively full-admin and bypasses per-admin RBAC — treat it like a root password.

See [Authentication](/auth/) for the credential precedence chain and storage details.

---

## Doctor

| Command | Type | Description |
|---|---|---|
| `ufi doctor` | read | Run a suite of connectivity and configuration checks. Exits 10 (`config_error`) if any check fails; exits 0 if all pass. |

Checks performed:

| Check | What it tests |
|---|---|
| `host` | `UNIFI_HOST` / `--host` is set |
| `api_key` | A key is present; reports its source (`env`, `keyring`, or `file`) |
| `connectivity` | Console is reachable and the key is valid; reports the application version |
| `cred_perms` | Credential file is not group/other readable (only present when a file-backed key is in use) |

```bash
ufi doctor --json
```

```json
{
  "ok": true,
  "checks": [
    { "name": "host", "ok": true, "detail": "https://192.168.1.1" },
    { "name": "api_key", "ok": true, "detail": "present (redacted), source=keyring" },
    { "name": "connectivity", "ok": true, "detail": "console reachable, key valid, version 10.4.57" }
  ]
}
```

---

## Schema

| Command | Type | Description |
|---|---|---|
| `ufi schema` | read | Print the machine-readable command tree, all flags, the exit-code table, the live safety state (`allow_mutations`, `dry_run`, `no_input`), and the Agent CLI Guidelines conformance block. Always emits JSON regardless of `--format`. |

The output shape:

```json
{
  "tool": "ufi",
  "version": "1.2.3",
  "conformance": {
    "spec": "agent-cli-guidelines",
    "version": "0.4.0",
    "level": "Full"
  },
  "commands": { "...": "..." },
  "exit_codes": {
    "ok": 0,
    "mutation_blocked": 12,
    "unsupported": 11
  },
  "safety": {
    "allow_mutations": false,
    "dry_run": false,
    "no_input": false
  }
}
```

The hidden `cloud` stub is **excluded** from `ufi schema` output.

See [Agents](/agents/) for how to wire `ufi schema` into an agent bootstrap flow.

---

## Agent

| Command | Type | Description |
|---|---|---|
| `ufi agent` | read | Print the embedded `SKILL.md` — the full usage contract for AI agents driving `ufi`. |

Setting `UFI_HELP=agent` and passing `-h` / `--help` on any invocation also prints this contract and exits 0.

---

## Version

| Command | Type | Description |
|---|---|---|
| `ufi version` | read | Print the current version. |
| `ufi version --check` | read | Check for a newer release (network call, short timeout, fail-silent). Never auto-updates. |

**Flags for `version`**

| Flag | Description |
|---|---|
| `--check` | Query the latest release and report whether an update is available. Fail-silent if the network is unreachable. |

```bash
ufi version --check --json
```

```json
{
  "current": "1.2.3",
  "latest": "1.3.0",
  "updateAvailable": true,
  "upgrade": "brew upgrade rnwolfe/tap/ufi"
}
```

Set `UFI_NO_UPDATE_CHECK=1` to suppress the human-readable update hint that appears when a newer version is detected. `ufi` never auto-updates; `upgrade` is the command to run with your package manager.

---

## Hidden: `ufi cloud`

`ufi cloud` is a **hidden stub** — not shown in `--help` output or `ufi schema`. Invoking it exits 11 (`unsupported`) with a structured error:

```json
{
  "error": "the Site Manager (cloud) surface is not available in this build — ufi is local-only for now",
  "code": "UNSUPPORTED",
  "remediation": "if you want cloud (api.ui.com) support, please open an issue: https://github.com/rnwolfe/ufi/issues"
}
```

`ufi` targets the UniFi Network **local** Integration API only (`https://{host}/proxy/network/integration/v1`). Site Manager / cloud support is deferred.

---

## Quick reference

```
ufi info
ufi site list

ufi device list
ufi device get <id>
ufi device stats <id>
ufi device restart <id>             [mutation]
ufi device port-cycle <id> <port>   [mutation]

ufi client list
ufi client get <id>
ufi client authorize <id>           [mutation]
ufi client unauthorize <id>         [mutation]

ufi wifi list
ufi wifi get <id>

ufi voucher list
ufi voucher create <name>           [mutation]
ufi voucher delete <id>             [mutation, idempotent]

ufi network list
ufi network get <id>
ufi network create                  [config mutation → plan + hash]
ufi network update <id>             [config mutation → plan + hash]
ufi network delete <id>             [config mutation → plan + hash]

ufi firewall policy list
ufi firewall policy get <id>
ufi firewall policy create          [config mutation → plan + hash]
ufi firewall policy update <id>     [config mutation → plan + hash]
ufi firewall policy delete <id>     [config mutation → plan + hash]
ufi firewall policy reorder <id>…   [config mutation → plan + hash]

ufi firewall zone list
ufi firewall zone get <id>
ufi firewall zone create            [config mutation → plan + hash]
ufi firewall zone update <id>       [config mutation → plan + hash]
ufi firewall zone delete <id>       [config mutation → plan + hash]

ufi acl list
ufi acl get <id>
ufi acl create                      [config mutation → plan + hash]
ufi acl update <id>                 [config mutation → plan + hash]
ufi acl delete <id>                 [config mutation → plan + hash]
ufi acl reorder <id>…               [config mutation → plan + hash]

ufi dns policy list
ufi dns policy get <id>
ufi dns policy create               [config mutation → plan + hash]
ufi dns policy update <id>          [config mutation → plan + hash]
ufi dns policy delete <id>          [config mutation → plan + hash]

ufi traffic-list list
ufi traffic-list get <id>
ufi traffic-list create             [config mutation → plan + hash]
ufi traffic-list update <id>        [config mutation → plan + hash]
ufi traffic-list delete <id>        [config mutation → plan + hash]

ufi apply <hash>                    [mutation — executes a persisted plan]

ufi auth status
ufi auth login
ufi auth logout
ufi auth refresh

ufi doctor
ufi schema
ufi agent
ufi version
ufi version --check

ufi cloud …                         [hidden stub — exits 11 UNSUPPORTED]
```

---

## Related pages

- [Flags and environment variables](/flags-env/) — global flags (`--allow-mutations`, `--dry-run`, `--format`, `--select`, `--limit`, `--cursor`, `--site`, `--insecure`, `--no-fence`, and more)
- [Safety model](/safety-model/) — mutation gate, `--dry-run` previews, and the plan → `apply` reviewed-artifact flow
- [Authentication](/auth/) — API key storage, `auth login`, credential precedence
- [Output](/output/) — list envelope, snake_case field mapping, pagination, `--select` projection
- [Exit codes](/exit-codes/) — all 14 codes with their symbolic names
- [Errors](/errors/) — structured error format `{ error, code, remediation }`
- [Vouchers](/vouchers/) — hotspot voucher workflow guide
- [Agents](/agents/) — wiring `ufi` into an AI agent
