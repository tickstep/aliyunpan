#!/bin/bash
# aliyunpan 安装/升级脚本（官方预编译二进制）
# 默认安装 v0.4.0 的 linux/darwin 预编译包到 ~/.local/bin。
#
# 用法:
#   bash install.sh                 # 安装默认版本
#   ALIYUNPAN_VERSION=0.4.0 bash install.sh
#   bash install.sh --latest        # 查询 GitHub 最新 release
#   bash install.sh --from-source   # 用本机 Go 从源码编译（需源码目录，见下）
#   ALIYUNPAN_INSTALL_DIR=/usr/local/bin bash install.sh

set -euo pipefail

INSTALL_DIR="${ALIYUNPAN_INSTALL_DIR:-$HOME/.local/bin}"
SOURCE_DIR="${ALIYUNPAN_SOURCE_DIR:-$HOME/aliyunpan}"
VERSION="${ALIYUNPAN_VERSION:-0.4.0}"
MODE="prebuilt"

for arg in "$@"; do
    case "$arg" in
        --latest) MODE="latest" ;;
        --from-source) MODE="source" ;;
        --help|-h)
            sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) echo "未知参数: $arg" >&2; exit 1 ;;
    esac
done

detect_os() {
    case "$(uname -s)" in
        Linux) echo "linux" ;;
        Darwin) echo "darwin-macos" ;;
        *) echo "不支持的系统: $(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        armv7l) echo "armv7" ;;
        i386|i686) echo "386" ;;
        *) echo "不支持的架构: $(uname -m)" >&2; exit 1 ;;
    esac
}

build_from_source() {
    local go_bin
    go_bin="$(command -v go || true)"
    if [ -z "$go_bin" ]; then
        echo "未找到 go，无法从源码编译；请安装 Go 或使用预编译安装。" >&2
        exit 1
    fi
    if [ ! -d "$SOURCE_DIR" ]; then
        echo "源码目录不存在: $SOURCE_DIR（可先 git clone https://github.com/tickstep/aliyunpan.git $SOURCE_DIR）" >&2
        exit 1
    fi
    echo "[INFO] 从源码编译: $SOURCE_DIR"
    mkdir -p "$INSTALL_DIR"
    ( cd "$SOURCE_DIR" && go build -o "$INSTALL_DIR/aliyunpan" . )
    echo "[INFO] 已安装: $INSTALL_DIR/aliyunpan"
}

install_prebuilt() {
    local os arch zip url tmp
    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ "$MODE" = "latest" ]; then
        local tag
        tag="$(curl -fsSL --max-time 30 https://api.github.com/repos/tickstep/aliyunpan/releases/latest \
               | grep -oE '"tag_name"\s*:\s*"[^"]+"' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/')"
        if [ -n "$tag" ]; then VERSION="$tag"; fi
    fi

    zip="aliyunpan-v${VERSION}-${os}-${arch}.zip"
    url="https://github.com/tickstep/aliyunpan/releases/download/v${VERSION}/${zip}"
    tmp="$(mktemp -d)"

    echo "[INFO] 下载: $url"
    if ! curl -fSL --max-time 180 -o "$tmp/$zip" "$url"; then
        echo "[ERROR] 下载失败，请检查版本号/网络: $url" >&2
        rm -rf "$tmp"; exit 1
    fi

    echo "[INFO] 解压并安装到: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    unzip -oq "$tmp/$zip" -d "$tmp/extracted"
    local bin
    bin="$(find "$tmp/extracted" -type f -name aliyunpan | head -1)"
    if [ -z "$bin" ]; then
        echo "[ERROR] 压缩包内未找到 aliyunpan 可执行文件" >&2
        rm -rf "$tmp"; exit 1
    fi
    install -m 0755 "$bin" "$INSTALL_DIR/aliyunpan"
    rm -rf "$tmp"

    echo "[INFO] ✓ 安装完成"
    "$INSTALL_DIR/aliyunpan" --help >/dev/null 2>&1 && echo "[INFO] 版本信息:" && "$INSTALL_DIR/aliyunpan" --help 2>&1 | grep -A2 '^VERSION' || true
}

case "$MODE" in
    source) build_from_source ;;
    *) install_prebuilt ;;
esac

if ! command -v aliyunpan >/dev/null 2>&1; then
    echo ""
    echo "[WARN] aliyunpan 不在 PATH 中。请将以下内容加入 shell 配置："
    echo "       export PATH=\"$INSTALL_DIR:\$PATH\""
fi

echo ""
echo "下一步：完成阿里云盘登录"
echo "  bash \"$(cd "$(dirname "$0")" && pwd)/login.sh\""
