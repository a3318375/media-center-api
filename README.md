# Media Center API

给 `media-center-web` 提供 API 的 Gin 单体后端，使用 SQLite 保存单用户、登录会话和分组配置。

## 首次启动

安装 Go 1.25 或更新版本后，在此目录运行：

```powershell
go mod tidy
go run ./cmd/server
```

服务默认监听 `http://localhost:8080`。首次打开前端时会显示创建管理员账号页面；账号 3–32 位，密码不能为空且最多 72 字节（bcrypt 限制）。初始化后可用新账号登录。健康检查地址是 `GET /api/health`。

默认数据保存在项目目录的 `data/`：`media-center.db` 是 SQLite 数据库，`master.key` 用于加密 Emby API Key、115 Cookie 和 Telegram Bot Token。备份和迁移时需要一起保存这两个文件；丢失密钥后无法解密已存的服务凭证。可用 `MEDIA_CENTER_DATA_DIR` 指定其他持久化目录，用 `HTTP_ADDR` 指定监听地址。

## 前端连接

在 `media-center-web` 目录启动开发服务器时，开启真实登录和配置 API、保留其他媒体页面的 mock：

```powershell
$env:VITE_BACKEND_AUTH = 'true'
pnpm dev
```

Vite 会把 `/api` 代理到 `127.0.0.1:8080`。如需让所有 API 请求都直连后端，可设 `VITE_ENABLE_MOCKS=false`；尚未实现的媒体业务接口会返回 404。

## 已实现接口

- `GET /api/auth/setup/status`、`POST /api/auth/setup`：一次性单用户初始化
- `POST /api/auth/login`、`GET /api/auth/me`、`POST /api/auth/logout`
- `PATCH /api/auth/account`、`POST /api/auth/password`
- `GET|POST /api/auth/api-keys`、`DELETE /api/auth/api-keys/{id}`：创建、查看和撤销集成密钥
- `GET /api/settings/{emby|115|organize|search|notifications}`
- `PUT /api/settings/{emby|115|organize|search|notifications}`
- `GET /api/v1/whoami`：验证网页会话或 API Key
- `GET /api/logs?level=INFO&source=auth&query=login&limit=200`：读取 SQLite 服务请求日志

所有受保护接口使用 `Authorization: Bearer <token>`。数据库只保存 session token 的 SHA-256 摘要；密码使用 bcrypt 哈希；敏感配置使用本地 AES-GCM 密钥加密。设置读取只返回 `configuredSecrets` 标记，不返回密钥明文；保存时空字符串表示保持现有密钥不变。

API Key 以 `mc_` 开头，数据库仅保存 SHA-256 摘要；完整密钥只在创建响应中返回一次。网页会话可管理密钥，API Key 仅用于 `/api/v1` 应用接口，不能管理账号、设置或其他 API Key。应用接口应统一挂载到 `requireAPI` 鉴权组，因而支持网页会话和 Hermes 等集成客户端。影视搜索、下载等媒体业务接口尚未接入；当前 `/api/v1/whoami` 仅用于验证认证链路。

日志接口由服务端记录请求时间、模块、级别、路径、状态码和耗时；最多保留最近 5000 条，不记录请求头或查询参数。日志查询支持级别、模块、内容和数量筛选。

验证密钥：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/whoami -Headers @{ Authorization = "Bearer <API_KEY>" }
```

## 项目约定

HTTP 路由和输入校验放在 `internal/httpapi`，领域 DTO 放在 `internal/media`，SQLite schema 和访问逻辑放在 `internal/store`。外部服务适配器与媒体业务接口后续按相同模块边界增加。
