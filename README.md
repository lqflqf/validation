# validation

A concurrent CLI tool that batch-validates OpenVPN `.ovpn` configuration files by attempting real connections, then ranks and copies the working ones to an output folder.

## Requirements

- Go 1.24+
- `openvpn` binary installed on the system

## Installation

```bash
go build -o validation .
```

## Usage

```bash
./validation              # uses config.json in the current directory
./validation myconfig.json
```

## Configuration

Edit `config.json` before running:

```json
{
    "openvpn": "/usr/local/bin/openvpn",
    "source folder": "/path/to/ovpn/files",
    "target folder": "/path/to/output",
    "password file": "",
    "timeout": 15,
    "thread": 10,
    "data-ciphers": "AES-128-CBC",
    "valid string": ""
}
```

| Key | Description |
|-----|-------------|
| `openvpn` | Absolute path to the `openvpn` binary |
| `source folder` | Directory to scan recursively for `.ovpn` files |
| `target folder` | Output directory — **cleared on each run** |
| `password file` | Path to an auth credentials file (leave empty if not needed) |
| `timeout` | Seconds to wait per connection attempt |
| `thread` | Number of concurrent `openvpn` processes |
| `data-ciphers` | Passed as `--data-ciphers` to openvpn (leave empty to omit) |
| `valid string` | Substring to look for in openvpn output to mark a file as valid. Defaults to `"Cannot allocate TUN/TAP dev dynamically"` if omitted |

## How it works

1. All `.ovpn` files in `source folder` are collected (recursive).
2. Duplicates are removed by connection target, keeping the most recently modified file.
3. Each file is tested concurrently (`thread` workers) by running `openvpn` with a timeout.
4. A file is considered **valid** when the expected string appears in openvpn's output — by default this is a "TUN/TAP" allocation error, which indicates the server was reachable even if a full tunnel couldn't be established (common without root).
5. Valid files are sorted by score (descending) and copied to `target folder` as `{rank}_{country}.ovpn`.

## Input filename format

Source `.ovpn` files must follow this naming scheme:

```
{country}_{part1}_{part2}_{part3}_{score}.ovpn
```

Example: `JP_1.2.3.4_1194_udp_100.ovpn`
