# Chronary CLI

The official command-line interface for the [Chronary](https://chronary.ai) calendar-as-a-service API.

## Install

### Shell installer (macOS / Linux)

```bash
curl -sSfL https://chronary.ai/install.sh | sh
```

Detects your OS + architecture, verifies the SHA256 checksum, and installs to `/usr/local/bin/chronary`. Pin a version with `sh -s -- --version 0.1.1` or change the prefix with `sh -s -- --prefix "$HOME/.local"`.

### Homebrew (macOS / Linux)

```bash
brew tap Chronary/tap
brew install chronary
```

### Debian / Ubuntu (`.deb`)

```bash
# replace X.Y.Z and amd64|arm64 as needed
curl -sSfLO https://github.com/Chronary/chronary-cli/releases/latest/download/chronary_X.Y.Z_linux_amd64.deb
sudo dpkg -i chronary_X.Y.Z_linux_amd64.deb
```

### RHEL / Fedora / Amazon Linux (`.rpm`)

```bash
# replace X.Y.Z and amd64|arm64 as needed
sudo rpm -i https://github.com/Chronary/chronary-cli/releases/latest/download/chronary_X.Y.Z_linux_amd64.rpm
```

### GitHub Releases (all platforms)

Download the archive for your platform from the [latest release](https://github.com/Chronary/chronary-cli/releases/latest):

- `chronary_<version>_darwin_amd64.tar.gz` — macOS Intel
- `chronary_<version>_darwin_arm64.tar.gz` — macOS Apple Silicon
- `chronary_<version>_linux_amd64.tar.gz` — Linux x86_64
- `chronary_<version>_linux_arm64.tar.gz` — Linux ARM64
- `chronary_<version>_windows_amd64.zip` — Windows x86_64
- `chronary_<version>_windows_arm64.zip` — Windows ARM64

Verify the checksum against `sha256sums.txt`, extract the archive, and move the `chronary` binary onto your `PATH`.

## Authentication

Export your API key:

```bash
export CHRONARY_API_KEY=chr_sk_live_...
```

Or sign in interactively:

```bash
chronary auth login
```

## Quick start

```bash
chronary agents list
chronary calendars create --name "Team calendar"
chronary events create --calendar cal_... --title "Standup" --start 2026-05-01T09:00:00Z --end 2026-05-01T09:15:00Z
chronary availability get --calendar cal_... --start 2026-05-01 --end 2026-05-07
```

Run `chronary --help` for the full command surface.

## Configuration

By default the CLI talks to `https://api.chronary.ai/v1`. Override with:

```bash
export CHRONARY_BASE_URL=https://api.example.com/v1
```

Output format:

```bash
chronary agents list --output json   # machine-readable
chronary agents list --output table  # default, human-readable
```

## Documentation

Full docs, including every command, flag, and error code:

- [docs.chronary.ai](https://docs.chronary.ai)
- [API reference](https://docs.chronary.ai/api-reference/overview)
- [CLI quickstart](https://docs.chronary.ai/getting-started/cli-quickstart)

## Issues and feedback

Please file issues at [github.com/Chronary/chronary-cli/issues](https://github.com/Chronary/chronary-cli/issues).

## License

Apache-2.0. See [LICENSE](./LICENSE).
