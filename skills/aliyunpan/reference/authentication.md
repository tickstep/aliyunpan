# 认证指南（aliyunpan 个人版）

`aliyunpan` 没有独立"登录脚本"概念，登录由 CLI 的 `login` 命令完成。本 skill 提供
`scripts/login.sh` 作为薄封装，用于引导交互式登录或透传 refresh token。

---

## 登录方式

### 方式一：交互式网页登录（默认，需人工）

```bash
aliyunpan login        # 或 bash ${CLAUDE_SKILL_DIR}/scripts/login.sh
```

流程（**两次登录**，因为融合了官方 OpenAPI + Web 端接口）：

1. 终端打印授权链接（**5 分钟内有效**）。
2. 复制到浏览器打开 → 点"允许"（第一次，阿里官方 OpenAPI 授权）。
3. 页面自动跳转到 Web 接口登录页 → 用**阿里云盘 APP 扫码**（第二次）。
4. 切回终端，按 Enter 完成。

Agent 无法替用户扫码，只能把链接给用户并等待。

### 方式二：Refresh Token 登录（无头 / 自动化）

若已持有 refresh token（可从已登录环境的配置文件中获取，注意属于敏感凭据）：

```bash
aliyunpan login -RefreshToken=<refresh_token>
# 或
bash ${CLAUDE_SKILL_DIR}/scripts/login.sh -RefreshToken <refresh_token>
```

无需扫码，适合服务器/CI。上游示例见官方 `sync.sh` / `webdav.sh`。

---

## 常用账号命令

| 命令 | 作用 |
|------|------|
| `aliyunpan who` | 显示当前账号与当前网盘 |
| `aliyunpan loglist` | 列出所有已登录账号 |
| `aliyunpan su <uid>` | 切换到指定账号（`uid` 来自 `loglist`） |
| `aliyunpan logout` | 退出当前账号（会二次确认） |
| `aliyunpan quota` | 查看空间配额（总量/已用） |
| `aliyunpan drive <driveId>` | 切换网盘（备份盘 / 资源库） |

多账号：`aliyunpan` 支持登录多个账号，单个阿里账号最多允许 **10 个客户端**同时在线
（客户端 ID 即配置项 `device_id`，修改后需重启生效）。

---

## 配置文件与凭据保护

- 配置目录优先级：
  1. 环境变量 `ALIYUNPAN_CONFIG_DIR`（必须是**已存在的绝对路径**）
  2. XDG 规范 `$XDG_CONFIG_HOME/aliyunpan`
  3. 程序所在目录（默认回退）
- 本 skill 的脚本统一使用 `ALIYUNPAN_CONFIG_DIR="$HOME/.config/aliyunpan"` 以保持一致。
- 配置内含 **refresh_token** 等敏感凭据：**禁止读取、打印、提交**到仓库或对话。

```bash
# 推荐：显式指定配置目录（脚本已内置）
export ALIYUNPAN_CONFIG_DIR="$HOME/.config/aliyunpan"
mkdir -p "$ALIYUNPAN_CONFIG_DIR"
```

---

## 登录状态校验

```bash
aliyunpan who
```

- 已登录：显示当前账号名与网盘。
- 未登录 / token 失效：提示需要重新 `login`。

token 失效时：`aliyunpan logout` 后重新登录，或用新的 refresh token 登录。
