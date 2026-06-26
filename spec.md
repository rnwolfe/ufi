# spec.md — ufi

> The build spec for an agent-focused CLI. Written by `cli-plan`; consumed by `cli-scaffold`,
> `cli-implement`, and `cli-publish`. Keep it current — it is the single source of truth.
>
> Aligned to `~/dev/agent-cli-factory/references/contract.md` and
> `references/research/blueprint-go.md` (the authoritative factory standard).

## Target
- **Service**: UniFi home network (Ubiquiti UniFi OS console — e.g. UDM / UDR / UCG / Cloud
  Key Gen2+ / UniFi OS Server) and the UniFi cloud fleet (unifi.ui.com).
- **Surface**: **Official API only** (operator's choice). Two official APIs:
  - **Local Network Integration API** (primary): `https://{console}/proxy/network/integration/v1/`
    — stateless, `X-API-KEY` header, no login/logout. The agent-hot path.
  - **Site Manager cloud API** (secondary): `https://api.ui.com/v1/` — `X-API-KEY` from
    unifi.ui.com. Cross-host/fleet reads + ISP/internet-health metrics.
  - The legacy "classic" controller API (`/proxy/network/api/s/{site}/…`, `/cmd/stamgr`,
    `/rest/portforward`) is **explicitly out of scope** — it is unofficial/reverse-engineered.
    Only the two capabilities still missing from the official API (classic **port-forward** CRUD
    and client **block/kick**) would require it; those are **documented as not-yet-available via
    the official API** rather than implemented against the brittle legacy path. (Firewall, WLAN,
    network, ACL, and DNS writes are now first-class in the official v1 API — see Config surface.)
- **Verified against live hardware** (2026-06-26): UniFi Network **10.4.57**, OpenAPI **3.1.0**.
  The console serves its own spec at `https://{console}/proxy/network/api-docs/integration.json`
  (note: NOT under `/integration/v1/`) — this is the single source of truth; a copy is committed
  as `integration-openapi.json`. Integration base path is `/proxy/network/integration/v1/`.
- **Rate limits / pagination**:
  - Site Manager: **10,000 req/min**, returns `429` + `Retry-After` on overflow → honor it
    (auto-retry with backoff).
  - Local Integration: no documented hard limit; be polite, single in-flight by default.
  - Pagination is **offset-based** with plain params (verified on-device): `limit` (page size,
    **default 25**), `offset` (default 0), `filter` (RSQL-style server filter). The list envelope
    is `{ offset, limit, count, totalCount, data:[…] }`; page until `offset + count >= totalCount`.
    `--limit` maps to `limit`; the client transparently pages when `--limit` exceeds one page.
    **There is NO `attrs`/field-projection param** on the Integration API — `--select` must be
    applied client-side. (The old `_start`/`_limit`/`_sort`/`attrs` scheme is the *legacy* classic
    API, not this one.)
- **ToS / risk**:
  - **Low breakage risk** — these are Ubiquiti's officially supported, versioned APIs
    (Integration `v1`, Site Manager `v1`). This is the main reason for choosing official-only.
  - **Write surface is LIVE and broad** (verified on-device, corrects the earlier "read-mostly"
    assumption): the Integration v1 API already exposes documented mutations — device `RESTART`
    (`POST …/devices/{id}/actions`), switch port `POWER_CYCLE`
    (`POST …/devices/{id}/interfaces/ports/{portIdx}/actions`), client `AUTHORIZE_GUEST_ACCESS` /
    `UNAUTHORIZE_GUEST_ACCESS` (`POST …/clients/{id}/actions`), voucher create + delete
    (per-id and bulk), device adopt/remove, and **full CRUD** on WLANs (`/wifi/broadcasts`),
    networks (`/networks`), firewall policies + zones, ACL rules (+ ordering), DNS policies, and
    traffic-matching lists. Action bodies use a `discriminator` on an `action` enum — only the
    enum values present in the on-device spec are valid (currently: device→`RESTART`,
    port→`POWER_CYCLE`, client→`AUTHORIZE_GUEST_ACCESS`/`UNAUTHORIZE_GUEST_ACCESS`). Never guess
    enum values; read them from `integration-openapi.json`.
  - **Credential sensitivity — state loudly**: a UniFi API key is effectively
    **admin-/full-access** to the console (it bypasses per-admin permission scoping — see
    home-assistant/core#149762). Treat it like a root password: keyring storage, never logged,
    redacted in `--verbose`/error output, and `doctor` must warn if it's sitting in a
    world-readable env/file.

## Language & framework
- **Language**: **Go**
- **Rationale (SDK gravity > distribution > performance)**: no SDK forces another language and
  the official surface is a plain `X-API-KEY` REST API (direct HTTP is trivial); Go is the
  factory default and gives the best single-binary + near-zero cold-start story for an agent
  hot-loop, with the strongest modern SDK option (`jonshaffer/go-unifi`) available if typed
  models help.
- **Framework**: **kong**
- **SDK/library used**: **direct HTTP** with hand-typed response structs (keeps the JSON output
  contract under our control and avoids coupling to the legacy API). May borrow model shapes
  from `github.com/jonshaffer/go-unifi` (supports API-key auth, TLS config, transparent
  pagination) as a reference, not a hard dependency.
- **Blueprint**: `~/dev/agent-cli-factory/references/research/blueprint-go.md` — kong + `internal/`-only
  layout, `main()→run(args,stdin,stdout,stderr)`, `//go:embed SKILL.md`, `99designs/keyring`,
  goreleaser v2 (`homebrew_casks`), testscript + `invopop/jsonschema` golden gate.
- **Language-specific gotchas to honor**: UniFi OS consoles serve a **self-signed TLS cert** by
  default → need a documented `--insecure` / `UNIFI_INSECURE=1` escape hatch (off by default;
  loud warning when on) plus optional CA-pinning. Ship static binaries (CGO disabled) for clean
  cross-compilation.

## Auth
- **Model**: **API key** via `X-API-KEY` header. Two independent keys/scopes:
  - **Local**: generated in the console UI → Settings → Control Plane → Integrations → API Keys.
    Paired with the console host/IP.
  - **Cloud (Site Manager)**: generated at unifi.ui.com → API.
- **Agent-completable path**: ✅ fully. No browser-only OAuth. The agent (or human) generates the
  key once in the UI and runs `ufi auth login` (or sets `UNIFI_API_KEY` / `UNIFI_HOST`). All
  subsequent calls are stateless header auth — no session, no refresh, no interactive step.
  **Verified end-to-end against live hardware** (Network 10.4.57 @ 192.168.1.1): the operator's
  key authenticated `/info`, `/sites`, `/devices`, `/clients`, `/wifi/broadcasts`, `/hotspot/vouchers`.
- **Username/password is deliberately NOT supported.** It only authenticates the *legacy* classic
  controller API (cookie session to `/api/auth/login`), which this tool treats as out of scope.
  The official Integration + Site Manager APIs are `X-API-KEY`-only, so a username/password would
  add an unofficial, brittle auth path for zero gain. If a future need forces a legacy-only
  endpoint, that is a separate, explicitly-flagged decision — not the default.
- **Secret storage**: OS keyring + `0600` XDG fallback (`$XDG_CONFIG_HOME/ufi/credentials`).
  Env vars (`UNIFI_API_KEY`, `UNIFI_HOST`, `UNIFI_CLOUD_API_KEY`, `UNIFI_INSECURE`) override.
- **Subcommands**: `auth login` (prompt/paste key + host, validate against `/info`) ·
  `auth status` (which keys are stored, which console, validity) · `auth logout` (clear stored
  creds). `auth refresh` is **N/A** (keys don't expire/refresh) — alias it to a no-op that
  prints guidance. `doctor` validates connectivity, TLS, key validity, and clock.

## Command surface (noun-verb)
Service-namespaced. Local Integration API unless marked **(cloud)**.

> **Wire vs. output naming**: the Integration API returns **camelCase** on the wire
> (`macAddress`, `ipAddress`, `connectedAt`, `uptimeSec`, `cpuUtilizationPct`). Our JSON output
> contract normalizes to **snake_case** (`mac`, `ip`, `connected_at`, `uptime_s`, `cpu_pct`) —
> `cli-implement` owns the wire→output mapping; the snake_case names below are the stable
> append-only contract. Verified on-device: live site has 5 devices, 56 clients (25/page), 2 SSIDs.
>
> **Two tiers, both in scope for this build:**
> - **Core** — read-first nouns + low-stakes single-target actions (the table below).
> - **Config** — the now-official declarative *config* surface (`network`, `firewall policy|zone`,
>   `acl`, `dns policy`, `traffic-list` CRUD). These are higher-stakes structured writes and MUST
>   use the reviewed-artifact `--dry-run → ufi apply <hash>` pattern (contract §2, §9) — see the
>   Config command table and "Declarative config mutations" below.

| Command | Read/Mutation | Description | Key output fields |
|---|---|---|---|
| `ufi auth login` | mutation (local creds) | Store + validate API key & console host | `console`, `version`, `key_valid` |
| `ufi auth status` | read | Show stored creds / validity | `console`, `has_local_key`, `has_cloud_key`, `valid` |
| `ufi auth logout` | mutation (local creds) | Remove stored creds | `cleared` |
| `ufi doctor` | read | Connectivity, TLS, key, version, clock checks | `checks[]{name,ok,detail}` |
| `ufi info` | read | Controller version & capabilities (`/info`) | `application_version`, `capabilities` |
| `ufi site list` | read | List sites (`/sites`) | `id`, `name`, `description` |
| `ufi device list` | read | Devices for a site (`/sites/{id}/devices`) | `id`, `name`, `model`, `mac`, `ip`, `state`, `adopted`, `uptime_s` |
| `ufi device get <id>` | read | Single device detail | full device object |
| `ufi device stats <id>` | read | Latest device statistics (`/devices/{id}/statistics/latest`) | `cpu_pct`, `mem_pct`, `uptime_s`, `load_1m/5m/15m`, `uplink`, `interfaces[]`, `last_heartbeat_at` (per on-device shape) |
| `ufi device restart <id>` | **mutation** | POST action `RESTART` (gated) | `id`, `action`, `accepted` |
| `ufi device port-cycle <id> <port>` | **mutation** | Power-cycle a switch port (gated) | `id`, `port`, `action`, `accepted` |
| `ufi client list` | read | Connected clients (`/sites/{id}/clients`) | `id`, `name`, `hostname`, `mac`, `ip`, `network`, `connected_since`, `tx_bytes`, `rx_bytes`, `is_guest`, `is_wired` |
| `ufi client get <id>` | read | Single client detail | full client object |
| `ufi client authorize <id>` | **mutation** | `AUTHORIZE_GUEST_ACCESS` action (gated); opt flags `--minutes`, `--data-mb`, `--rx-kbps`/`--tx-kbps` | `id`, `action`, `accepted` |
| `ufi client unauthorize <id>` | **mutation** | `UNAUTHORIZE_GUEST_ACCESS` action (gated) | `id`, `action`, `accepted` |
| `ufi wifi list` | read | Broadcast/SSID info (`/wifi/broadcasts`) | `id`, `ssid`, `enabled`, `band`, `security`, `hidden` |
| `ufi wifi get <id>` | read | Single SSID detail | full broadcast object |
| `ufi voucher list` | read | Hotspot vouchers (`/hotspot/vouchers`) | `id`, `code`, `quota`, `duration_min`, `used`, `note`, `created_at` |
| `ufi voucher create` | **mutation** | Generate voucher(s) (gated) | `created[]{id,code,quota,duration_min}` |
| `ufi voucher delete <id>` | **mutation** | Delete a voucher (gated) | `id`, `deleted` |
| `ufi cloud host list` | read **(cloud)** | UniFi OS hosts on account (`/hosts`) | `id`, `name`, `hardware`, `state`, `ip` |
| `ufi cloud site list` | read **(cloud)** | Sites across all hosts (`/sites`) | `id`, `host_id`, `name` |
| `ufi cloud device list` | read **(cloud)** | Devices across all sites (`/devices`) | `id`, `host_id`, `name`, `model`, `state` |
| `ufi cloud isp-metrics` | read **(cloud)** | Internet-health / ISP metrics | `host_id`, `latency_ms`, `download_mbps`, `upload_mbps`, `uptime_pct`, `window` |

**Genuine official-API gaps (verified absent from on-device v1 spec — documented, not faked):**
client **block/unblock/kick** (only guest authorize/unauthorize exists; no `BLOCK` action) and
classic **port-forward** CRUD (no `/portforward` resource; NAT is expressed via firewall/network
config instead). Commands touching these exit `11 UNSUPPORTED` with a structured error pointing
to the limitation. *(Note: firewall, WLAN, network, ACL, and DNS CRUD are NO LONGER gaps — they
moved into the official v1 API and are specced as the Config command surface below.)*

### Config command surface (declarative CRUD — `apply <hash>` gated)
All under the local Integration API. Create/update/delete take a JSON config body via
`--data @file.json` or stdin (matching the on-device schema in `integration-openapi.json`).
Mutations here do NOT execute directly: `--dry-run` (default for any config write) prints the
plan + a content hash; `ufi apply <hash>` commits exactly that plan.

| Command | Read/Mutation | Endpoint | Notes / key fields |
|---|---|---|---|
| `ufi network list\|get` | read | `/networks`, `/networks/{id}` | `id`, `name`, `enabled`, `vlan_id`, `management`, `default` |
| `ufi network create\|update\|delete` | **mutation (config)** | `POST/PUT/DELETE /networks` | VLAN/LAN config body |
| `ufi firewall policy list\|get` | read | `/firewall/policies` | ⚠ requires Zone-Based Firewall enabled (see below) |
| `ufi firewall policy create\|update\|delete` | **mutation (config)** | `POST/PUT/PATCH/DELETE /firewall/policies` | rule config body |
| `ufi firewall policy reorder` | **mutation (config)** | `PUT /firewall/policies/ordering` | ordered id list |
| `ufi firewall zone list\|get\|create\|update\|delete` | read / **mutation (config)** | `/firewall/zones` | ⚠ ZBF-gated |
| `ufi acl list\|get\|create\|update\|delete` | read / **mutation (config)** | `/acl-rules` | + `acl reorder` → `/acl-rules/ordering` |
| `ufi dns policy list\|get\|create\|update\|delete` | read / **mutation (config)** | `/dns/policies` | `domain`, `type`, `ipv4_address`, `ttl_s`, `enabled` |
| `ufi traffic-list list\|get\|create\|update\|delete` | read / **mutation (config)** | `/traffic-matching-lists` | match-list config body |

**Zone-Based Firewall caveat (verified on-device):** `firewall/policies` and `firewall/zones`
return `400 { code: "api.firewall.zone-based-firewall-not-configured" }` on a console not running
ZBF. `ufi` must map this to a structured error (`code: UNSUPPORTED`/`CONFIG`, exit `11`/`10`) with
remediation ("enable Zone-Based Firewall in the console"), NOT a bare 400 or empty list — silent
emptiness here is a correctness bug for an agent.

## Exit codes
Per contract §4 (stable, documented in `schema --json`). Target-specific addition: `11`.

| Code | Machine string | Meaning for `ufi` |
|---|---|---|
| 0 | `OK` | Success |
| 1 | `GENERIC` | Unexpected error |
| 2 | `USAGE` | Bad flags/args/parse |
| 3 | `EMPTY` | Query succeeded but returned no results |
| 4 | `AUTH_REQUIRED` | Missing/invalid API key → names `ufi auth login` |
| 5 | `NOT_FOUND` | site/device/client/voucher id not found |
| 6 | `PERMISSION_DENIED` | `403` — API key lacks required scope |
| 7 | `RATE_LIMITED` | `429` — honor `Retry-After`, retry, then fail with this |
| 8 | `TRANSIENT` | Retryable: console unreachable, TLS handshake, 5xx |
| 10 | `CONFIG` | Bad/missing config (no host set, malformed URL) |
| 11 | `UNSUPPORTED` | **(ufi)** Operation unavailable on this console — needs the unofficial legacy API (port-forward, client block/kick) OR a console feature that's off (e.g. Zone-Based Firewall) |
| 12 | `MUTATION_BLOCKED` | Mutation attempted without `--allow-mutations` |
| 13 | `INPUT_REQUIRED` | `--no-input` set but a prompt was needed |
| 130 | `CANCELLED` | SIGINT |

## Output schema
Stable, **append-only** JSON (contract §10). Carries a top-level `schemaVersion`. List
commands return an envelope `{ "schemaVersion": N, "items": [...], "count": N, "nextCursor":
<opaque-string|null> }`; omitting `nextCursor` signals end-of-results. The UniFi APIs page by
offset (`offset`/`limit`, envelope `{offset,limit,count,totalCount,data}`) internally — the
cursor is an **opaque** token (base64 offset), not a raw integer, so the wire format stays
stable if pagination changes.

**Field naming (as shipped):** output mirrors the official API's field set, transformed
**generically** from the wire's camelCase to snake_case (`macAddress`→`mac_address`,
`uptimeSec`→`uptime_sec`, `connectedAt`→`connected_at`, `vlanId`→`vlan_id`). The "Key output
fields" names in the command tables above are *illustrative* of each resource, not hand-curated
renames — the actual keys are the API's own, snake_cased — which keeps the mapping robust and
append-only as Ubiquiti adds fields. `schema --json` emits the full command tree, every
flag, the exit-code table, and live safety state (`allow_mutations`, `dry_run`, `no_input`),
reflected from the kong grammar so it can't drift. New fields may be added; existing fields are
never removed or retyped (a `schema --json` golden test is a required CI gate).

**Structured errors** (contract §3): `{ "error": <msg>, "code": "<MACHINE_STRING>",
"remediation": <next step> }` to **stderr** (stdout stays clean), then exit with the mapped
code above.

## Universal contract surface (provided by scaffold — confirm no conflicts)
`--format json|plain|tsv` (+ `--json` alias) · `--allow-mutations`/`--write` · `--dry-run` ·
`--yes`/`--force` · `--no-input` · `--limit` (default 50) · `--page`/`--cursor` · `--select a,b.c`
· `--concise` (default) / `--detailed` · `--wrap-untrusted` · `schema --json` · `agent` ·
example-led `--help` + `UFI_HELP=agent` terse mode.

**Conflict check:** none. `ufi` adds only `--host`/`UNIFI_HOST`, `--insecure`/`UNIFI_INSECURE`,
and `--cloud` (route a read through Site Manager instead of local). `--limit`/`--cursor` map to
the API's `limit`/`offset`; `--select` is **client-side only** (the Integration API has no
field-projection param). **Two mutation tiers** (both call `GuardMutation()` first — default-deny → `12 MUTATION_BLOCKED`):
- **Core actions** (`device restart`, `device port-cycle`, `client authorize|unauthorize`,
  `voucher create|delete`) are single-target, low-stakes → global gate + `--dry-run` (print the
  intended POST/DELETE + named target, perform nothing, exit 0) suffice.
- **Declarative config** (`network`/`firewall`/`acl`/`dns`/`traffic-list` create|update|delete|
  reorder) is high-stakes → **reviewed-artifact `apply <hash>`** (§2): `--dry-run` (default for
  config writes) computes the resolved request(s) + a stable content hash, prints the plan, and
  persists it under `$XDG_STATE_HOME/ufi/plans/<hash>.json`. A separate top-level
  `ufi apply <hash> --allow-mutations` executes **only** that exact persisted plan, closing the
  TOCTOU gap a blind `--yes` opens. `apply` of a stale/unknown hash → `2 USAGE`.

**Upstream error mapping (verified shape):** the Integration API returns errors as
`{ statusCode, statusName, code, message, timestamp, requestPath, requestId }`. `cli-implement`
maps `code`/`statusCode` → our exit-code table and surfaces `message` as the structured-error
`error`, the upstream `code` in `remediation` context, and never leaks `requestId` into stdout.

## Distribution
- **Targets**: `go install` · Homebrew tap (`brew install rnwolfe/tap/ufi`) · `goreleaser`
  release binaries for darwin/linux × amd64/arm64 (static, CGO off).
- **Trial path (human)**: `brew install` or download a prebuilt binary; Go devs `go install`.
- **Agent hot-loop path**: prebuilt static binary on `$PATH` — near-zero cold start, stateless
  API-key auth, no daemon.

## Publish
- **Flag**: **full**
- **If full**: docs site (starlight-docs) · doc content (harvest-docs) · release (release skill) ·
  README + VHS demo · hygiene files (LICENSE/CONTRIBUTING/CI/issue templates) · discoverability
  targets (Homebrew tap, `topics`, Show HN).
- **Competitive note (must be addressed in README/positioning) — landscape is NOT empty:**
  `hyperb1iss/unifly` (Rust, Apache-2.0, **207★, actively developed**) occupies the "elegant
  UniFi CLI + TUI for humans and agents" niche and even ships an agent skill; `ClifHouck/unified`
  (Go, MIT) exists; `jonshaffer/go-unifi` (Go SDK, MPL-2.0) supports the Integration API +
  `X-API-KEY` and is the best **structural reference** for the v1 shapes. A field of **UniFi MCP
  servers** also now targets LLM-driven management (`enuno/unifi-mcp-server`, `unifi-network-mcp`
  on PyPI, unifimcp.com). `ufi`'s defensible wedge is the formal **agent-CLI contract** none of
  them fully meet: machine-discoverable `schema --json` (with a `conformance` block), an
  append-only stable JSON output contract, default-deny `--allow-mutations` + `--dry-run` (and
  reviewed-artifact `apply <hash>` for Phase-2 config), exit-code rigor, and prompt-injection
  fencing of network-supplied free text — delivered as a **single static binary** (vs. an MCP
  daemon), official-API-only for stability. Lead with that contrast or the project reads as a
  redundant clone. **Verdict: BUILD.**

## Prompt-injection surface
Several read commands return **free text the network's own devices/users control** — a hostile
guest could name their device `Ignore previous instructions and …`. Per contract §8, these
fields are fenced as untrusted, **default-ON in agent mode** (JSON output or non-TTY stdout),
disablable with **`--no-fence`**: the carriers are `client.name`/`hostname`/`note`,
`device.name`, `wifi` SSID `name`, and `voucher` `name`/`note` (`client list|get`,
`device list|get`, `wifi list|get`, `voucher list`). Each value is wrapped
`[UNTRUSTED_DATA_BEGIN] … [UNTRUSTED_DATA_END]`, never emitted as bare text an agent might read
as instructions. (Operator-set fields like `site.name`/`network.name` are NOT fenced.)
