---
title: Inspecting your network
description: Read sites, devices, clients, WiFi broadcasts, networks, DNS policies, ACL rules, and traffic-matching lists from the CLI using pagination, field projection, and jq.
---

`ufi` is read-only by default. No flag is required to fetch data — just point it at your console and run a command. This page walks through every read surface: what commands exist, what fields you get, how to slice large result sets with `--limit`/`--cursor`/`--page`, and how to project exactly the fields you need with `--select` and `jq`.

All list reads return the same stable envelope:

```json
{
  "schemaVersion": 1,
  "items": [ ... ],
  "count": 2,
  "nextCursor": null
}
```

Fields are always snake_cased (`macAddress` → `mac_address`, `uptimeSec` → `uptime_sec`, `vlanId` → `vlan_id`). See [Output, pagination & projection](/output/) for the full contract.

---

## Prerequisites

Before issuing any reads you need a host and API key configured. The fastest path:

```bash
export UNIFI_HOST=https://192.168.1.1
export UNIFI_API_KEY=<your-key>
export UNIFI_INSECURE=1   # most local consoles use a self-signed cert
```

Confirm connectivity:

```bash
ufi doctor --json
```

See [Authentication](/auth/) for the keyring-based alternative and credential precedence rules.

---

## Resolving --site

Every site-scoped command (`device`, `client`, `wifi`, `network`, `firewall`, `acl`, `dns`, `traffic-list`) needs to know which site to query.

**Single-site consoles** (most home and small-office setups): ufi resolves the site automatically. No flag needed.

**Multi-site consoles**: if you omit `--site`, ufi lists the available site names and exits with a usage error. Pass `--site` with the site's `id`, `name`, or `internalReference`:

```bash
ufi device list --site "Main Office" --json
ufi device list --site default --json          # internalReference
ufi device list --site 7c6e2a1f-... --json     # UUID id
```

You can also export it as an environment variable so every command picks it up:

```bash
export UNIFI_SITE="Main Office"
ufi device list --json
```

To discover your site names and IDs, run `ufi site list` — it does not require `--site`.

---

## ufi info

`ufi info` fetches the console's metadata — version, capabilities — without needing a site. Useful for verifying connectivity and recording the firmware version in automation output.

```bash
ufi info --json
```

```json
{
  "application_version": "10.4.57",
  "name": "UniFi Network"
}
```

---

## ufi site list

List every site the API key can see. The result is a flat list; there is no `site get` because the sites endpoint only returns the list.

```bash
ufi site list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "7c6e2a1f-3d89-4b12-a05f-d123456789ab",
      "internal_reference": "default",
      "name": "Default"
    },
    {
      "id": "3b1c9e4d-8f22-47a6-b8e0-e987654321cd",
      "internal_reference": "branch",
      "name": "Branch Office"
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID used with `--site` |
| `internal_reference` | Legacy short name used in older UniFi APIs; also accepted by `--site` |
| `name` | Human name; also accepted by `--site` |

---

## ufi device list / get / stats

Devices are the adopted hardware on a site: gateways, switches, and access points.

### List all adopted devices

```bash
ufi device list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "mac_address": "fc:ec:da:11:22:33",
      "ip_address": "192.168.1.1",
      "name": "[UNTRUSTED_DATA_BEGIN] gateway [UNTRUSTED_DATA_END]",
      "model": "UDMPRO",
      "state": "ONLINE",
      "supported": true,
      "firmware_version": "4.0.6",
      "firmware_updatable": false,
      "features": ["gateway", "switching", "accessPoint"],
      "interfaces": ["ports", "radios"]
    },
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "mac_address": "fc:ec:da:44:55:66",
      "ip_address": "192.168.1.2",
      "name": "[UNTRUSTED_DATA_BEGIN] core-switch [UNTRUSTED_DATA_END]",
      "model": "USW-Pro-48-POE",
      "state": "ONLINE",
      "supported": true,
      "firmware_version": "7.0.66",
      "firmware_updatable": true,
      "features": ["switching"],
      "interfaces": ["ports"]
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `device get`, `device restart`, `device port-cycle` |
| `mac_address` | Hardware MAC (snake_cased from `macAddress`) |
| `ip_address` | Current IP on the management network |
| `name` | Operator-assigned name — fenced in agent mode (see below) |
| `model` | Product model code, e.g. `UDMPRO`, `USW-Pro-48-POE`, `U6-LR` |
| `state` | One of: `ONLINE`, `OFFLINE`, `UPDATING`, `ADOPTING`, `GETTING_READY`, `CONNECTION_INTERRUPTED`, `ISOLATED`, `DELETING`, `PENDING_ADOPTION`, `U5G_INCORRECT_TOPOLOGY` |
| `features` | Capabilities of the device: `gateway`, `switching`, `accessPoint` |
| `interfaces` | Available interface groups: `ports`, `radios` |
| `firmware_updatable` | `true` when a newer firmware is available |

