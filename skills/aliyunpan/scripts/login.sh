#!/bin/bash
# aliyunpan 登录辅助脚本
#
# 用法:
#   bash login.sh                          # 交互式网页登录（浏览器授权 + 阿里APP扫码）
#   bash login.sh -RefreshToken <token>    # 用 refresh token 非交互登录（无头服务器）
#   bash login.sh --status                 # 仅查看登录状态
#
# 说明: 登录是强交互流程（两次登录），Agent 无法替用户扫码，只能引导用户操作。

set -euo pipefail

export PATH="$HOME/.local/bin:$PATH"
export ALIYUNPAN_CONFIG_DIR="${ALIYUNPAN_CONFIG_DIR:-$HOME/.config/aliyunpan}"
mkdir -p "$ALIYUNPAN_CONFIG_DIR"

if ! command -v aliyunpan >/dev/null 2>&1; then
    echo "[ERROR] 未找到 aliyunpan，请先运行: bash \"$(dirname "$0")/install.sh\"" >&2
    exit 1
fi

print_status() {
    echo "[INFO] 登录状态:"
    aliyunpan who 2>&1 || true
}

case "${1:-}" in
    -RefreshToken=*)
        token="${1#-RefreshToken=}"
        echo "[INFO] 使用 refresh token 登录（非交互）..."
        aliyunpan login "-RefreshToken=${token}"
        print_status
        ;;
    -RefreshToken)
        token="${2:-}"
        if [ -z "$token" ]; then
            echo "[ERROR] -RefreshToken 需要一个 token 值" >&2
            exit 1
        fi
        echo "[INFO] 使用 refresh token 登录（非交互）..."
        aliyunpan login "-RefreshToken=${token}"
        print_status
        ;;
    --status)
        print_status
        ;;
    *)
        echo "[INFO] 检查登录状态..."
        if aliyunpan who 2>/dev/null | grep -qiv '未登录' && aliyunpan who 2>/dev/null | grep -qiE 'uid|账号|昵称|用户'; then
            echo "[INFO] 已登录，无需重复授权。"
            print_status
            exit 0
        fi
        echo ""
        echo "================================================================"
        echo " 阿里云盘登录（两次登录，需人工操作）"
        echo "================================================================"
        echo " 1. 终端会打印一条授权链接（有效 5 分钟）。"
        echo " 2. 复制到浏览器打开 → 点『允许』（第一次登录）。"
        echo " 3. 页面自动跳转到网页登录页 → 用阿里云盘 APP 扫码（第二次登录）。"
        echo " 4. 切回终端，按 Enter 完成。"
        echo "================================================================"
        echo ""
        echo "[INFO] 启动登录流程（需要你按提示在浏览器/APP 完成操作）..."
        echo ""
        aliyunpan login
        print_status
        ;;
esac
