# Copilot Instructions

## Build & Run

```bash
go build -o validation .
go run validation.go [config.json]   # config.json is the default if no arg given
```

Dependencies:
```bash
go mod tidy
go mod download
```

## Architecture

This is a single-file Go CLI tool (`validation.go`) that batch-validates OpenVPN `.ovpn` configuration files by actually attempting to connect using the system `openvpn` binary, then copying valid files (ranked by score) to an output folder.

**Flow:**
1. Parse `config.json` (or a path provided as CLI arg) → populate global vars
2. `cleanFolder()` — wipe and recreate the target output folder
3. `getFiles()` — walk source folder, collect all `.ovpn` files
4. `removeDup()` — deduplicate by `connectInfo` key, keeping the most recently modified file
5. Fan-out across `thread` goroutines via buffered channels `ic`/`oc`
6. Each goroutine runs `openvpn` with a timeout; success is determined by checking if the third-to-last line of stdout contains `validstr`
7. Valid files are sorted by score (descending) and copied to target folder with a rank prefix: `{rank}_{country}.ovpn`

## Config (`config.json`)

| Key | Description |
|-----|-------------|
| `openvpn` | Absolute path to openvpn binary |
| `source folder` | Directory to scan for `.ovpn` files (recursive) |
| `target folder` | Output directory (cleared on each run) |
| `password file` | Path to auth credentials file (empty = no auth) |
| `timeout` | Seconds to wait per connection attempt |
| `thread` | Number of concurrent openvpn processes |
| `data-ciphers` | Passed as `--data-ciphers` to openvpn (empty = omit flag) |
| `valid string` | Substring to match in openvpn stdout to count as success. Defaults to `"Cannot allocate TUN/TAP dev dynamically"` |

## Key Conventions

**`.ovpn` filename format** — files in the source folder must follow this naming scheme for metadata parsing to work:
```
{country}_{part1}_{part2}_{part3}_{score}.ovpn
```
`getExtrainfo()` splits on `_` and expects exactly 5 segments (indices 0–4). Files not matching this format will panic/misbehave.

**Deduplication key** — `connectInfo` is `nl[1] + "_" + nl[2] + "_" + nl[3]` (the middle three segments), so files with the same connection target but different countries/scores are treated as duplicates.

**Validation heuristic** — the default `validstr` (`"Cannot allocate TUN/TAP dev dynamically"`) is a *successful partial connection* error from OpenVPN, meaning the VPN server is reachable even though the client can't create a TUN device (common in unprivileged environments). This intentionally treats "almost connected" as valid.

**Global state** — all configuration values are package-level globals set by `parseJSON()`. There is no config struct passed around.