The `name` field is fenced with `[UNTRUSTED_DATA_BEGIN] … [UNTRUSTED_DATA_END]` in agent mode (JSON output or non-TTY stdout). Device names are network-controlled free text. See [Output](/output/#prompt-injection-fencing) and [Agents](/agents/) for details.

### Get one device

```bash
ufi device get a1b2c3d4-e5f6-7890-abcd-ef1234567890 --json
```

Returns the full device detail object. The shape is a superset of the list item and includes port and radio detail for devices that support them.

### Device statistics

Fetch live health metrics for one device — CPU, memory, load averages, uplink throughput, and per-interface counters:

```bash
ufi device stats a1b2c3d4-e5f6-7890-abcd-ef1234567890 --json
```

```json
{
  "uptime_sec": 1209612,
  "last_heartbeat_at": "2025-06-26T12:00:00Z",
  "next_heartbeat_at": "2025-06-26T12:00:30Z",
  "load_average1_min": 0.42,
  "load_average5_min": 0.38,
  "load_average15_min": 0.35,
  "cpu_utilization_pct": 12.5,
  "memory_utilization_pct": 34.1,
  "uplink": {
    "rx_bytes_per_second": 45000,
    "tx_bytes_per_second": 12000
  },
  "interfaces": {}
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `uptime_sec` | Seconds since last reboot |
| `cpu_utilization_pct` | Current CPU load, 0–100 |
| `memory_utilization_pct` | Current RAM utilization, 0–100 |
| `load_average1_min` | 1-minute load average |
| `uplink` | Uplink interface throughput counters |

There is no list variant for stats — run per-device with a known ID from `device list`.

---

## ufi client list / get

Clients are end-user devices that are currently connected to the network: wired, wireless, VPN, and Teleport connections.

### List connected clients

```bash
ufi client list --json --limit 20
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "c3d4e5f6-a1b2-3456-cdef-123456789012",
      "type": "WIRELESS",
      "name": "[UNTRUSTED_DATA_BEGIN] ryans-laptop [UNTRUSTED_DATA_END]",
      "mac_address": "aa:bb:cc:dd:ee:ff",
      "ip_address": "192.168.10.42",
      "connected_at": "2025-06-26T08:15:00Z",
      "uplink_device_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "access": {
        "type": "LOCAL"
      }
    },
    {
      "id": "d4e5f6a1-b2c3-4567-def0-234567890123",
      "type": "WIRED",
      "name": "[UNTRUSTED_DATA_BEGIN] homelab-server [UNTRUSTED_DATA_END]",
      "mac_address": "bb:cc:dd:ee:ff:00",
      "ip_address": "192.168.1.50",
      "connected_at": "2025-06-26T00:00:01Z",
      "uplink_device_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "access": {
        "type": "LOCAL"
      }
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `client get`, `client authorize`, `client unauthorize` |
| `type` | Connection type: `WIRED`, `WIRELESS`, `VPN`, `TELEPORT` |
| `name` | Operator-assigned display name — fenced in agent mode |
| `hostname` | Network-reported hostname — fenced in agent mode |
| `mac_address` | Hardware MAC |
| `ip_address` | Assigned IP |
| `connected_at` | ISO 8601 connection timestamp |
| `uplink_device_id` | UUID of the device the client is connected through |
| `access.type` | `LOCAL`, `GUEST`, `VPN`, `TELEPORT` |

The `name`, `hostname`, and `note` fields are fenced in agent mode. All three can be controlled by network participants.

### Get one client

```bash
ufi client get c3d4e5f6-a1b2-3456-cdef-123456789012 --json
```

Returns the full client detail object, which includes access details and connection-type-specific fields (signal strength for wireless, tunnel info for VPN, etc.).

---

## ufi wifi list / get

WiFi broadcasts are the SSIDs that access points are configured to advertise.

### List WiFi broadcasts

