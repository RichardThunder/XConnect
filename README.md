# XConnect (Go)

Cross-device clipboard, file, and message sync over [Tailscale](https://tailscale.com). Works on macOS, Linux, and Windows.

## Prerequisites

- **Mode 1 (recommended):** Install [Tailscale](https://tailscale.com/download) on each device and log in. Enable MagicDNS in the admin console.
- **Mode 2:** No system Tailscale required; the server can embed Tailscale via tsnet (use `-tsnet` and an auth key).
- **Clipboard (Linux):** On Linux, XConnect needs a clipboard utility. If you see "No clipboard utilities available", run:
  ```bash
  ./scripts/install-clipboard-deps.sh
  ```
  Or install manually: **xclip** (e.g. `sudo dnf install xclip` on Fedora), **xsel**, or **wl-clipboard** (Wayland). Windows and macOS use system APIs and need no extra install.

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

**Clipboard auto-sync (no manual pull):**

When you copy on one device, automatically broadcast to other Tailscale devices:

```bash
./xconnect -sync
# Peers are discovered via `tailscale status --json`. Optionally:
#   -hostname my-mac        use this name as "self" when excluding from peer list
#   -peers "linux,win"     comma-separated peer hostnames (skip discovery)
#   -sync-interval 1s      poll clipboard interval (default 1s)
#   -api-token ...         or TAILSCALE_API_TOKEN for API-based discovery
```

Run `./xconnect -sync` on each device; when you copy on any device, others receive the content and write it to their clipboard.

**Service mode (run in background, with logging):**

Run as a background process; logs are written to a file. Works on Linux, macOS, and Windows.

```bash
./xconnect -daemon
# Starts a detached process. Logs go to a platform-specific path:
#   Windows: %LocalAppData%\XConnect\logs\xconnect.log
#   Linux/macOS: ~/.local/state/xconnect/xconnect.log (or $XDG_STATE_HOME/xconnect/xconnect.log)
```

Optional: specify log file and combine with other flags:

```bash
./xconnect -daemon -sync -log-file /var/log/xconnect.log
# Or on Windows: -log-file "C:\Logs\xconnect.log"
```

- **Linux/macOS:** The process is started in a new session (`setsid`); it does not receive terminal signals from the parent.
- **Windows:** The process is started with `DETACHED_PROCESS | CREATE_NO_WINDOW` (no console window, runs independently).

To run as a system service, use your OS mechanism (e.g. systemd unit on Linux, launchd on macOS, Task Scheduler or NSSM on Windows) and run `xconnect -log-file <path>` (or `-daemon` once, then the service manager can start the binary without `-daemon` and redirect stdout/stderr to a log file).

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

## Clipboard dependencies (Linux / Windows)

| Platform | Notes |
|----------|--------|
| **Linux** | Needs **xclip**, **xsel**, or **wl-clipboard** (Wayland). Run `./scripts/install-clipboard-deps.sh` to auto-install for Fedora, Debian/Ubuntu, Arch, openSUSE. |
| **Windows** | Uses Win32 API; no extra install. Run as the logged-in user (not a headless service) for clipboard access. |
| **macOS** | Uses system APIs; no extra install. |

If clipboard read/write fails, the error message includes install hints for your OS.

## Testing

From the repo root (with server and CLI built):

```bash
./scripts/test-xconnect.sh
```

The script starts the server on port 18315, then runs:

- **HTTP:** GET/POST `/clipboard`, POST `/message`, POST/GET `/files` (upload + download)
- **CLI:** `push`, `pull`, `message`, `file`, `list`

In headless/CI there is no clipboard; clipboard-related steps are skipped. File upload/download and CLI `message` / `file` are asserted. On a machine with Tailscale and clipboard (e.g. desktop), clipboard and `list` will work as well.
