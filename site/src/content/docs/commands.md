---
title: Command reference
description: Every ufi command, its flags, and what it returns. Reads are safe by default; mutations are gated.
---

Reads are **safe by default**; mutations are gated behind `--allow-mutations`. `schema --json`
is the always-current machine-readable source for the full command tree, flags, exit codes, and
live safety state.

## Global flags

| flag | purpose |
|---|---|
| `--json` / `--format json\|plain\|tsv` | output format (data → stdout, everything else → stderr) |
| `--select a,b.c` | project fields (dot-path) |
| `--limit N` | bound list size (default 25) |
| `--cursor` / `--page` | paginate (opaque cursor; `nextCursor` until end-of-results) |
| `--detailed` / `--concise` | richer vs. compact output (concise is the default) |
| `--host` / `UNIFI_HOST` · `--site` / `UNIFI_SITE` | target console + site |
| `--insecure` / `UNIFI_INSECURE` | accept the console's self-signed cert |
| `--allow-mutations` / `--write` | opt in to a state-changing call (else `MUTATION_BLOCKED`, exit 12) |
| `--dry-run` | preview a mutation; for config writes, emit the plan + hash without committing |
| `--no-fence` | disable prompt-injection fencing of network-controlled text |
| `--no-input` | never prompt; fail with exit 13 instead |

## Reads (safe)

```bash
ufi info                          # controller version
ufi site list                     # sites on the console
ufi device list|get|stats <id>    # adopted devices + latest statistics
ufi client list|get <id>          # connected clients
ufi wifi list|get <id>            # WiFi broadcasts (SSIDs)
ufi voucher list                  # hotspot vouchers
ufi network list|get <id>         # networks / VLANs
ufi firewall policy list          # firewall policies   (Zone-Based Firewall)
ufi firewall zone list            # firewall zones      (Zone-Based Firewall)
ufi acl list                      # access-control rules
ufi dns policy list               # DNS policies
ufi traffic-list list             # traffic rules
```

Lists return an envelope `{ schemaVersion, items, count, nextCursor }`; page with `--cursor`.
`--select` projects fields, `--limit` bounds size.

## Mutations (gated single-target)

Each requires `--allow-mutations`; preview first with `--dry-run`.

```bash
ufi device restart <id> --allow-mutations
ufi device port-cycle <id> <port> --allow-mutations          # PoE power-cycle a switch port
ufi client authorize <id> --minutes 60 --allow-mutations     # guest access
ufi client unauthorize <id> --allow-mutations
ufi voucher create "guest-pass" --minutes 1440 --count 5 --allow-mutations
ufi voucher delete <id> --allow-mutations                    # idempotent
```

## Declarative config (reviewed `apply <hash>`)

`network` / `firewall` / `acl` / `dns` / `traffic-list` writes don't run directly — they go
through preview → hash → apply. See [Reviewed config](/config).

```bash
ufi firewall policy create --data @rule.json --dry-run       # → prints a plan + hash
ufi apply <hash> --allow-mutations                           # runs exactly that plan
```

> Firewall commands require **Zone-Based Firewall** enabled on the console; otherwise they
> return `UNSUPPORTED` (exit 11) with a remediation pointing at the UI toggle.

## Auth & ops

```bash
ufi auth login|status|logout      # X-API-KEY: stdin → OS keyring
ufi doctor                        # host / key / connectivity / TLS / clock diagnostics
ufi schema                        # command tree, flags, exit codes, conformance, live safety
ufi agent                         # print the embedded agent guide (SKILL.md)
ufi version [--check]             # installed version; --check pulls GitHub for the latest release
```

> ufi never auto-updates. On the human path it prints a once-a-day upgrade hint to stderr
> (silent for agents; disable with `UFI_NO_UPDATE_CHECK=1`).

## Local-only for now

The Site Manager **cloud** surface (`ufi cloud …`) is deferred — it returns `UNSUPPORTED`
(exit 11) with a pointer to [open an issue](https://github.com/rnwolfe/ufi/issues).

## Exit codes

| code | name | meaning |
|---|---|---|
| 0 | ok | success |
| 2 | usage | usage / parse error (incl. a stale/unknown `apply` hash) |
| 3 | empty | no results (distinct from an error) |
| 4 | auth_required | missing / invalid API key |
| 5 | not_found | target doesn't exist |
| 6 | permission | key lacks permission |
| 7 | rate_limited | throttled |
| 8 | retryable | transient (auto-retried with backoff) |
| 10 | config | configuration problem |
| 11 | unsupported | needs a console feature/legacy API (e.g. Zone-Based Firewall, `cloud`) |
| 12 | mutation_blocked | mutation attempted without `--allow-mutations` |
| 13 | input_required | `--no-input` hit a prompt |
| 130 | cancelled | interrupted |

Errors are `{error, code, remediation}` on stderr. The full, always-current table is
`ufi schema | jq .exit_codes`.
