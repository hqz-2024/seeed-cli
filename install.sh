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

# latest 必须用 releases/latest/download/，不能用 releases/download/latest/（后者要求 tag 名就叫 latest）
if [ "$VERSION" = "latest" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${FILE_NAME}"
else
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE_NAME}"
fi

echo "Downloading ${FILE_NAME}..."

# 下载到临时文件：当前目录下若已有同名目录 ~/seeed-cli（配置目录）会与 -o seeed-cli 冲突
TMP_FILE="$(mktemp "${TMPDIR:-/tmp}/seeed-cli-install.XXXXXX")"
trap 'rm -f "$TMP_FILE"' EXIT

# ===== 下载（-f：HTTP 错误时失败，避免把 404 正文当二进制安装）=====
curl -fL "$DOWNLOAD_URL" -o "$TMP_FILE"

# ===== 加权限 =====
chmod +x "$TMP_FILE"

# ===== 安装路径 =====
INSTALL_DIR="/usr/local/bin"

echo "Installing to ${INSTALL_DIR}..."

# ===== 移动 =====
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
  sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi
trap - EXIT

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