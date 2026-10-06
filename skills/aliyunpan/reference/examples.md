# 使用示例（自然语言 → 命令）

> 约定：脚本/Agent 中优先使用**绝对路径**（网盘根为 `/`）。下载用 `--np` 避免进度条刷屏。
> 所有命令都可用 `bash ${CLAUDE_SKILL_DIR}/scripts/...` 之外的裸 `aliyunpan` 直接执行。

---

## 1. 查看网盘里有什么

用户："看看阿里云盘根目录/我的文档里有什么"

```bash
aliyunpan ls /                 # 简要
aliyunpan ll /我的文档          # 详细（含大小/时间）
aliyunpan tree /我的文档        # 树形
```

## 2. 上传文件到网盘

用户："把本地的 ./report.pdf 上传到阿里云盘 /文档 目录"

```bash
aliyunpan upload /abs/path/report.pdf /文档
```

- 整个目录：`aliyunpan upload /abs/path/photos /相册`
- 排除文件：`aliyunpan upload -exn '\.jpg$' /abs/path/dir /视频`
- 大文件：`aliyunpan upload -bs 30720 /abs/path/big.mp4 /视频`
- 覆盖同名（**高风险，需确认**）：`aliyunpan upload -ow /abs/path/f.txt /文档`

## 3. 从网盘下载

用户："把阿里云盘 /文档/report.pdf 下载到本地"

```bash
aliyunpan config set -savedir /abs/path/downloads   # 可选：设置保存目录
aliyunpan download --np /文档/report.pdf
```

- 整目录：`aliyunpan download --np /文档`
- 默认保存到程序所在目录的 `Downloads/`；用 `config set -savedir` 自定义。

## 4. 列出/搜索式查看

```bash
aliyunpan ls /相册/*.jpg       # 支持通配符
aliyunpan ls --time /我的资源   # 按时间排序
```

## 5. 新建 / 移动 / 复制 / 重命名

```bash
aliyunpan mkdir /项目/2026
aliyunpan mv /文档/report.pdf /归档
aliyunpan cp /模板/base.txt /项目/2026
aliyunpan rename /文档/old.pdf new.pdf
```

## 6. 分享文件（生成链接）

```bash
# 普通分享（阿里仅支持部分文件类型）
aliyunpan share set -mode 1 /文档/report.pdf
# 快传链接（支持 zip 等大多数文件）
aliyunpan share set -mode 3 /文档/assets.zip
# 查看 / 取消
aliyunpan share list
aliyunpan share cancel <shareid>
```

## 7. 删除（**必须先向用户确认**）

```bash
aliyunpan ls /文档                  # 1) 先展示待删对象
aliyunpan rm /文档/report.pdf       # 2) 用户明确确认后再执行
```

## 8. 同步备份（Beta）

```bash
# 本地 → 云盘（保持本地有完整备份）
aliyunpan sync start -ldir /abs/local/docs -pdir /备份盘/我的文档 -mode upload
# 云盘 → 本地
aliyunpan sync start -ldir /abs/local/docs -pdir /备份盘/我的文档 -mode download
# 双向
aliyunpan sync start -ldir /abs/local/docs -pdir /备份盘/我的文档 -mode sync
```

> 网盘目录必须与本地目录**独占使用**；启动前用 `-mode` 明确方向并经用户确认。

## 9. 共享相册

```bash
aliyunpan album list
aliyunpan album list-file "我的相簿2025"
aliyunpan album download-file 我的相簿2025
```

## 10. WebDAV 挂载（把云盘映射为本地）

```bash
aliyunpan webdav start -ip "0.0.0.0" -port 23077 \
  -webdav_user "admin" -webdav_password "admin" -pan_dir_path "/" -bs 1024
```

## 11. 账号与配额

```bash
aliyunpan who        # 当前账号 + 网盘
aliyunpan loglist    # 所有账号
aliyunpan su <uid>   # 切换账号
aliyunpan quota      # 空间配额
aliyunpan logout     # 退出（需确认）
```

## 12. 无头后台任务（上传/下载/同步）

参考官方 `sync.sh` / `webdav.sh` 模式：用 `-RefreshToken` 登录 + `nohup` 或 systemd 常驻。
详见 [commands.md](./commands.md) 的"Linux 后台下载/上传"小节。
