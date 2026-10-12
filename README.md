# Media Center API

给 `media-center-web` 提供 API 的 Gin 单体后端，使用 SQLite 保存单用户、登录会话和分组配置。

## 首次启动

安装 Go 1.26 或更新版本后，在此目录运行：

```powershell
go mod tidy
go run ./cmd/server
```

服务默认监听 `http://localhost:8080`。首次打开前端时会显示创建管理员账号页面；账号 3–32 位，密码不能为空且最多 72 字节（bcrypt 限制）。初始化后可用新账号登录。健康检查地址是 `GET /api/health`。

默认数据保存在项目目录的 `data/`：`media-center.db` 是 SQLite 数据库，`master.key` 用于加密 TMDB/Emby API Key、115 Cookie 和 Telegram Bot Token。备份和迁移时需要一起保存这两个文件；丢失密钥后无法解密已存的服务凭证。可用 `MEDIA_CENTER_DATA_DIR` 指定其他持久化目录，用 `HTTP_ADDR` 指定监听地址。

## 前端连接

在 `media-center-web` 目录启动开发服务器时，开启真实登录和配置 API、保留其他媒体页面的 mock：

```powershell
$env:VITE_BACKEND_AUTH = 'true'
pnpm dev
```

Vite 默认把 `/api` 代理到 `127.0.0.1:18080`；后端请设置 `HTTP_ADDR=127.0.0.1:18080`，或用 `VITE_API_PROXY_TARGET` 指向实际后端地址。如需让所有 API 请求都直连后端，可设 `VITE_ENABLE_MOCKS=false`；尚未实现的媒体业务接口会返回 404。

## 已实现接口

- `GET /api/auth/setup/status`、`POST /api/auth/setup`：一次性单用户初始化
- `POST /api/auth/login`、`GET /api/auth/me`、`POST /api/auth/logout`
- `PATCH /api/auth/account`、`POST /api/auth/password`
- `GET|POST /api/auth/api-keys`、`DELETE /api/auth/api-keys/{id}`：创建、查看和撤销集成密钥
- `GET /api/settings/{tmdb|emby|115|organize|search|notifications}`
- `PUT /api/settings/{tmdb|emby|115|organize|search|notifications}`
- `GET /api/v1/whoami`：验证网页会话或 API Key
- `GET /api/logs?level=INFO&source=auth&query=login&limit=200`：读取 SQLite 服务请求日志

所有受保护接口使用 `Authorization: Bearer <token>`。数据库只保存 session token 的 SHA-256 摘要；密码使用 bcrypt 哈希；敏感配置使用本地 AES-GCM 密钥加密。设置读取只返回 `configuredSecrets` 标记，不返回密钥明文；保存时空字符串表示保持现有密钥不变。

API Key 以 `mc_` 开头，数据库仅保存 SHA-256 摘要；完整密钥只在创建响应中返回一次。网页会话可管理密钥，API Key 仅用于 `/api/v1` 应用接口，不能管理账号、设置或其他 API Key。应用接口应统一挂载到 `requireAPI` 鉴权组，因而支持网页会话和 Hermes 等集成客户端。TMDB 元数据探索接口使用 `/api/media` 网页会话鉴权；下载与资源搜索尚未接入；当前 `/api/v1/whoami` 仅用于验证认证链路。

日志接口由服务端记录请求时间、模块、级别、路径、状态码和耗时；最多保留最近 5000 条，不记录请求头或查询参数。日志查询支持级别、模块、内容和数量筛选。

