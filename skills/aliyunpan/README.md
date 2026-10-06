# aliyunpan Agent Skill（阿里云盘 Skill）

一个面向 AI Agent（Claude Code / Cursor / Codex / OpenClaw 等）的 **阿里云盘（个人版 / alipan）文件管理 Skill**，
封装本仓库的 [`aliyunpan`](https://github.com/tickstep/aliyunpan) 命令行客户端，让 Agent 用自然语言完成网盘操作。

## 能力

- 列出 / 树形查看网盘目录（`ls` / `ll` / `tree`）
- 上传、下载文件与目录
- 新建目录、移动、复制、重命名
- 分享文件/目录、列出/取消分享
- 同步备份（本地↔云盘）、WebDAV 挂载、共享相册
- 账号与配额查询（`who` / `loglist` / `quota` / `drive`）

## 目录结构

```
skills/aliyunpan/
├── SKILL.md                     # Skill 定义（Agent 行为规范）
├── VERSION                      # 对应 aliyunpan CLI 版本
├── reference/
│   ├── commands.md              # 完整命令手册
│   ├── plugin.md                # JavaScript 插件手册
│   ├── plugin-samples/          # 官方插件样例
│   ├── authentication.md        # 登录/登出、refresh token、配置位置
│   ├── examples.md              # 自然语言任务 → 命令组合
│   └── troubleshooting.md       # 常见问题排查
└── scripts/
    ├── install.sh               # 安装/升级 aliyunpan（官方预编译，或 --from-source 编译）
    ├── login.sh                 # 登录引导（交互扫码 / -RefreshToken 非交互 / --status）
    └── uninstall.sh             # 卸载（--purge 清配置）
```

## 安装

把 `skills/aliyunpan/` 放入你的 Agent 技能目录，例如：

- Claude Code：`~/.claude/skills/aliyunpan/`
- Cursor：`~/.cursor/skills/aliyunpan/`

然后让 Agent 触发该 Skill，或手动安装 CLI 与登录：

```bash
# 安装 aliyunpan CLI（官方预编译）
bash skills/aliyunpan/scripts/install.sh

# 登录（浏览器授权 + 阿里云盘 APP 扫码，共两次）
bash skills/aliyunpan/scripts/login.sh

# 或使用 refresh token 非交互登录（无头服务器）
bash skills/aliyunpan/scripts/login.sh -RefreshToken <token>
```

## 使用示例

安装并登录后，直接对 Agent 说：

```
看看阿里云盘根目录有什么
把本地的 ./report.pdf 上传到阿里云盘 /文档
把网盘里 /depmap 下载到本地
把 /xdftools 生成分享链接
```

## 安全说明

- 所有 `bdpan`/网盘凭据文件（`$ALIYUNPAN_CONFIG_DIR` 或程序目录下的 `aliyunpan_config.json`，含 `refresh_token`）
  均为敏感信息，**禁止读取、输出或提交到公开仓库**。
- 删除、覆盖、退出登录、同步等高风险操作需用户明确确认后才执行。
- 登录授权链接有效期短，请勿转发给他人。

## 许可证

本 Skill 遵循本仓库 [Apache License 2.0](../../LICENSE)。底层 `aliyunpan` CLI 由 [tickstep/aliyunpan](https://github.com/tickstep/aliyunpan) 提供。
