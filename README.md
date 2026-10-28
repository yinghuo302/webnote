# Webnote：后端与部署

此目录只包含 Go 1.26 / PostgreSQL API。前端在**同级的 `webnote-web/` 仓库**，Go 不再嵌入或直接提供任何页面；Nginx 托管前端并代理 `/api/` 到 Go。默认 API 只监听 `127.0.0.1:8000`。

维护文档：架构与数据流见 [docs/architecture.md](<docs/architecture.md>)；会话、登录和安全响应见 [docs/security.md](<docs/security.md>)；原生 `/opt/webnote` 部署见 [docs/operations.md](<docs/operations.md>)；旧 SQLite 数据迁移见 [docs/sqlite-migration.md](<docs/sqlite-migration.md>)。其他 AI agents 应先阅读 [AGENTS.md](<AGENTS.md>)。

## Cookie 与登录安全模型

- 登录/注册成功后生成 **32 字节加密安全随机数**，以 64 字符十六进制不透明 Cookie 发送给浏览器；浏览器只持有令牌，**不保存用户 ID、邮箱或密码**。PostgreSQL `sessions` 表只存令牌 SHA-256 散列、用户 ID 与过期时间。每个需授权的请求都重新查询服务端会话。
- 有效期固定 **24 小时**（不是滑动续期）。Cookie 的 `Max-Age=86400` 和 `Expires` 与服务端 `expires_at` 配合。生产模式使用 `__Host-webnote_session`（无 Domain、Path=/、Secure、HttpOnly、SameSite=Strict）。`WEBNOTE_SECURE_COOKIES` 默认启用，仅本机纯 HTTP 开发时才可显式设为 `false`（Cookie 名改为 `webnote_session`）。浏览器关闭后 Cookie 可能仍保留，直至过期或清除。
- 退出登录会删除服务端这一会话；修改密码会撤销该账号全部设备会话。发现 Cookie 泄露时，可调用 `POST /api/private/logout-all` 撤销全部设备会话，或修改密码；后台也可删除该用户的 `sessions` 行立即吊销。**已被窃取且未被撤销的有效令牌仍可被重放**，无法仅靠 HttpOnly 阻止；须配合 HTTPS、前端防 XSS、Nginx 安全响应头、短有效期、日志不记录 Cookie、主机与数据库权限。旧版 `LoginSession` Cookie 在升级后不被识别。
- 密码使用 bcrypt 存储，旧明文密码在成功登录后升级；新密码至少 12 字符。登录 IP 限流与 Nginx 限流降低在线猜测风险。邮件验证码来自系统安全随机数，以 bcrypt 储存，10 分钟内有效，同邮箱发送间隔至少 60 秒，错误尝试最多 5 次，验证成功后删除；注册/登录错误不泄露密码。SMTP 必须支持 STARTTLS。危险操作检查 `Origin` / `Sec-Fetch-Site`，Cookie 使用 Strict；不要在前端插入未经净化的 Markdown HTML。以上措施不保证对所有分布式攻击或已入侵主机的绝对安全。

## 本机原生安装：/opt/webnote

准备本机 PostgreSQL（已安装 14）、独立角色及数据库；旧 SQLite 数据不会自动迁移。后端目录为 `webnote/`、独立前端为 `webnote-web/`。在后端目录构建后端：

```sh
CGO_ENABLED=0 go build -trimpath -o bin/webnote .
cd ../webnote-web && pnpm run build
```

使用与 OS 用户不同的 PostgreSQL 专属角色 `webnote`，仅授予 Webnote 数据库权限；数据库口令放进源 `.env` 的 `WEBNOTE_DATABASE_URL=postgres://webnote:<URL编码后的密码>@127.0.0.1:5432/webnote?sslmode=disable`（本机回环连接），文件权限 0600。安装脚本只将白名单数据库/SMTP配置复制进 root-only 凭据，再通过 systemd `LoadCredential` 在服务私有凭据目录交给程序，不进入环境变量、Nginx 静态根或 Nginx worker 可读的普通文件；手动开发时可将环境变量临时加载进当前 shell。按需增加 `WEBNOTE_SMTP_HOST`、`WEBNOTE_SMTP_PORT=587`、`WEBNOTE_SMTP_USER`、`WEBNOTE_SMTP_PASSWORD`。不要提交密钥；首次注册必须有 SMTP，否则验证码接口返回 503。过去的 `WEBNOTE_SESSION_KEY` 已弃用。

