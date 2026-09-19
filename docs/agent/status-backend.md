# Status — Backend（server/ + Docker 部署物）

负责人：Backend Agent
日期：2026-09-19

## 完成了什么

1. **`server/` 从零实现全部 47 项契约接口**（Go 1.27 + Gin + GORM + glebarez/sqlite 纯 Go 驱动，
   module `myblog/server`），统一响应 `{code,message,data}`（业务错误 HTTP 200、鉴权失败 401、
   内部错误 500），全字段 camelCase，分页 `{list,total,page,pageSize}` 且 pageSize≤50。
   - 分层：`config/ common/ model/ middleware/ handler/ router/ uploads/`，handler 直接 GORM（无 service 层）。
   - 环境变量：`BLOG_PORT`(8080)、`BLOG_JWT_SECRET`(dev-secret)、`BLOG_DB_TYPE`(sqlite/mysql)、
     `BLOG_DB_PATH`(blog.db)、`BLOG_MYSQL_DSN`、`BLOG_UPLOAD_DIR`(uploads)。
     零配置 `go run .` 用 SQLite 直接跑；MySQL 分支有完整代码路径（DSN 缺
     `charset=utf8mb4&parseTime=True&loc=Local` 时校验并自动补齐+警告；连接池单独调参）。
   - 业务规则全部落实：slug 空/重复自动生成（post-{id} / 追加 -id）、首次发布写 published_at、
     详情浏览量+1、相关文章=同分类取5、归档按年倒序、评论两级树（顶级倒序 children 升序、仅已通过、
     深层回复扁平化到顶级）、管理员回复 isAdmin=1 直接通过、删评论连带子孙、删文章连带 post_tags+评论、
     删分类文章 categoryId 置 0、删标签清理 post_tags、上传仅图片≤10MB 存 `uploads/YYYYMM/随机名.扩展名`
     （魔数嗅探+扩展名白名单，SVG 因脚本风险不放开）、RSS 2.0 根路径 /rss（近20篇，链接用 siteUrl）、
     `/uploads` 静态服务、CORS 全放开、LIKE 转义 `% _ \`（显式 `ESCAPE '\'`）、游客评论同 IP 60s 限流
     （成功入库才占用窗口，map 过期清理防无界增长）。
   - Seed 幂等：admin/admin123（bcrypt）、未分类、Go/Gin/Vue3/随笔、3 篇已发布中文示例文章
     （各 300+ 字，含二级标题与 Go/JS 代码块，发布时间错开 24/48/72h 便于仪表盘趋势展示）+1 草稿、
     3 条已通过评论（含 1 条管理员回复）、3 友链、关于页（slug=about）、Settings 默认值。
   - 优雅关闭（SIGINT/SIGTERM + Shutdown 超时兜底）。

2. **测试 33 个用例全绿**：httptest + SQLite 内存库（`file:memdbN?mode=memory&cache=shared`，
   MaxOpenConns=1 保活），每个用例独立建库+Seed。

3. **Docker 部署物全套**（见下方清单），支持「compose 自带 MySQL」（profile bundled）与
   「复用服务器已有 MySQL」两种模式。

## 新建文件清单

```
server/
├── Dockerfile                    # 多阶段构建：golang:1.27-alpine(GOPROXY=goproxy.cn, CGO_ENABLED=0) → alpine:3.20
│                                 #   + ca-certificates/tzdata(Asia/Shanghai) + 非 root 用户；EXPOSE 8080
├── go.mod  go.sum
├── main.go                       # 启动装配 + 优雅关闭
├── config/config.go              # 环境变量 + MySQL DSN 校验补齐
├── common/                       # errcode.go response.go page.go slug.go
├── model/                        # user post taxonomy comment content(链接/页面/设置/上传) db dto seed(8 文件)
├── middleware/                   # cors.go jwt.go ratelimit.go
├── handler/                      # public_site/post/comment/misc + rss + admin_auth/stats/post/taxonomy/
│                                 #   comment/link/page/setting/upload + common.go（15 文件）
├── router/router.go              # 全部路由装配（/api/v1/*、/uploads/*、/rss）
├── uploads/.gitkeep
└── *_test.go                     # main/auth/post/comment/cascade/upload/misc（7 个测试文件）

deploy/
├── Dockerfile                    # 门户镜像：node:20-alpine 构建 web/ + admin/ dist（npmmirror）→ nginx:alpine
└── nginx.conf                    # 80=前台 / 8081=管理端，SPA 回退，反代 /api /uploads /rss → server:8080，20m

docker-compose.yml                # mysql(profiles:["bundled"],healthcheck) + server(mysql DSN,上传卷,8080) + nginx(80,8081)
.env.example                      # MYSQL_ROOT_PASSWORD / BLOG_JWT_SECRET（带"请修改"注释）
docs/03-部署文档.md               # 模式 A 全套 / 模式 B 复用已有 MySQL（含建库 SQL、备份、运维命令）
```

## 测试与验证结果

