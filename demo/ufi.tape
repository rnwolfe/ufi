# ufi demo — deterministic & offline, so the GIF never hangs on a console connection.
# Regenerate: `vhs demo/ufi.tape` (needs ufi on PATH: `go build -o ~/.local/bin/ufi ./cmd/ufi`).
Output demo/ufi.gif

Set Shell bash
Set FontSize 16
Set Width 1180
Set Height 720
Set Padding 20
Set Theme "Catppuccin Mocha"

Hide
Type "export PATH=$HOME/.local/bin:$PATH" Enter
Type "unset UNIFI_HOST UNIFI_API_KEY; export NO_COLOR=1; clear" Enter
Show

Type "# ufi — an agent-friendly CLI for Ubiquiti UniFi Network. Read-only by default." Enter Sleep 1s
Type "ufi schema | jq '{conformance, safety}'" Enter Sleep 2.5s

Type "# mutations are gated — a blocked one is a structured error an agent can branch on" Enter Sleep 900ms
Type "ufi device restart ap-living-room --json; echo exit=$?" Enter Sleep 2.5s

Type "# high-stakes config is previewed, then applied by hash (no blind writes)" Enter Sleep 900ms
Type "ufi network update n1 --data @/tmp/ufi-net.json --allow-mutations --json | jq '{action,hash,note}'" Enter Sleep 3s

Type "# missing creds? a structured AUTH_REQUIRED (exit 4) that names the fix" Enter Sleep 900ms
Type "ufi device list --json; echo exit=$?" Enter Sleep 2.5s

Type "# everything self-describes: ufi agent prints the embedded contract" Enter Sleep 900ms
Type "ufi agent | head -n 8" Enter Sleep 2.5s
