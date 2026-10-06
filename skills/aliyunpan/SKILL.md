---
name: aliyunpan
description: >-
  阿里云盘（个人版 / alipan）文件管理 — 通过开源 CLI `aliyunpan`（tickstep/aliyunpan）
  列出、上传、下载、移动、复制、重命名、删除、分享、同步备份、WebDAV 挂载、共享相册。
  TRIGGER: 用户提及"阿里云盘 / alipan / aliyunpan / 阿里云盘个人版"并涉及文件操作
           （上传、下载、列目录、分享、删除、同步/备份等），或要求把文件存到/取自阿里云盘。
  DO NOT TRIGGER: 百度网盘、夸克/115/OneDrive/Google Drive 等其他网盘；阿里云盘**企业版 PDS /
           OpenClaw Skill**（那是付费企业产品，本 skill 不适用）；纯本地文件操作。
allowed-tools: Bash, Read, Glob, Grep, AskUserQuestion
---

# 阿里云盘 Skill（基于 tickstep/aliyunpan）

用开源 CLI [`aliyunpan`](https://github.com/tickstep/aliyunpan) 操作**阿里云盘个人版**。
所有命令的二进制为 `aliyunpan`（默认安装于 `~/.local/bin/aliyunpan`，需在 PATH 中）。

> 本 Skill 收录于本仓库 `skills/aliyunpan/`，是 `aliyunpan` CLI 的 Agent 使用封装。
> 完整命令手册见 [reference/commands.md](./reference/commands.md)（上游官方 manual）。
> JS 插件手册见 [reference/plugin.md](./reference/plugin.md)。

---

## 关键区分（务必先确认）

- 本 skill 面向 **阿里云盘个人版（alipan / openapi.alipan.com）**。
- 用户若提到"阿里云盘**企业版** / PDS / OpenClaw 网盘 Skill / domain_id / 200GB 6.6 元"，
  那属于阿里云**付费企业产品**，**不是本 skill**，应说明不适用并停止执行本 skill 的命令。
- 本 skill 与"百度网盘 skill（bdpan）"相互独立，勿混用命令。

---

## 前置检查（每次触发时按顺序执行）

1. **安装检查**：`command -v aliyunpan`
   - 未安装 → 告知用户并确认后执行 `bash ${CLAUDE_SKILL_DIR}/scripts/install.sh`
   - 默认安装于 `~/.local/bin/aliyunpan`；若命令不在 PATH，先 `export PATH="$HOME/.local/bin:$PATH"`
2. **登录检查**：`aliyunpan who`
   - 输出包含"未登录 / 请登录 / 失败"等 → 需要登录，执行 `bash ${CLAUDE_SKILL_DIR}/scripts/login.sh`
   - `aliyunpan loglist` 可列出所有已登录账号；`aliyunpan su <uid>` 切换账号
3. **网盘检查（可选）**：`aliyunpan who` 会显示当前网盘（默认工作在**备份盘**，`aliyunpan drive <driveId>` 可切到**资源库**）

> 登录是**强交互**流程（浏览器授权 + 阿里 APP 扫码），Agent 无法独立完成，必须引导用户操作，见下节。

---

## 登录（需要用户配合）

`aliyunpan` 融合了阿里官方 OpenAPI + Web 端接口，**需要两次登录**：

1. 运行 `bash ${CLAUDE_SKILL_DIR}/scripts/login.sh`（等价于 `aliyunpan login`）。
2. 终端会打印一条**有效 5 分钟**的授权链接 → 用户复制到浏览器打开 → 点"允许"（第一次登录）。
3. 页面自动跳转到网页接口登录页 → 用户用**阿里 APP 扫码**（第二次登录）。
4. 切回终端，按 Enter 完成登录。

**展示链接要求（给用户）**：把链接放到代码块**外**，用 Markdown 形式 `[点击此处完成阿里云盘授权](URL)`，便于手机端点击。

**非交互 / 无头服务器登录（推荐用于自动化）**：若用户能提供 refresh token，可用
`aliyunpan login -RefreshToken=<token>` 直接登录，无需扫码。
`bash ${CLAUDE_SKILL_DIR}/scripts/login.sh -RefreshToken <token>` 也支持透传。

**登出**：`aliyunpan logout`（会二次确认）。**查看状态**：`aliyunpan who` / `aliyunpan quota`。

---

## 核心概念

- **工作目录（cwd）**：`aliyunpan` 有远端当前目录，用 `cd`/`pwd` 操作；相对路径基于它，绝对路径以 `/` 开头（根为 `/`）。
  - 每次调用 CLI 是**独立进程**，cwd 不跨命令保持；脚本里请用**绝对路径**，或先用 `cd`（CLI 会持久化最后目录到配置）。
- **网盘（drive）**：默认"备份盘"；资源库需 `aliyunpan drive <driveId>` 切换。可用 `aliyunpan drive` 无参交互选择。
- **本地目录**：`lcd`/`lpwd`/`lls` 用于查看/切换本地工作目录。
- **下载保存目录**：默认程序所在目录的 `Downloads/`；用 `aliyunpan config set -savedir <dir>` 自定义。

---

## 常用命令速查（自然语言 → 命令）

| 用户意图 | 命令 |
|---------|------|
| 列出网盘目录 | `aliyunpan ls <目录>` / 详细 `aliyunpan ll <目录>` / 树形 `aliyunpan tree <目录>` |
| 上传文件/目录 | `aliyunpan upload <本地路径...> <目标网盘目录>` |
| 下载文件/目录 | `aliyunpan download <网盘路径...>`（保存到 savedir） |
| 新建目录 | `aliyunpan mkdir <网盘目录>` |
| 移动 | `aliyunpan mv <源...> <目标目录>` |
| 复制 | `aliyunpan cp <源...> <目标目录>` |
| 重命名 | `aliyunpan rename <路径> <新名字>` |
| 删除 | `aliyunpan rm <路径...>`（**需用户明确确认**） |
| 分享 | `aliyunpan share set -mode 1 <路径...>`（普通分享）/ `-mode 3`（快传链接） |
| 列出/取消分享 | `aliyunpan share list` / `aliyunpan share cancel <shareid...>` |
| 空间配额 | `aliyunpan quota` |
| 同步备份 | `aliyunpan sync start -ldir <本地> -pdir <网盘> -mode upload\|download\|sync` |
| 共享相册 | `aliyunpan album list` / `album list-file <相簿>` / `album download-file <相簿>` |
| WebDAV 挂载 | `aliyunpan webdav start -ip 0.0.0.0 -port 23077 ...` |

常用参数：
- 上传：`-ow`（覆盖，同名文件移入回收站）、`-skip`（跳过同名）、`-exn '<正则>'`（排除）、`-bs <KB>`（分片大小，大文件调大）、`-p <并发>`、`--np`（不显示进度）
- 下载：`-p <并发>`、`--np`；多用户联合下载见手册
- 全局：`--verbose`（调试日志）；`aliyunpan help <命令>` 查看用法

**下载/上传进度条**在非交互/被 Agent 捕获的终端里可能刷屏，建议加 `--np`。

---

## 安全与确认规则（重要）

| 风险 | 操作 | 策略 |
|------|------|------|
| **高** | `rm` 删除、`-ow` 覆盖上传、`logout`、`sync`（会改文件） | 仅在用户**明确要求**时执行；执行前**列出目标路径**并等待用户**明确确认**；`sync` 前确认方向（upload/download/sync）与目录对 |
| **中** | `upload`、`download`、`mv`、`cp`、`rename` | 路径明确直接执行；路径/目标不明确（序数、代词、"那个"）先确认 |
| **低** | `ls`、`ll`、`tree`、`pwd`、`who`、`quota`、`mkdir`、`share list`、`config`（只读） | 直接执行 |

补充规则：
- 用户取消意图（"算了""不要了""取消"）→ 立即中止，不执行任何命令。
- `sync` 会持续改变文件状态，网盘目录必须与本地目录**独占使用**，禁止指向用户其他在用目录；启动前必须确认。
- 禁止路径穿越与越权：只在用户指定范围内操作；不要臆造路径。

---

## 配置与凭据保护

- 配置文件：默认在程序目录或 `$ALIYUNPAN_CONFIG_DIR`（本 skill 的脚本统一设为 `~/.config/aliyunpan`）。
  - 含 `refresh_token` 等敏感凭据，**禁止读取或输出其内容**，禁止写入公开仓库或对话。
- 下载目录等：`aliyunpan config`（只读查看）、`aliyunpan config set -savedir <dir>` 等。
- Agent 禁止主动导出/改写 token 环境变量或伪造 refresh token。

---

## 参考文档（按需查阅，无需预加载）

| 文档 | 何时查阅 |
|------|---------|
| [reference/commands.md](./reference/commands.md) | 完整命令参数、示例（上游官方手册） |
| [reference/plugin.md](./reference/plugin.md) | JavaScript 插件（上传/下载/删除/同步/令牌钩子） |
| [reference/authentication.md](./reference/authentication.md) | 登录/登出细节、refresh token、配置位置 |
| [reference/examples.md](./reference/examples.md) | 常见自然语言任务的完整命令组合 |
| [reference/troubleshooting.md](./reference/troubleshooting.md) | 报错排查（登录失效、下载慢、路径问题等） |
