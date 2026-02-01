# XConnect 跨平台构建：macOS / Windows / Linux (X11/Wayland)
# 托盘 GUI (cmd/tray) 使用 Fyne，需开启 CGO；服务与 CLI 可关闭 CGO。

BINARY_server = xconnect
BINARY_cli    = xconnect-cli
BINARY_tray   = xconnect-tray
GO            = go
CGO_ENABLED   = 1

.PHONY: all server cli tray clean build-darwin build-windows build-linux

all: server cli tray

server:
	$(GO) build -o $(BINARY_server) .

cli:
	$(GO) build -o $(BINARY_cli) ./cmd/cli

# 托盘 GUI：需 CGO，且目标平台需有对应图形环境（macOS/Windows/Linux 桌面）
tray:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINARY_tray) ./cmd/tray

# ---------- 多平台交叉编译 ----------
# 服务与 CLI：CGO=0 可交叉编译。托盘 (Fyne) 需 CGO，请在目标系统上执行 make tray 或使用 fyne-cross。

OUT_DIR ?= dist

build-darwin: build-darwin-amd64 build-darwin-arm64
build-darwin-amd64:
	@mkdir -p $(OUT_DIR)/darwin-amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/darwin-amd64/$(BINARY_server) .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/darwin-amd64/$(BINARY_cli) ./cmd/cli

build-darwin-arm64:
	@mkdir -p $(OUT_DIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/darwin-arm64/$(BINARY_server) .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/darwin-arm64/$(BINARY_cli) ./cmd/cli

build-windows:
	@mkdir -p $(OUT_DIR)/windows-amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/windows-amd64/$(BINARY_server).exe .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/windows-amd64/$(BINARY_cli).exe ./cmd/cli

build-linux:
	@mkdir -p $(OUT_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/linux-amd64/$(BINARY_server) .
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/linux-amd64/$(BINARY_cli) ./cmd/cli

# 仅构建服务与 CLI（无需 CGO，适合 CI/无头环境）
build-nocgo:
	@mkdir -p $(OUT_DIR)
	CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/$(BINARY_server) .
	CGO_ENABLED=0 $(GO) build -o $(OUT_DIR)/$(BINARY_cli) ./cmd/cli

clean:
	rm -f $(BINARY_server) $(BINARY_cli) $(BINARY_tray)
	rm -rf $(OUT_DIR)
