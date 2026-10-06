# 故障排查

## 命令找不到（command not found: aliyunpan）

- 二进制应在 `~/.local/bin/aliyunpan`。先补 PATH：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

- 未安装时执行 `bash ${CLAUDE_SKILL_DIR}/scripts/install.sh`。

## 登录相关

| 现象 | 处理 |
|------|------|
| 提示未登录 / token 失效 | `aliyunpan logout` 后重新 `login`；或用新的 `-RefreshToken` 登录 |
| 授权链接 5 分钟内失效 | 重新运行 `login`，尽快在浏览器完成 |
| 只完成了第一次登录（浏览器允许），没扫码 | 这是**两次登录**流程，必须再用阿里 APP 扫码；切回终端按 Enter |
| 无图形界面服务器 | 用 `-RefreshToken=<token>` 非交互登录，或先在有浏览器的机器登录后复制配置 |
| 多设备冲突（单账号最多 10 客户端） | 减少同时在线设备，或修改 `device_id` 后重启 |

> 登录链接只在**当前终端**输出，且有效期短；不要复制到公开渠道。

## 下载 / 上传

| 现象 | 处理 |
|------|------|
| 下载慢 / 硬盘占用高 | `aliyunpan config set -cache_size 64KB`（1KB~256KB） |
| 想提高并发 | `aliyunpan config set -max_download_parallel 3` / `-max_upload_parallel 10` |
| 限速 | `config set -max_download_rate <KB/s>` / `-max_upload_rate <KB/s>` |
| 大文件上传失败 | 调大分片：`upload -bs 30720 ...`；调大超时 `--timeout 60` |
| 进度条刷屏 / 干扰日志 | 加 `--np`（no progress） |
| 想跳过同名 | `upload -skip ...`；想覆盖（**高风险**）`upload -ow ...` |
| 阿里 ECS 环境下载慢 | `aliyunpan config set -transfer_url_type 2` |

## 路径相关

| 现象 | 处理 |
|------|------|
| Windows 风格路径报错 | 用 `/` 而非 `\`；网盘路径以 `/` 开头 |
| 相对路径找不到文件 | 相对路径基于 CLI 的**工作目录**；脚本中改用绝对路径或先 `cd` |
| 下载保存位置不对 | `aliyunpan config set -savedir <绝对目录>` |

## 权限 / 网盘范围

| 现象 | 处理 |
|------|------|
| 找不到文件 | 确认当前网盘：`aliyunpan who`；资源库文件需 `aliyunpan drive <driveId>` 切换 |
| 分享失败 | 阿里仅支持部分文件类型的普通分享；用 `share set -mode 3` 快传链接 |

## 调试

- 加全局参数 `--verbose` 打印调试日志：`aliyunpan --verbose <命令>`。
- 查看环境：`aliyunpan env`；查看历史：`aliyunpan history`。
- 官方 FAQ / Issue：<https://github.com/tickstep/aliyunpan/issues>

## 配置文件位置异常

若 `aliyunpan config` 显示配置目录在程序目录下，设置环境变量后**重新登录**：

```bash
export ALIYUNPAN_CONFIG_DIR="$HOME/.config/aliyunpan"
mkdir -p "$ALIYUNPAN_CONFIG_DIR"
```

注意：切换配置目录后原登录态不会自动迁移，需要重新登录。
