#!/usr/bin/env bash

set -e

# ===== 基本信息 =====
REPO="wangzongming/seeed-cli"
VERSION="latest"
BINARY_NAME="seeed-cli"

# ===== 检测系统 =====
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64) ARCH="arm64" ;;
  *) echo "Unsupported ARCH: $ARCH"; exit 1 ;;
esac

FILE_NAME="${BINARY_NAME}-${OS}-${ARCH}"

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE_NAME}"

echo "Downloading ${FILE_NAME}..."

# ===== 下载 =====
curl -L "$DOWNLOAD_URL" -o "$BINARY_NAME"

# ===== 加权限 =====
chmod +x "$BINARY_NAME"

# ===== 安装路径 =====
INSTALL_DIR="/usr/local/bin"

echo "Installing to ${INSTALL_DIR}..."

# ===== 移动 =====
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
  sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

echo ""
echo "  ███████╗███████╗███████╗███████╗██████╗       ██████╗██╗     ██╗"
echo "  ██╔════╝██╔════╝██╔════╝██╔════╝██╔══██╗     ██╔════╝██║     ██║"
echo "  ███████╗█████╗  █████╗  █████╗  ██║  ██║     ██║     ██║     ██║"
echo "  ╚════██║██╔══╝  ██╔══╝  ██╔══╝  ██║  ██║     ██║     ██║     ██║"
echo "  ███████║███████╗███████╗███████╗██████╔╝     ╚██████╗███████╗██║"
echo "  ╚══════╝╚══════╝╚══════╝╚══════╝╚═════╝       ╚═════╝╚══════╝╚═╝"
echo ""
echo "🚀 SEEED-CLI installed successfully!"
echo ""
echo "Please execute seeed-cli --h to view the help"
echo ""
echo "Run: ${BINARY_NAME}"