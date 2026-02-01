#!/usr/bin/env bash
# Install clipboard utilities for XConnect on Linux and optionally detect Windows.
# Run from repo root or with no args. On Linux, detects distro and installs xclip or wl-clipboard.

set -e

case "$(uname -s)" in
  Linux)
    if command -v xclip &>/dev/null || command -v xsel &>/dev/null || command -v wl-copy &>/dev/null; then
      echo "Clipboard utility already installed (xclip, xsel, or wl-clipboard)."
      exit 0
    fi
    if [ -f /etc/os-release ]; then
      . /etc/os-release
      case "$ID" in
        fedora|rhel|centos|ol)
          echo "Installing xclip (Fedora/RHEL)..."
          sudo dnf install -y xclip
          ;;
        debian|ubuntu|pop)
          echo "Installing xclip (Debian/Ubuntu)..."
          sudo apt-get update -qq && sudo apt-get install -y xclip
          ;;
        arch|manjaro)
          echo "Installing xclip (Arch)..."
          sudo pacman -Sy --noconfirm xclip
          ;;
        opensuse*)
          echo "Installing xclip (openSUSE)..."
          sudo zypper install -y xclip
          ;;
        *)
          echo "Unknown distro: $ID. Install one of: xclip, xsel, wl-clipboard"
          echo "  xclip:   https://github.com/astrand/xclip"
          echo "  xsel:    https://github.com/kfish/xsel"
          echo "  Wayland: wl-clipboard (e.g. dnf install wl-clipboard)"
          exit 1
          ;;
      esac
    else
      echo "Cannot detect Linux distro. Install xclip, xsel, or wl-clipboard manually."
      exit 1
    fi
    echo "Done. Run xconnect or xconnect-cli again."
    ;;
  Darwin)
    echo "macOS: clipboard uses system APIs; no extra install needed."
    ;;
  MINGW*|MSYS*|CYGWIN*)
    echo "Windows: clipboard uses Win32 API; no extra install needed."
    ;;
  *)
    echo "Unknown OS: $(uname -s). Install a clipboard utility for your platform."
    exit 1
    ;;
esac