在审阅 [安装脚本](<deploy/install-local.sh>) 后，**由你自己选择**运行 `sudo sh webnote/deploy/install-local.sh`（从父目录运行）；它会创建 `/opt/webnote/bin`、`/opt/webnote/data`、`/opt/webnote/config`、`/opt/webnote/www`，以低权限 `web` 用户拥有数据目录、root 持有二进制和静态前端；SMTP 配置只存 root-only 文件并通过 systemd `LoadCredential` 私密挂载给应用，不进入进程环境、不让 Nginx 直接读取。脚本安装 [systemd 单元](<deploy/webnote.service>)。它**不会启动服务、更改现有 Nginx 站点或导入旧数据**，且不会覆盖已有的 `/opt/webnote/config/webnote.env`。当前主机 Nginx worker 也以 `web` 用户运行；数据库角色使用独立名称 `webnote` 和口令认证，避免 Nginx worker 继承 peer 数据库访问。数据库与 SMTP 密钥通过 systemd 私有 credential 传递，前端静态目录不含这些文件。不要手工改成 `EnvironmentFile` 秘密注入。

申请域名与 TLS 证书后，替换 [Nginx HTTPS 模板](<deploy/nginx-webnote.conf>) 的 `YOUR_DOMAIN` 和证书路径，再自行将配置安装到 `/etc/nginx/conf.d/`，先 `nginx -t` 再重载 Nginx。仅在证书、域名及配置验证后启用：`sudo systemctl enable --now webnote`，`curl http://127.0.0.1:8000/api/ping`，再用 HTTPS 验证前端和登录 Cookie。Nginx 覆盖 `X-Forwarded-For` 为真实客户端 IP；Go 只信任本机反向代理。服务单元默认 `WEBNOTE_SECURE_COOKIES=true`，不应通过公网 HTTP 提供登录入口。TLS 证书未就绪时，不要把此模板直接启用。

## 旧 SQLite 离线迁移

先停止旧服务，备份 SQLite 数据库和 WAL/SHM、旧 `data/`、`img/`。只读预检 `go run ./cmd/import-sqlite --check --sqlite ./webnote.db --data-dir ./data` 不连接 PostgreSQL，只报告数量和冲突。数据所有者确认旧邮箱冲突处理方案后，才可对**空** PostgreSQL 库执行 `go run ./cmd/import-sqlite --sqlite ./webnote.db --data-dir ./data`。导入在事务中执行，重复邮箱或正文丢失会中断/回滚，工具不会自动删除、改名或合并账户。迁移完成后核对正文，旧图片需经授权复制到 `/opt/webnote/data/img`；旧数据绝不可直接删除。日常以 `pg_dump -Fc` 备份 PostgreSQL，同时备份图片、离线验证恢复。旧版仓库曾含 SMTP 明文凭据，须在服务端轮换并检查 Git 历史。

## 可选 Docker 与验证

[Dockerfile](<Dockerfile>) 和 [Compose](<compose.yaml>) 仅维持兼容；**此主机不使用 Docker 部署**。容器仍只运行 Go API，独立前端需 Nginx 另行托管；默认 Secure Cookie 要求 HTTPS。

`GET /api/ping`、`POST /api/public/signin`、`POST /api/public/signup`、`POST /api/public/code`；会话内提供 `/api/private/user`、`/api/private/files`、`/api/private/file`、`/api/private/share`、`/api/private/img`、`/api/private/avatar`，公开文章通过 `/api/public/articles`、`/api/public/article`、`/api/public/img/:name` 获取。笔记正文最大 1 MiB，图片最大 5 MiB。数据库集成测试使用独立临时 schema：`WEBNOTE_TEST_DATABASE_URL='postgres://.../专用测试库?sslmode=disable' go test ./...`；另运行 `go vet ./...`。无测试 URL 时会跳过需要 PostgreSQL 的测试。