验证密钥：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/whoami -Headers @{ Authorization = "Bearer <API_KEY>" }
```

## 项目约定

HTTP 路由和输入校验放在 `internal/httpapi`，领域 DTO 放在 `internal/media`，SQLite schema 和访问逻辑放在 `internal/store`。外部服务适配器与媒体业务接口后续按相同模块边界增加。

## TMDB 电影/电视剧探索（2026-10-12）

在全局设置的 TMDB 分类保存官方 v3 API Key（`tmdb.apiKey`），沿用 AES-GCM、空字符串保留密钥及 `configuredSecrets` 语义。浏览器的设置读取只得到已配置标记；上游错误正文、凭证 URL 和密钥不写入响应或日志。无密钥时返回 503，不生成替代作品。

- `GET /api/media/discover?source=tmdb&mediaType=movie&page=1&pageSize=20`：mediaType 必填 movie/tv；page 为 1–500；pageSize 仅接受 20。
- 可选 genre 为相应原生类型的 TMDB 类型 ID；region 是产地 CN/HK/TW/JP/KR/US/GB；year 为 1870–2100；sort 为 popularity.desc、vote_average.desc、date.desc。服务器映射电影/电视剧各自的日期排序与年份参数，不做页面内过滤。
- query 使用对应 search/movie 或 search/tv，最多 200 字符，支持年份；与 genre、region、非热度排序组合返回 400。
- `GET /api/media/tmdb/{providerId}?mediaType=movie|tv`：原生正整数 ID，类型必填；详情包含真实演员和季信息，缺失字段不补演示数据。

契约变化：身份新增必填 provider/providerId，保留 source/sourceId 兼容别名；mediaId=`tmdb:{movie|tv}:{原生ID}`。探索 items 直接使用正式 Summary（无需 id/kind/rating/summary/palette/artwork 展示兼容字段），people 移除。page/pageSize/total/hasMore/source/fetchedAt/stale 保留；total 原样使用 total_results，缺失为 null；缺失/错误 page 或 total_pages 返回 502，不推算总数。hasMore 使用上游 total_pages，并受 500 页访问边界限制。评分无投票时返回空数组，未知年份省略。

错误为 `{message}`：参数 400、会话 401、作品不存在 404、请求取消 408、上游错误或格式异常 502、未配置/上游凭证失效/上游限流 503、超时 504。503 限流响应附有界 Retry-After，provider 401 不映射成浏览器登录失效。

客户端 HTTP 超时 8 秒、整体预算 15 秒；共享限流 4 次/秒、并发最多 2；429/5xx 最多 3 次尝试，指数退避并解析 Retry-After 秒数或 HTTP 日期，等待超过 2 秒时直接返回。请求和等待均服从调用方取消，禁止自动跟随重定向。上游响应体限制 1 MiB。

相对设计文档的首版取舍：只实现 TMDB 单类型探索，不实现 all 混合流、人物或榜单。使用有界进程内 TTL 缓存（最多 128 条，容量达到上限清空），探索 30 分钟、详情 24 小时；不新增 SQLite 缓存表、SWR 或后台刷新。缓存键包含密钥摘要、类型、参数，配置更换不会复用旧凭证缓存；只缓存校验成功的上游 JSON，缓存命中保留 fetchedAt，stale 恒为 false。重启会丢失缓存，也没有同键请求合并。

### 本轮验证

全部 TMDB 测试使用随机生成的测试凭证和本地假上游。沙箱禁止监听端口，因此使用 `net/http/httptest.ResponseRecorder` + 注入 Transport，在进程内驱动上游 Handler；不联网，不使用真实密钥。测试覆盖参数、同号身份、归一化、真实/未知分页、缓存/过期/配置更换、取消/超时、错误/有界重试及设置密钥读取脱敏。

设置 `PATH=/home/yuxh/.local/bin:$PATH`、`GOCACHE=/tmp/media-center-go-build` 后，实际执行 `go build ./...`、`go vet ./...` 均退出 0、无诊断；`go test ./...` 退出 0，httpapi 与 tmdb 测试通过，其余包无测试。默认 Go 缓存目录不可写，首次尝试失败后改用上述可写目录。

前端实际执行 typecheck、lint、build 均退出 0；Vite 成功构建，提示主 JS chunk 超过 500 kB。尚未验证真实 TMDB API、图片 CDN、部署出站网络或浏览器点击联调；提供密钥后才能做这些验收。未接入豆瓣/Bangumi/AniList、资源搜索或下载。
