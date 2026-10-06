#!/bin/bash
# aliyunpan 卸载脚本
#
# 用法:
#   bash uninstall.sh            # 交互确认后删除二进制
#   bash uninstall.sh --yes      # 跳过确认
#   bash uninstall.sh --purge    # 同时删除配置目录（含登录凭据）

set -euo pipefail

INSTALL_DIR="${ALIYUNPAN_INSTALL_DIR:-$HOME/.local/bin}"
CONFIG_DIR="${ALIYUNPAN_CONFIG_DIR:-$HOME/.config/aliyunpan}"
ASSUME_YES="no"
PURGE="no"

for arg in "$@"; do
    case "$arg" in
        --yes|-y) ASSUME_YES="yes" ;;
        --purge) PURGE="yes" ;;
        *) echo "未知参数: $arg" >&2; exit 1 ;;
    esac
done

confirm() {
    if [ "$ASSUME_YES" = "yes" ]; then return 0; fi
    local reply
    read -r -p "$1 [y/N] " reply
    [[ "$reply" =~ ^[Yy]$ ]]
}

BIN="$INSTALL_DIR/aliyunpan"
if [ -f "$BIN" ]; then
    if confirm "删除可执行文件 $BIN ?"; then
        rm -f "$BIN"
        echo "[INFO] 已删除: $BIN"
    else
        echo "[INFO] 跳过删除二进制"
    fi
else
    echo "[INFO] 未找到 $BIN"
fi

if [ "$PURGE" = "yes" ] && [ -d "$CONFIG_DIR" ]; then
    if confirm "删除配置目录 $CONFIG_DIR（含登录凭据，不可恢复）?"; then
        rm -rf "$CONFIG_DIR"
        echo "[INFO] 已删除配置目录: $CONFIG_DIR"
    else
        echo "[INFO] 跳过删除配置目录"
    fi
fi

echo "[INFO] 完成。若 .bashrc 中添加过 PATH，可自行清理。"