```bash
ufi wifi list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "e5f6a1b2-c3d4-5678-ef01-345678901234",
      "type": "STANDARD",
      "name": "[UNTRUSTED_DATA_BEGIN] HomeWifi [UNTRUSTED_DATA_END]",
      "enabled": true,
      "network": {
        "id": "f6a1b2c3-d4e5-6789-f012-456789012345",
        "name": "Default"
      },
      "security_configuration": {
        "type": "WPA2_PERSONAL"
      }
    },
    {
      "id": "a1b2c3d4-e5f6-7890-0123-567890123456",
      "type": "STANDARD",
      "name": "[UNTRUSTED_DATA_BEGIN] IoT [UNTRUSTED_DATA_END]",
      "enabled": true,
      "network": {
        "id": "b2c3d4e5-f6a1-8901-1234-678901234567",
        "name": "IoT"
      },
      "security_configuration": {
        "type": "WPA2_PERSONAL"
      }
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `wifi get` |
| `type` | `STANDARD` or `IOT_OPTIMIZED` |
| `name` | SSID name — fenced in agent mode |
| `enabled` | Whether the SSID is broadcasting |
| `network.id` | The network (VLAN) this SSID is associated with |
| `security_configuration.type` | Security mode, e.g. `WPA2_PERSONAL`, `WPA3_PERSONAL`, `OPEN` |

### Get one WiFi broadcast

```bash
ufi wifi get e5f6a1b2-c3d4-5678-ef01-345678901234 --json
```

Returns full broadcast details including captive portal config, client filtering policies, device tag filters, and per-radio settings.

---

## ufi network list / get

Networks are the VLANs and LAN segments defined on the site.

### List networks

```bash
ufi network list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "f6a1b2c3-d4e5-6789-f012-456789012345",
      "management": "GATEWAY",
      "name": "Default",
      "enabled": true,
      "vlan_id": 1,
      "default": true
    },
    {
      "id": "b2c3d4e5-f6a1-8901-1234-678901234567",
      "management": "GATEWAY",
      "name": "IoT",
      "enabled": true,
      "vlan_id": 20,
      "default": false
    },
    {
      "id": "c3d4e5f6-a1b2-0123-2345-789012345678",
      "management": "SWITCH",
      "name": "Servers",
      "enabled": true,
      "vlan_id": 30,
      "default": false
    }
  ],
  "count": 3,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `network get` or config write commands |
| `management` | `GATEWAY` (IP-managed), `SWITCH` (L2 only), or `UNMANAGED` |
| `name` | Network name |
| `vlan_id` | VLAN ID; always `1` for the default network, `2–4009` for others |
| `default` | `true` for the primary LAN |

Network names are operator-set (you created them), so they are **not** fenced.

### Get one network

```bash
ufi network get f6a1b2c3-d4e5-6789-f012-456789012345 --json
```

Returns the full network detail including DHCP, NAT, IPv6 config, and subnet assignments.

To modify a network, see [Declarative config](/config/).

---

## ufi firewall policy list / get

Firewall policies are the rules within the Zone-Based Firewall.

:::caution[Zone-Based Firewall must be enabled]
These commands require Zone-Based Firewall to be turned on in **Settings → Security** on the console. If it is not enabled, the API returns a `400` and ufi exits with code `11` (`unsupported`) with a remediation message.
:::

