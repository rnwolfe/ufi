---
title: Installation
description: Install ufi on macOS, Linux, or Windows using Homebrew, the curl installer, go install, or a prebuilt release binary.
---

Get `ufi` on your machine in a few seconds. Pick the method that fits your workflow — all four land the same static binary with no runtime dependencies.

## Homebrew (macOS and Linux) — recommended

```bash
brew install rnwolfe/tap/ufi
```

Homebrew pins a versioned formula, handles future upgrades with `brew upgrade ufi`, and strips the macOS Gatekeeper quarantine attribute automatically so the binary runs without an approval dialog.

## Shell installer (macOS and Linux)

```bash
curl -fsSL https://uficli.sh/install.sh | sh
```

The script:

1. Detects your OS and architecture (`linux`/`darwin` × `amd64`/`arm64`).
2. Fetches the latest release tarball from GitHub over HTTPS.
3. Downloads `checksums.txt` from the same release and **verifies the SHA-256** of the tarball before extracting — it will hard-error and clean up if the checksum does not match.
4. Installs the binary at `$HOME/.local/bin/ufi` (mode `0755`).

The entire script is wrapped in a function that is invoked on the last line, so a truncated download cannot execute a partial script.

### Installer environment variables

| Variable | Default | Purpose |
|---|---|---|
| `UFI_INSTALL_DIR` | `$HOME/.local/bin` | Where the binary is placed |
| `UFI_VERSION` | latest release | Pin a specific version, e.g. `v0.3.1` |

```bash
# Install a pinned version to a custom directory
UFI_VERSION=v0.3.1 UFI_INSTALL_DIR=/usr/local/bin curl -fsSL https://uficli.sh/install.sh | sh
```

If `$UFI_INSTALL_DIR` is not on your `PATH`, the installer prints a reminder showing the exact `export` line to add.

### Review before running

If you prefer not to pipe directly to a shell:

```bash
curl -fsSL https://uficli.sh/install.sh -o install.sh
# read install.sh, then:
sh install.sh
```

## Go install (any platform)

```bash
go install github.com/rnwolfe/ufi/cmd/ufi@latest
```

Requires Go 1.25 or newer. The binary is built with `CGO_ENABLED=0` (pure Go, no C library), so cross-compilation works normally. The version string is embedded from VCS build info when installed this way — `ufi version` will show the commit hash rather than a semver tag, which is expected.

## Prebuilt release binaries

Every release on [github.com/rnwolfe/ufi/releases](https://github.com/rnwolfe/ufi/releases) ships:

- **Tarballs** (`.tar.gz`) for Linux and macOS; **zip** for Windows.
- **Platforms**: `linux`, `darwin`, `windows` × `amd64`, `arm64`.
- **`checksums.txt`** — SHA-256 of every archive.
- **SBOM** (Software Bill of Materials, generated with Syft) per archive.
- **Build-provenance attestation** via GitHub's release workflow.

Archive names follow the pattern `ufi_<version>_<os>_<arch>.tar.gz`, for example:

```text
ufi_0.3.1_linux_arm64.tar.gz
ufi_0.3.1_darwin_amd64.tar.gz
ufi_0.3.1_windows_amd64.zip
```

Download the right archive and `checksums.txt`, verify, then extract:

```bash
# Example: Linux amd64
VERSION=v0.3.1
curl -fsSL "https://github.com/rnwolfe/ufi/releases/download/${VERSION}/ufi_${VERSION#v}_linux_amd64.tar.gz" -o ufi.tar.gz
curl -fsSL "https://github.com/rnwolfe/ufi/releases/download/${VERSION}/checksums.txt" -o checksums.txt
sha256sum --check --ignore-missing checksums.txt
tar -xzf ufi.tar.gz
install -m 0755 ufi /usr/local/bin/ufi
```

## Verify the installation

After any install method, confirm the binary is on your `PATH` and shows a version:

```bash
ufi version
```

```json
{
  "version": "0.3.1"
}
```

Then run the built-in diagnostics to confirm connectivity to your console:

```bash
ufi doctor --json
```

```json
{
  "ok": true,
  "checks": [
    { "name": "host",         "ok": true, "detail": "https://192.168.1.1" },
    { "name": "api_key",      "ok": true, "detail": "present (redacted), source=keyring" },
    { "name": "connectivity", "ok": true, "detail": "console reachable, key valid, version 10.4.57" }
  ]
}
```

If `doctor` is not yet configured (no host or key set), that is expected here — see [Authentication](/auth/) to connect to your console.

## Checking for updates

`ufi` never auto-updates. To see whether a newer release is available:

```bash
ufi version --check
```

```json
{
  "current": "0.3.1",
  "latest": "0.4.0",
  "updateAvailable": true,
  "upgrade": "go install github.com/rnwolfe/ufi/cmd/ufi@latest"
}
```

The check is network-bound with a short timeout and **fail-silent** — if GitHub is unreachable, the command still exits `0` with a `note` field rather than erroring. On the human path (TTY output), `ufi` prints a once-a-day upgrade hint to stderr when a newer release is detected; agents and scripts never see it. Set `UFI_NO_UPDATE_CHECK=1` to suppress the hint entirely.

The `upgrade` field in the JSON output always shows the `go install` command derived from the module's build info, regardless of how the binary was installed. Homebrew users should run `brew upgrade ufi` instead.

## Platform and architecture support

| OS | amd64 | arm64 |
|---|---|---|
| macOS (darwin) | Yes | Yes (Apple Silicon) |
| Linux | Yes | Yes |
| Windows | Yes | Yes |

The shell installer and Homebrew formula cover macOS and Linux only. For Windows, use `go install` or a prebuilt release zip.

## Next steps

- [Authentication](/auth/) — generate an API key and connect `ufi` to your console.
- [Getting Started](/getting-started/) — your first commands and a quick orientation.
- [Flags and environment variables](/flags-env/) — all global flags including `--insecure` for self-signed console certificates.
