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

# 托盘 GUI（系统托盘 + 剪贴板历史窗口，需 CGO）
go build -o xconnect-tray ./cmd/tray
# 或：make server cli tray
```

**跨平台构建（服务与 CLI）：**

```bash
make build-darwin   # darwin/amd64 + darwin/arm64 → dist/darwin-*/
make build-windows  # windows/amd64 → dist/windows-amd64/
make build-linux    # linux/amd64 → dist/linux-amd64/
```

托盘应用 (Fyne) 需 CGO，建议在目标系统上直接 `make tray` 或 `go build -o xconnect-tray ./cmd/tray`。Linux 下同一二进制在 X11 与 Wayland 下均可运行（由 Fyne/GLFW 根据环境选择）。

## Releases & 安装包

推送 **tag**（如 `v1.0.0`）到 GitHub 会触发 [Release 工作流](.github/workflows/release.yml)，自动构建并发布到 [Releases](https://github.com/xconnect/xconnect-go/releases)：

| 平台 | 产物 | 安装方式 |
|------|------|----------|
| **Linux** | `.deb`、`.rpm`、`*_linux_amd64.zip` | `sudo dpkg -i xconnect_*_amd64.deb`（Debian/Ubuntu）或 `sudo rpm -i xconnect_*_amd64.rpm`（Fedora/RHEL）；或解压 zip 到 PATH |
| **Windows** | **`.msi`**（官方安装器）、`*_windows_amd64.zip` | 双击 `xconnect_*_windows_amd64.msi` 安装到「Program Files\XConnect」；或解压 zip 使用 |
| **macOS** | **`.pkg`**（官方安装器）、`*_darwin_amd64.zip`、`*_darwin_arm64.zip` | 双击 `xconnect_*_darwin_*.pkg` 安装到 `/usr/local/bin`；或解压 zip 使用 |

**安装后自动行为（deb/rpm、msi、pkg）：**

- **Linux**：安装 `/etc/xdg/autostart/xconnect.desktop`，用户登录图形会话后自动运行「xconnect -daemon -sync」并启动托盘。
- **Windows**：在「所有用户」启动文件夹创建快捷方式，用户登录后自动运行同步服务（-daemon -sync）并启动托盘。
- **macOS**：安装 launchd 用户代理到 `/Library/LaunchAgents/com.xconnect.bin.plist`，用户登录图形会话（Aqua）后自动运行同步与托盘。

触发发布（需有仓库写权限）：

```bash
git tag v1.0.0
git push origin v1.0.0
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

## 托盘 GUI (xconnect-tray)

在后台运行 xconnect 后，可启动托盘应用，从系统托盘打开主窗口查看剪贴板历史（内容 + 来源机器）。

```bash
# 先启动服务（本机或远程，默认连 http://127.0.0.1:8315）
./xconnect -sync

# 再启动托盘（需图形环境）
./xconnect-tray
```

- **托盘：** 点击托盘图标打开菜单，「显示主窗口」打开/显示窗口，「退出」退出应用。
- **主窗口：** 显示从本地 xconnect 服务拉取的剪贴板历史；每条显示内容预览与来源主机。可通过「刷新」按钮重新拉取。
- **环境变量：** `XCONNECT_API=http://host:8315` 可指定 xconnect API 地址（默认 `http://127.0.0.1:8315`）。

支持 macOS、Windows、Linux（X11 / Wayland）。

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
| POST | /clipboard | Set remote clipboard (body = text); optional header `X-From-Host` for history |
| GET | /clipboard/history | JSON array of recent clipboard entries (content, from_host, at) |
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