```bash
ufi firewall policy list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "d4e5f6a1-b2c3-1234-def0-890123456789",
      "name": "Block IoT Outbound",
      "enabled": true,
      "action": {
        "type": "BLOCK"
      },
      "source": {
        "zones": ["IoT"]
      },
      "destination": {
        "zones": ["WAN"]
      },
      "index": 0
    }
  ],
  "count": 1,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `firewall policy get` or config commands |
| `name` | Policy name |
| `enabled` | Whether the rule is active |
| `action.type` | `ALLOW`, `BLOCK`, or `REJECT` |
| `index` | Evaluation order (lower = higher priority) |

### Get one firewall policy

```bash
ufi firewall policy get d4e5f6a1-b2c3-1234-def0-890123456789 --json
```

---

## ufi firewall zone list / get

Zones are named groups of networks used to define traffic boundaries in the Zone-Based Firewall.

```bash
ufi firewall zone list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "e5f6a1b2-c3d4-2345-ef01-901234567890",
      "name": "Internal",
      "network_ids": [
        "f6a1b2c3-d4e5-6789-f012-456789012345",
        "b2c3d4e5-f6a1-8901-1234-678901234567"
      ],
      "metadata": { "type": "USER_DEFINED" }
    },
    {
      "id": "f6a1b2c3-d4e5-3456-f012-012345678901",
      "name": "IoT",
      "network_ids": [
        "c3d4e5f6-a1b2-0123-2345-789012345678"
      ],
      "metadata": { "type": "USER_DEFINED" }
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID |
| `name` | Zone name |
| `network_ids` | UUIDs of networks assigned to this zone |
| `metadata.type` | `USER_DEFINED` or `SYSTEM_DEFINED`; system-defined zones can only have their network list changed |

:::note
Both `firewall policy` and `firewall zone` require Zone-Based Firewall to be configured on the console (exit code `11` if not).
:::

---

## ufi acl list / get

ACL rules control switch-level Layer 2/3 traffic between network segments.

```bash
ufi acl list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "a1b2c3d4-e5f6-4567-abcd-123456789012",
      "type": "IPV4",
      "name": "Block IoT to Servers",
      "enabled": true,
      "action": "BLOCK",
      "index": 0,
      "source_filter": { "network_ids": ["b2c3d4e5-f6a1-8901-1234-678901234567"] },
      "destination_filter": { "network_ids": ["c3d4e5f6-a1b2-0123-2345-789012345678"] },
      "metadata": { "type": "USER_DEFINED" }
    }
  ],
  "count": 1,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `acl get` or config commands |
| `type` | `IPV4` or `MAC` |
| `name` | Rule name |
| `action` | `ALLOW` or `BLOCK` |
| `index` | Evaluation order (lower = higher priority) |
| `metadata.type` | `USER_DEFINED` rules can be modified; `SYSTEM_DEFINED` rules cannot |

### Get one ACL rule

```bash
ufi acl get a1b2c3d4-e5f6-4567-abcd-123456789012 --json
```

---

## ufi dns policy list / get

DNS policies define custom DNS records and forward-domain rules served by the console's built-in resolver.

```bash
ufi dns policy list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "b2c3d4e5-f6a1-5678-bcde-234567890123",
      "type": "A_RECORD",
      "enabled": true,
      "domain": "nas.home",
      "metadata": { "type": "USER_DEFINED" }
    },
    {
      "id": "c3d4e5f6-a1b2-6789-cdef-345678901234",
      "type": "FORWARD_DOMAIN",
      "enabled": true,
      "domain": "corp.example.com",
      "metadata": { "type": "USER_DEFINED" }
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `dns policy get` or config commands |
| `type` | Record type: `A_RECORD`, `AAAA_RECORD`, `CNAME_RECORD`, `MX_RECORD`, `TXT_RECORD`, `SRV_RECORD`, `FORWARD_DOMAIN` |
| `enabled` | Whether this policy is active |
| `domain` | The domain name this policy applies to |

### Get one DNS policy

```bash
ufi dns policy get b2c3d4e5-f6a1-5678-bcde-234567890123 --json
```

Returns the full record including the resolved address or upstream forwarder.

---

## ufi traffic-list list / get

Traffic-matching lists are named groups of IP addresses or ports, used as reusable references in firewall policies and ACL rules.

```bash
ufi traffic-list list --json
```

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "d4e5f6a1-b2c3-7890-def0-456789012345",
      "type": "IPV4_ADDRESSES",
      "name": "Trusted Subnets"
    },
    {
      "id": "e5f6a1b2-c3d4-8901-ef01-567890123456",
      "type": "PORTS",
      "name": "Web Services"
    }
  ],
  "count": 2,
  "nextCursor": null
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `id` | UUID; pass to `traffic-list get` or config commands |
| `type` | `IPV4_ADDRESSES`, `IPV6_ADDRESSES`, or `PORTS` |
| `name` | List name |

### Get one traffic-matching list

```bash
ufi traffic-list get d4e5f6a1-b2c3-7890-def0-456789012345 --json
```

Returns the full list including all address or port entries.

---

## Pagination

All list commands support the same three flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--limit N` | `50` | Maximum items to return |
| `--cursor <token>` | — | Resume from an opaque cursor returned in `nextCursor` |
| `--page N` | — | Jump to a 1-based page; ignored when `--cursor` is set |

### Cursor-based iteration

```bash
# First page
ufi client list --json --limit 25 > page1.json

# Read the cursor
CURSOR=$(jq -r '.nextCursor' page1.json)

# Next page — pass the cursor verbatim
ufi client list --json --limit 25 --cursor "$CURSOR" > page2.json
```

When `nextCursor` is `null`, all results are exhausted. Cursors are opaque base64 blobs — never construct one by hand.

### Page-number shortcut

```bash
# Third page of 25 clients
ufi client list --json --limit 25 --page 3
```

### Walking all pages in a shell loop

```bash
cursor=""
while true; do
  args=(--json --limit 100)
  [[ -n "$cursor" ]] && args+=(--cursor "$cursor")

  result=$(ufi device list "${args[@]}")
  echo "$result" | jq '.items[]'

  cursor=$(echo "$result" | jq -r '.nextCursor // empty')
  [[ -z "$cursor" ]] && break
