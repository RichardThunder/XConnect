# XConnect (Go)

Cross-device clipboard, file, and message sync over [Tailscale](https://tailscale.com). Works on macOS, Linux, and Windows.

## Prerequisites

- **Mode 1 (recommended):** Install [Tailscale](https://tailscale.com/download) on each device and log in. Enable MagicDNS in the admin console.
- **Mode 2:** No system Tailscale required; the server can embed Tailscale via tsnet (use `-tsnet` and an auth key).

## Build

```bash
# Server (run on each device)
go build -o xconnect .

# CLI client (push/pull clipboard, send messages, list devices)
go build -o xconnect-cli ./cmd/cli
```

## Run the server

**Using system Tailscale (default):**

```bash
./xconnect
# Listens on :8315. Other devices reach it at http://<hostname>:8315 or http://100.x.x.x:8315
```

**Using embedded Tailscale (tsnet):**

```bash
./xconnect -tsnet -hostname my-device
# First run: open the printed auth URL in a browser to join the tailnet.
# Or: TS_AUTHKEY=tskey-auth-xxx ./xconnect -tsnet -hostname my-device
```

## CLI usage

```bash
# List devices (uses `tailscale status --json` or TAILSCALE_API_TOKEN)
./xconnect-cli list

# Push local clipboard to a peer
./xconnect-cli push <hostname-or-100.x.x.x>

# Pull peer's clipboard to local
./xconnect-cli pull <hostname-or-100.x.x.x>

# Send a short message (writes to peer's clipboard)
./xconnect-cli message <peer> "hello"

# Upload a file to a peer
./xconnect-cli file <peer> /path/to/file
```

## API (HTTP)

| Method | Path | Description |
|--------|------|-------------|
| GET | /clipboard | Get remote clipboard (text) |
| POST | /clipboard | Set remote clipboard (body = text) |
| POST | /files | Upload file (multipart), returns `file_id` |
| GET | /files/:id | Download file |
| POST | /message | JSON `{"text":"..."}` — sets peer clipboard |
| GET | /ws | WebSocket (placeholder) |

Port default: **8315**.

## Testing

From the repo root (with server and CLI built):

```bash
./scripts/test-xconnect.sh
```

The script starts the server on port 18315, then runs:

- **HTTP:** GET/POST `/clipboard`, POST `/message`, POST/GET `/files` (upload + download)
- **CLI:** `push`, `pull`, `message`, `file`, `list`

In headless/CI there is no clipboard; clipboard-related steps are skipped. File upload/download and CLI `message` / `file` are asserted. On a machine with Tailscale and clipboard (e.g. desktop), clipboard and `list` will work as well.