| 命令 | 结果 |
|---|---|
| `go fmt ./...` | 通过（无 diff） |
| `go vet ./...` | 通过 |
| `go test ./...` | **33 个用例全部 PASS**（`ok myblog/server`） |
| `go build ./...` | 通过 |
| `go test -race` | **未验证**——本机无 gcc，race 检测无法编译（环境限制，非代码问题） |
| 冒烟（`go run` 起服务后 curl，测完即停） | `/api/v1/site` code 0；`/rss` 返回 XML；登录返回 token；无 token 访问 `/admin/stats` → HTTP 401 |
| `docker compose config -q` / `docker build` | **本地未验证，需服务器验证**——本机无 docker；已用 PyYAML 校验 docker-compose.yml 语法与结构（services/volumes/profiles/depends_on 均正确） |

### 测试覆盖点（对应用户要求的 9 项 + 扩展）

1. 登录成功 / 密码错误 → 20001（TestLoginSuccess/TestLoginWrongPassword）
2. 未带 token / 伪造 token 访问 5 条 /admin 路径 → HTTP 401 + code 10002（TestAdminRequiresAuth）
3. 公开列表：分页结构、置顶优先、草稿不可见、keyword 命中标题/内容、特殊字符 `% _ a\b 100% __` 不报错
   且 `%` 按字面量处理（TestPublicPostListPaginationAndTop/TestPublicPostKeywordFilter）
4. 创建文章（带标签）→ 详情可见 → 浏览量 +1 → 点赞递增回显（TestPostCreateDetailViewChildLike）
5. 游客评论 → 待审（公开不可见、管理端可见含 IP）→ 通过 → 公开树可见（TestGuestCommentFlow）
6. 管理员回复 isAdmin=1 直接公开可见 + 种子树结构断言（TestAdminReplyVisible）
7. 级联：删文章连带 post_tags+评论（DB 行数断言）、删分类 categoryId 置 0、删标签清 post_tags、
   删评论连带 children（TestDeletePostCascades 等 4 个）
8. 上传：内存 multipart 假 PNG（PNG 魔数）成功且静态可访问内容一致、txt/假 PNG 拒绝、>10MB 拒绝、
   列表+删除清文件（TestUploadPNG 等 4 个）
9. RSS 返回 XML 2.0、3 item、无草稿（TestRSS）

另覆盖：me/profile、改密码全流程、仪表盘统计与 7 天趋势、数字 id 访问详情 + prev/next、
slug 自动生成（空/重复）、管理端列表过滤与 AdminPostItem 字段、状态切换与隐藏 10004、
评论关闭 20002、限流 20003、评论校验 10001、分类/标签名唯一 10001、site 端点 postCount、
归档分组倒序、公开页面（草稿/不存在 10004）、友链 visible+sort、设置读写回环。

## 契约偏离 / 需 Orchestrator 知悉的事项

1. **admin 前端需改一处（后端按契约实现）**：契约 #45 上传返回 `data:{upload:{...}}`，而
   `admin/src/api/media.ts` 把 `uploadImage` 声明为直接返回 `UploadItem`，`ProfileView.vue`(约 60 行)
   与 `components/MarkdownEditor.vue`(约 31 行) 读了 `result.url`。联调时需改为 `result.upload.url`。
2. **契约内部冲突（未改契约，实现取保守值）**：#26 标签管理「pageSize 默认 100」与全局约定
   「pageSize 上限 50」矛盾；实现统一按上限 50 夹取（默认 100 传入后夹为 50）。若确需 100，
   需 Orchestrator 决策后先改契约。
3. **创建/更新类接口返回体**（契约未明确的部分）：分类/标签/友链/页面的创建与更新直接返回对象本身
   （对齐 admin 前端预期，PageEditView 依赖 `created.id`）；文章/评论回复/上传按契约包装
   `{post}` / `{comment}` / `{upload}`。
4. prev/next 方向契约未明确，实现为 prev=较早发布、next=较晚发布（已测试锁定）。
5. 归档按「年」分组（契约 #7）；需求文档 1.3 写「按年月」，以契约为准。
6. 上传不支持 SVG（魔数白名单 png/jpg/gif/webp/bmp）——SVG 内嵌脚本经同源静态服务有 XSS 风险。
7. 登录接口无防爆破限流（契约无此要求；单管理员风险低，记为剩余风险）。

## 未完成 / 未验证事项

- `docker compose config -q`、`docker build`、容器端到端验证：**本地未验证**（无 docker），
  需按 docs/03-部署文档.md 在服务器执行；deploy/Dockerfile 依赖 web/ 与 admin/ 的
  package.json 与 `npm run build`（admin 已就绪，web 由 Web Agent 提供后即可构建）。
- MySQL 真实连通性：按任务约定本地仅测 SQLite；MySQL 分支代码路径（DSN 校验/补齐、连接池、
  gorm mysql driver）已就绪，待服务器验证。
- `go test -race`：本机无 C 编译器无法运行。
- 无阻塞项，未写 blocked.md。