done
```

### Empty results

When a query returns zero items, ufi still emits the full envelope and exits with code **3** (`empty_results`). A script can branch on the exit code without parsing JSON:

```bash
ufi client list --json
if [[ $? -eq 3 ]]; then
  echo "No clients connected"
fi
```

---

## Field projection with --select

`--select` keeps only the named fields from each item. Projection is **client-side** — ufi fetches the full response from the API, then strips unwanted fields before writing to stdout.

```bash
ufi device list --json --select id,name,model,state
```

```json
{
  "schemaVersion": 1,
  "items": [
    { "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "model": "UDMPRO",          "name": "[UNTRUSTED_DATA_BEGIN] gateway [UNTRUSTED_DATA_END]",      "state": "ONLINE" },
    { "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901", "model": "USW-Pro-48-POE",  "name": "[UNTRUSTED_DATA_BEGIN] core-switch [UNTRUSTED_DATA_END]",  "state": "ONLINE" }
  ],
  "count": 2,
  "nextCursor": null
}
```

Use dot notation to reach into nested objects:

```bash
ufi client list --json --select id,ip_address,access.type
```

Fields that do not exist in a given item are silently omitted — no error. The envelope keys (`schemaVersion`, `count`, `nextCursor`) are always preserved regardless of `--select`.

Always use the **snake_case** field name in `--select` expressions, because the camelCase-to-snake_case transform runs before projection.

---

## Using jq

`--select` is fast for simple projections. For more powerful transformations, pipe to `jq`.

### Extract all device IDs

```bash
ufi device list --json | jq -r '.items[].id'
```

### Find offline devices

```bash
ufi device list --json | jq '.items[] | select(.state == "OFFLINE") | {id, name, model}'
```

### Count clients by connection type

```bash
ufi client list --json --limit 200 | jq '.items | group_by(.type) | map({type: .[0].type, count: length})'
```

### Pull all networks as a lookup table (id → name)

```bash
ufi network list --json | jq '[.items[] | {key: .id, value: .name}] | from_entries'
```

### Combine select and jq for a TSV report

```bash
ufi device list --json --select id,name,model,state,firmware_version \
  | jq -r '.items[] | [.id, .name, .model, .state, .firmware_version] | @tsv'
```

When you need all pages, loop with cursors (see above) and accumulate with `jq -s`.

---

## Quick reference

| Command | What it reads |
|---------|--------------|
| `ufi info` | Console version and capabilities |
| `ufi site list` | All sites on the console |
| `ufi device list` | Adopted devices (gateways, switches, APs) |
| `ufi device get <id>` | Full detail for one device |
| `ufi device stats <id>` | Live CPU / memory / uplink stats |
| `ufi client list` | Currently connected clients |
| `ufi client get <id>` | Full detail for one client |
| `ufi wifi list` | WiFi broadcasts (SSIDs) |
| `ufi wifi get <id>` | Full detail for one SSID |
| `ufi network list` | VLANs and LAN segments |
| `ufi network get <id>` | Full detail for one network |
| `ufi firewall policy list` | Zone-Based Firewall rules |
| `ufi firewall policy get <id>` | Full detail for one rule |
| `ufi firewall zone list` | Firewall zones |
| `ufi firewall zone get <id>` | Full detail for one zone |
| `ufi acl list` | Switch ACL rules |
| `ufi acl get <id>` | Full detail for one ACL rule |
| `ufi dns policy list` | DNS records and forward-domain policies |
| `ufi dns policy get <id>` | Full detail for one DNS policy |
| `ufi traffic-list list` | Named IP/port matching lists |
| `ufi traffic-list get <id>` | Full detail for one traffic-matching list |

Every list command accepts `--limit`, `--cursor`, `--page`, and `--select`. Every command accepts `--json` for structured output and `--site` to target a specific site.

---

## Related pages

- [Output, pagination & projection](/output/) — the list envelope, cursor mechanics, `--select`, camelCase→snake_case, and fencing details
- [Agents](/agents/) — how fencing works, `ufi schema`, `--no-input`, and the full agent safety contract
- [Authentication](/auth/) — `UNIFI_API_KEY`, keyring storage, credential precedence, `ufi auth status`
- [Safety model](/safety-model/) — read-only default, `--allow-mutations`, `--dry-run`
- [Declarative config](/config/) — how to modify networks, firewall rules, ACLs, DNS, and traffic lists
- [Vouchers](/vouchers/) — hotspot voucher reads and mutations
- [Exit codes](/exit-codes/) — the full stable exit-code table
