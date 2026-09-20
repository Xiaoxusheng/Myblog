# MyBlog API 契约（唯一真相来源）

后端与两个前端（web/admin）均以本文档为准。字段命名一律 **camelCase**。

## 通用约定

- Base URL：`/api/v1`；RSS 特例 `GET /rss`（根路径，返回 XML）
- 所有业务响应 HTTP 200，body：`{"code":0,"message":"ok","data":...}`
- HTTP 401：未登录 / token 失效（前端统一跳登录）；HTTP 500：服务器错误
- 错误码：`0` 成功；`10001` 参数错误；`10002` 未认证；`10003` 无权限；`10004` 资源不存在；`10005` 内容冲突（并发编辑保护）；`20001` 用户名或密码错误；`20002` 评论已关闭；`20003` 操作过于频繁
- 防护层（反爬/恶意请求/自动封禁，环境变量 `BLOG_RATE_LIMIT_PER_MIN`/`BLOG_WAF_MODE`/`BLOG_BAN_*`）：命中封禁或 WAF 规则 → HTTP 403 + `{"code":10003,"message":"请求被拒绝"}`；超过全局限流 → HTTP 429 + `{"code":20003,...}` + `Retry-After`。违规按次计点，窗口内达阈值自动封禁（封禁持久化，重启不失效；防护事件仅内存）
- 分页：query `page`(默认1)、`pageSize`(默认10，上限50)；响应 data：`{"list":[...],"total":123,"page":1,"pageSize":10}`
- 时间：RFC3339 字符串（如 `2026-09-19T12:00:00+08:00`）
- 鉴权：请求头 `Authorization: Bearer <token>`，仅 `/admin/*` 需要
- 前端错误提示分级（管理端 `admin/src/api/http.ts`）：
  - 默认：请求失败由 axios 拦截器统一弹全局错误 toast（业务错误码 / 500 / 网络失败）。
  - **`{ silent: true }`**：用于「后台轮询（如 60s 通知）」「页面次级面板（如搜索统计）」
    这类失败不应打断用户的请求 —— 拦截器不弹 toast，由调用方局部降级展示。
    注意 `silent` **不影响** 401/10002 的跳登录逻辑。
  - 约定：凡是用 `void fn()` fire-and-forget 触发的 async 函数，**必须**有 `catch`
    （或调用点 `.catch()`），否则会逃逸成 unhandledrejection。
    该规则由 `admin/scripts/check-void-async.mjs` 在 `npm run build` 前静态校验。
- 枚举：post.status `0`草稿 `1`已发布 `2`隐藏 `3`定时发布（到点由后端调度器自动置为 `1`，`publishedAt` = 计划时间；`3` 不会出现在任何公开接口）；comment.status `0`待审核 `1`已通过 `2`已拒绝 `3`垃圾 `4`回收站（公开接口仅返回 `1`）

## 对象结构

**PostSummary（列表项）**：`id,title,slug,summary,cover,viewCount,likeCount,status,isTop,createdAt,publishedAt,category:{id,name,slug},tags:[{id,name,slug}]`（不含 content）

**PostDetail**：PostSummary 全部字段 + `content`(Markdown 原文) + `updatedAt` + `seoTitle`/`seoDescription`/`canonical`/`ogImage`（SEO 扩展，空串=用默认规则）

**AdminPostItem**：PostSummary + `content` + `categoryId` + `tagNames:[string]` + `commentCount` + `publishAt`(定时发布的计划时间，RFC3339；其余状态为 `null`) + `seriesId`(0=不属于专题) + `seriesSort`(专题内序号) + `seoTitle` + `seoDescription` + `canonical` + `ogImage`（SEO 扩展字段，均可空串）

**PostRevisionItem**（版本列表项，不含 content）：`{id,postId,version,remark,createdAt}`；`remark` 为该版本的变更说明（如 `首次保存`、`修改标题、正文`、`恢复自 v3`）

**PostRevisionDetail**：PostRevisionItem + `{title,slug,summary,cover,content,categoryId,isTop,status}`（该版本保存时的文章全量快照）

**Series**：`{id,name,slug,description,cover,visible,sort,postCount?,createdAt,updatedAt}`；`postCount` 仅列表接口返回（公开列表=已发布数，管理列表=全部）

**Redirect**：`{id,source,target,type,enabled,createdAt,updatedAt}`；`source/target` 为站内路径（`/post/go-guide`），`type` 301|302

**文章详情 series 扩展**（仅当文章属于专题时返回）：`series:{id,name,slug,index,total}`（第 index/total 篇）、`seriesPrev:{id,title,slug}|null`、`seriesNext:{id,title,slug}|null`（按专题内序号相邻）

**Tag**：`{id,name,slug,postCount?}`；**Category**：`{id,name,slug,description?,postCount?}`

**公开评论 CommentPublic**：`{id,parentId,nickname,website,content,isAdmin,createdAt,children:[CommentPublic]}`（仅已通过；两级树，children 升序，顶级倒序）

**管理评论 CommentAdmin**：`{id,postId,postTitle,parentId,nickname,email,website,content,status,ip,isAdmin,createdAt}`

**Link**：`{id,name,url,logo,description,visible,sort,createdAt}`

**Page**：`{id,title,slug,content,status,createdAt,updatedAt}`（status 0 草稿 1 已发布）

**TimelineEvent**（技术时间线节点）：`{id,title,content,eventDate,image,postId,post:{id,title,slug}|null,projectName,projectUrl,visible,sort,createdAt,updatedAt}`；`eventDate` 为节点日期（RFC3339）；`postId`=0 或文章已删除时 post 为 null（公开侧仅关联已发布文章，管理侧关联任意状态并携带 status）；`content` 为 Markdown 原文，前台安全渲染；排序规则：`sort` 升序（默认 0，负值置顶）→ `eventDate` 倒序 → `id` 倒序（默认即时间倒序）

**Changelog**（版本发布记录）：`{id,version,title,content,releasedAt,status,sort,createdAt,updatedAt}`；`version` 兼容 SemVer（`1.2.0`/`v1.2.0-beta.1`，校验去 v 前缀后三段数字，存储统一去 v 前缀且唯一）；status 0 草稿 1 已发布；`content` 为 Markdown 原文，前台安全渲染；排序规则：`sort` 升序 → `releasedAt` 倒序 → `id` 倒序（默认即版本倒序）

**Upload**：`{id,url,filename,size,mime,createdAt}`，url 形如 `/uploads/202609/xxx.png`

**User**：`{id,username,nickname,email,avatar}`

**Settings（结构化对象，不是 map）**：`{siteName,siteDescription,siteKeywords,siteUrl,logo,notice,icp,footerText,commentEnabled(bool),postPageSize(number),autoRedirectOnSlugChange(bool)}`

---

## 公开接口（无需登录）

| # | 方法+路径 | 说明 |
|---|---|---|
| 1 | GET `/site` | 站点信息：Settings + `categories:[Category]` + `tags:[Tag]` |
| 2 | GET `/posts?keyword=&categoryId=&tagId=&page=&pageSize=&sort=` | 文章分页，仅已发布；`sort`=`newest`(默认)/`views`/`likes`；置顶优先；keyword 模糊匹配标题/摘要/内容（转义 %_\） |
| 3 | GET `/posts/:slug` | 详情：`{post:PostDetail,prev:{id,title,slug}\|null,next:...\|null,related:[PostSummary],series?:{id,name,slug,index,total},seriesPrev?...,seriesNext?...}`；:slug 同时接受数字 id 或 slug；浏览量 +1；非已发布 → 10004 |
| 4 | GET `/posts/:slug/comments` | `{list:[CommentPublic]}` 仅已通过 |
| 5 | POST `/posts/:slug/comments` | body `{parentId?,nickname,email,website?,content}`；昵称/邮箱/内容必填；评论开关关闭→20002；限流→20003；命中黑名单或垃圾规则（重复内容/链接过多/高频）→ status=3（垃圾，前台不可见）；否则 status=0；返回 `{comment:{id,parentId,nickname,website,content,createdAt}}` |
| 6 | POST `/posts/:slug/like` | likeCount +1，返回 `{likeCount}` |
| 7 | GET `/archive` | `[{year:2026,items:[{id,title,slug,createdAt}]}]` 按年倒序 |
| 8 | GET `/pages/:slug` | 已发布自定义页面 `{page:Page}` |
| 9 | GET `/links` | `{list:[Link]}` 仅 visible |
| 10 | GET `/rss`（根路径） | RSS 2.0 XML，最近 20 篇，链接用 settings.siteUrl |
| 48 | GET `/sitemap.xml`（根路径） | sitemap 0.9 XML：首页+已发布文章+已发布页面+全部分类/标签；siteUrl 为空回退请求 Host |
| 49 | GET `/robots.txt`（根路径） | 纯文本：Allow 全站、Disallow /api/ 与 /admin、声明 Sitemap 地址 |
| 83 | GET `/llms.txt`（根路径） | AEO：站点简介、核心页面、分类/专题（仅可见）、已发布文章索引（title+URL+摘要，≤200 篇），纯文本 |
| 86 | GET `/admin/auth/captcha` | 图形验证码（无需登录）：`{captchaId,image}`，image 为 SVG 字符串；单次有效，5 分钟过期。响应带 `Cache-Control: no-store`，禁止任何层缓存（否则客户端会拿到已消费的旧 id） |
| 85 | GET `/admin/analytics/searches?range=7d\|30d\|90d` | 站内搜索统计（默认 30d）：`{list:[{keyword,count,noResultCount}](≤20，count 降序)}`；来自公开搜索接口的关键词记录 |
| 84 | GET `/admin/health` | `{version,goVersion,dbType,dbStatus,postCount,commentCount,mediaCount,uploadSize,latestBackupAt}`；克制采集，不含磁盘空间等平台敏感信息 |
| 53 | GET `/series` | `{list:[Series]}` 仅 visible，sort 升序，postCount=已发布文章数 |
| 54 | GET `/series/:slug` | `{series:Series,posts:[PostSummary]}` 仅已发布文章，按专题内序号排列；404 → 10004 |
| 55 | GET `/redirects/resolve?path=/post/old` | `{redirect:{source,target,type}\|null}` 仅 enabled；无匹配返回 `{redirect:null}`（code 0）；供前台 404 兜底路由使用 |
| 66 | POST `/track` | 无鉴权埋点：body `{path, postId?, referer?}`（前台路由切换时上报，`navigator.sendBeacon` 或 fetch keepalive）。写 page_views：UA 解析 device/browser/os、referer 归类 `direct/search/github/social/other`、IP 仅存 SHA-256 哈希（不存明文）。同 IP 每分钟 ≤60 条，超限与非法输入返回 `{ok:false}`（HTTP 200，不打错误码） |
| 99 | GET `/timeline` | `{list:[TimelineEvent]}` 仅 visible，按 TimelineEvent 排序规则（默认时间倒序）；`post` 仅携带已发布文章 |
| 100 | GET `/changelog` | `{list:[Changelog]}` 仅 status=1，≤200 条，按 Changelog 排序规则（默认版本倒序） |

## 管理接口（Bearer）

### 认证
| 11 | POST `/admin/auth/login` | `{username,password,remember?,captchaId,captchaCode}` → `{token,user:User}`；验证码先校验（单次有效，错误→10001，不消耗失败计数）；**验证码失败时 message 区分两种情况**：题目不存在/已过期 → `验证码已过期，请点击图片刷新后重试`；答案不匹配 → `验证码不正确，请重新输入或点击图片刷新`（两种情况均返回 10001，且该题一律销毁）；remember=true 签发 7 天，否则 24h；同 IP 连续失败 5 次锁定 15 分钟→20003 |
| 12 | GET `/admin/auth/me` | `{user:User}` |
| 13 | PUT `/admin/auth/password` | `{oldPassword,newPassword}`(≥6位) → data:null |
| 14 | PUT `/admin/auth/profile` | `{nickname,email,avatar}` → `{user:User}` |

### 仪表盘
| 15 | GET `/admin/stats` | `{postCount,draftCount,scheduledCount,commentCount,pendingCommentCount,viewCount,likeCount,linkCount,trend:[{date:"2026-09-13",posts,comments}](近7天，按 created_at),recentComments:[{id,postTitle,nickname,content,status,createdAt}](5条),scheduledPosts:[{id,title,publishAt}](计划发布文章 ≤5 条，publishAt 升序),todayPv,todayUv,yesterdayPv,yesterdayUv}` |

### 文章
| 16 | GET `/admin/posts?keyword=&status=&categoryId=&tagId=&sort=&page=&pageSize=` | 分页 AdminPostItem，缺省按创建时间倒序；status 可为 `0/1/2/3`；`tagId` 按标签筛选（JOIN post_tags，同一文章不重复计数）；`sort=updatedAt` 按最后编辑时间倒序（草稿工作区用），其余取值忽略 |
| 17 | POST `/admin/posts` | `{title,slug?,summary?,content,cover?,categoryId,tags:[名称字符串],status,isTop,publishAt?,seriesId?,seriesSort?,seoTitle?,seoDescription?,canonical?,ogImage?}` → `{post:AdminPostItem}`；slug 空/重复则自动生成（post-{id} 或追加 -id）；tags 按 name upsert；status=3 时 publishAt 必填（RFC3339）；seriesId>0 加入该专题（seriesSort 0=自动排末尾）；创建成功即写入首个版本（v1，remark=首次保存） |
| 18 | GET `/admin/posts/:id` | `{post:AdminPostItem}` |
| 19 | PUT `/admin/posts/:id` | 同 17，全量更新（含 SEO 四字段；版本快照不含 SEO 字段）；可选 `auto:true`（前端自动保存标记）。**并发编辑保护**：可选 `baseUpdatedAt`（RFC3339，编辑器读取时的 `updatedAt`）；提供且与服务器当前 `updated_at` 不一致（比较前统一截断到毫秒，消除驱动精度差异）→ `10005`（文章已在其他窗口被修改），内容不被覆盖；缺省则跳过该检查（向后兼容）。版本生成规则：title/slug/summary/cover/content/categoryId/isTop/status 与最新版本相比有变化 → 自动生成新版本（remark 按变更字段生成，如 `修改标题、正文`）；`auto=true` 且距最新版本 < 120s → 仅保存内容不生成版本（防抖）；无变化不生成。status=3 时 publishAt 必填；status 非 3 时 publishAt 置空；seriesId 变更同步专题归属。**slug 变更且设置 `autoRedirectOnSlugChange` 开启（默认开）时自动创建 `/post/旧slug → /post/新slug` 301 重定向**。成功响应中的 `updatedAt` 即新的并发基线，客户端应回填后再发起下次保存 |
| 20 | PUT `/admin/posts/:id/status` | `{status,publishAt?}` → data:null；status=3 需 publishAt；状态实际变化时生成版本（remark=修改状态），仅调整计划时间不生成版本 |
| 21 | DELETE `/admin/posts/:id` | data:null；同时删除其 post_tags、评论与版本历史 |
| 50 | GET `/admin/posts/:id/revisions?page=&pageSize=` | 分页 PostRevisionItem，version 倒序；仅内容真正变化才产生版本 |
| 51 | GET `/admin/posts/:id/revisions/:version` | `{revision:PostRevisionDetail}`；版本不存在 → 10004 |
| 52 | POST `/admin/posts/:id/revisions/:version/restore` | 恢复版本：事务内先将当前内容快照为新版本（remark=恢复前快照，与最新版本相同则跳过），再应用目标版本并生成新版本（remark=恢复自 vN）→ `{post:AdminPostItem}`；任何恢复均可通过恢复「恢复前快照」撤销。恢复仅应用内容字段（标题/Slug/摘要/封面/正文/分类/置顶），**不改变当前发布状态与 publishAt**；目标 Slug 已被其他文章占用 → 10001 并回滚 |

### 分类 / 标签
| 22 | GET `/admin/categories?page=&pageSize=` | 分页，含 postCount |
| 23 | POST `/admin/categories` | `{name,slug?,description?}`；name 唯一 |
| 24 | PUT `/admin/categories/:id` | 同上 |
| 25 | DELETE `/admin/categories/:id` | 该分类下文章 categoryId 置 0 |
| 26 | GET `/admin/tags?page=&pageSize=` | 分页（受全局 pageSize≤50 约束，默认 50），含 postCount |
| 27 | POST `/admin/tags` | `{name,slug?}` |
| 28 | PUT `/admin/tags/:id` | 同上 |
| 29 | DELETE `/admin/tags/:id` | 同步清理 post_tags |

### 评论
| 30 | GET `/admin/comments?status=&postId=&page=&pageSize=` | 分页 CommentAdmin，最新在前；status 支持 0待审/1通过/2拒绝/3垃圾/4回收站 |
| 31 | PUT `/admin/comments/:id/status` | `{status}` → data:null；status 允许 0~4（0 恢复待审、1 通过、2 拒绝、3 标垃圾、4 回收站） |
| 32 | POST `/admin/comments/:id/reply` | `{content}`：以管理员身份（isAdmin=1，昵称取管理员 nickname，status=1 直接通过）回复，parentId=该评论 id，挂在其 postId 下 → `{comment:CommentAdmin}` |
| 33 | DELETE `/admin/comments/:id` | 连同其 children 一起删 |

### 友链 / 页面
| 34 | GET `/admin/links?page=` | 分页 |
| 35 | POST `/admin/links` | `{name,url,logo?,description?,visible,sort?}` |
| 36 | PUT `/admin/links/:id` | 同上 |
| 37 | DELETE `/admin/links/:id` | |
| 38 | GET `/admin/pages?page=&pageSize=` | 分页 |
| 39 | POST `/admin/pages` | `{title,slug,content,status}` slug 必填唯一 |
| 40 | GET `/admin/pages/:id` | `{page:Page}` |
| 41 | PUT `/admin/pages/:id` | 同 39 |
| 42 | DELETE `/admin/pages/:id` | |

### 设置 / 媒体
| 43 | GET `/admin/settings` | `{settings:Settings}`（key 缺省给默认值） |
| 44 | PUT `/admin/settings` | body 为 Settings 结构化对象 → data:null |
| 45 | POST `/admin/uploads` | multipart 字段名 `file`，仅图片，≤10MB → `{upload:Upload}` |
| 46 | GET `/admin/uploads?page=&pageSize=` | 分页 Upload，最新在前 |
| 47 | DELETE `/admin/uploads/:id` | 同时删文件 |

### 专题 / 重定向
| 56 | GET `/admin/series?page=&pageSize=` | 分页 Series（含 postCount=全部文章数），sort 升序 |
| 57 | POST `/admin/series` | `{name,slug?,description?,cover?,visible,sort?}`；name/slug 唯一，slug 空自动生成 |
| 58 | PUT `/admin/series/:id` | 同 57 |
| 59 | DELETE `/admin/series/:id` | 同步删除 series_posts 关联（文章本身不动） |
| 60 | GET `/admin/series/:id/posts` | `{list:[{id,title,slug,status,sort}]}` 全部状态，按 sort、id 升序 |
| 61 | PUT `/admin/series/:id/posts` | `{items:[{postId,sort}]}` 批量调整专题内序号（仅已在该专题的文章生效）→ data:null |
| 62 | GET `/admin/redirects?page=&pageSize=` | 分页 Redirect，createdAt 倒序 |
| 63 | POST `/admin/redirects` | `{source,target,type,enabled}`；source/target 必须以 `/` 开头的站内路径（去 query/hash）；type 301\|302；source≠target；**环检测**：沿 target 链回溯若回到 source 或超过 10 层 → 10001 |
| 64 | PUT `/admin/redirects/:id` | 同 63（环检测排除自身后校验） |
| 65 | DELETE `/admin/redirects/:id` | data:null |
| 69 | GET `/admin/comment-blacklist?type=ip\|email\|keyword&page=&pageSize=` | 分页黑名单：`{id,type,value,createdAt}`，createdAt 倒序 |
| 70 | POST `/admin/comment-blacklist` | `{type,value}`；type 限 ip/email/keyword；value 必填 ≤200 字；(type,value) 唯一 → `{item}` |
| 71 | DELETE `/admin/comment-blacklist/:id` | data:null |
| 72 | POST `/admin/comments/batch` | `{action:"approve"\|"reject"\|"spam"\|"delete", ids:[number]}`（ids ≤100）→ `{updated:int64}`；approve→1、reject→2、spam→3；delete→物理删除（连同 children） |
| 73 | GET `/admin/notifications` | `{list:[{id,type,title,content,link,read,createdAt}](≤10 条最新),unreadCount:int64}`；type：comment_pending（新待审评论）/comment_spam（捕获垃圾）/post_published（定时文章已上线）/backup（备份完成或失败） |
| 74 | PUT `/admin/notifications/read-all` | 全部标记已读 → data:null |
| 75 | PUT `/admin/notifications/:id/read` | 单条已读 → data:null |
| 76 | GET `/admin/audit-logs?page=&pageSize=&action=` | 操作日志分页：`{id,action,resourceType,resourceId,description,ipHash,createdAt}`，最新在前；action 可前缀过滤（如 `post.`）；不记录密码/JWT/完整请求体 |
| 77 | GET `/admin/backups` | `{list:[{name,size,createdAt,type}]}` type：database(文件)/full(zip)，createdAt 倒序 |
| 78 | POST `/admin/backups` | `{type:"database"\|"full"}` → `{item:{name,size,createdAt,type}}`；database=复制数据库文件（仅 SQLite；MySQL 模式返回 10001 提示使用导出，不伪造）；full=数据库+uploads 打包 zip；成功/失败写通知(type=backup) |
| 79 | GET `/admin/backups/:name/download` | 备份文件流下载；name 白名单校验（防路径穿越） |
| 80 | DELETE `/admin/backups/:name` | 删除备份文件 |
| 81 | GET `/admin/export` | 全站导出 zip：`posts.json/pages.json/categories.json/tags.json/links.json/comments.json/settings.json` + `media/`（uploads 原样） |
| 82 | POST `/admin/import` | multipart 字段 `file`（zip，≤50MB，条目 ≤5000、解压总量 ≤200MB 防炸弹）；query `dryRun=true` 仅预览返回 `{summary:{posts:N,...},conflicts:[{type,value}]}`；`strategy=skip`(默认，冲突跳过)\|`update`(按 slug/名称更新已有)；媒体存入 uploads 跳过同名 → `{updated:int64,summary}` |
| 67 | GET `/admin/analytics?range=today\|7d\|30d\|90d` | 访问分析（默认 7d）：`{range,totals:{pv,uv},trend:[{date,pv,uv}](按日分桶),topPosts:[{postId,title,pv,uv,likeCount,commentCount}](≤10，pv 降序),sources:[{source,pv}] (direct/search/github/social/other),devices:[{device,pv}](desktop/mobile/tablet),browsers:[{browser,pv}],oses:[{os,pv}]}`；空数据返回空数组，不伪造 |
| 68 | GET `/admin/analytics/posts/:id?range=7d\|30d\|90d` | 单篇文章分析：`{post:{id,title},range,totals:{pv,uv,likeCount,commentCount},trend:[{date,pv,uv}],sources:[{source,pv}],devices:[{device,pv}]}`；文章不存在 → 10004 |

### 安全防护（反爬 / 恶意请求 / IP 封禁）
| 87 | GET `/admin/security/bans` | 生效中的封禁列表：`{list:[{ip,reason,source,createdAt,expiresAt}]}`；source：`auto`（自动）/`manual`（手动）；expiresAt `null` = 永久 |
| 88 | POST `/admin/security/bans` | `{ip,durationMinutes?,reason?}` 手动封禁 → `{list}`；ip 必须合法；durationMinutes 缺省/`0`=永久，≤525600；**禁止封禁白名单 IP 与当前登录 IP** → 10001；写审计日志 `security.ban` |
| 89 | DELETE `/admin/security/bans/:ip` | 解除封禁 → `{list}`；未封禁 → 10004；写审计日志 `security.unban` |
| 90 | GET `/admin/security/events?limit=` | 最近防护事件（默认 100，≤300，仅进程内存、重启清零）：`{list:[{time,ip,kind,detail}]}`，kind：`waf`/`rate_limit`/`login_block`/`auto_ban`/`manual_ban`/`unban`，新→旧 |

### 时间线 / 版本记录
| 91 | GET `/admin/timeline?page=&pageSize=` | 分页 TimelineEvent（含隐藏节点），排序同公开规则 |
| 92 | POST `/admin/timeline` | `{title,content?,eventDate,image?,postId?,projectName?,projectUrl?,visible,sort?}` → `{item:TimelineEvent}`；title 必填 ≤100 字；content ≤5000 字；eventDate 必填（RFC3339）；image/projectUrl ≤512；projectName ≤100；postId>0 时文章须存在 → 10001 |
| 93 | PUT `/admin/timeline/:id` | 同 92 全量更新 |
| 94 | DELETE `/admin/timeline/:id` | data:null |
| 95 | GET `/admin/changelogs?page=&pageSize=&status=` | 分页 Changelog（status 缺省=全部，含草稿） |
| 96 | POST `/admin/changelogs` | `{version,title?,content?,releasedAt,status,sort?}` → `{item:Changelog}`；version 必填且 SemVer 兼容（非法 → 10001）、重复 → 10001；title ≤100；content ≤20000；releasedAt 必填（RFC3339）；status 0\|1 |
| 97 | PUT `/admin/changelogs/:id` | 同 96（version 唯一性排除自身） |
| 98 | DELETE `/admin/changelogs/:id?force=` | data:null；status=1（已发布）且未带 `force=true` → 10001 删除保护（前端二次确认后携带 force） |

## 联调冒烟清单（QA 用）

1. `go run .` 启动后 `curl :8080/api/v1/site` → code 0
2. 登录拿 token → 创建文章（含标签）→ 公开列表可见 → 详情 viewCount+1
3. 游客评论 → admin 待审 1 条 → 通过 → 公开可见且为树
4. 上传图片 → 返回 url 可访问
5. `curl :8080/rss` → XML
6. 编辑文章改标题保存 → 版本列表 v2（remark 含 标题）→ 恢复 v1 → 文章标题还原且产生恢复版本
7. 创建 status=3 文章（publishAt 过去时间）→ 等 1 个调度周期 → 自动变已发布且公开列表可见
8. 创建专题 → 两篇文章设 seriesId=1/2 → `/series/:slug` 按序返回；文章详情含 series.index/total 与 seriesPrev/Next
9. 创建重定向 `/post/a → /post/b`（301）→ `/redirects/resolve?path=/post/a` 返回 target；再建 `/post/b → /post/a` → 10001 环检测
10. 修改文章 slug → 自动出现旧 slug→新 slug 的 301 重定向
11. 前台页面多次访问（/track 上报）→ `/admin/analytics` PV/UV/热门文章/来源/设备 与实际一致
12. 防护：`curl -A sqlmap :8080/api/v1/site` → 403；快速请求超过 `BLOG_RATE_LIMIT_PER_MIN` → 429 且计点达阈值自动封禁 → 后续 403；登录拿 token → `POST /admin/security/bans` 手动封禁某 IP → 该 IP 请求 403 → `DELETE` 解封恢复；`GET /admin/security/events` 可见 waf/rate_limit/auto_ban 事件
13. 时间线：创建节点（含关联文章、sort=-1 置顶）→ `GET /api/v1/timeline` 首位可见且带 post 标题；PUT visible=false → 前台消失；DELETE → 消失；postId 不存在 → 10001
14. 版本记录：创建 v1.0.0（status=0）→ `GET /api/v1/changelog` 不可见；PUT status=1 → 可见；再创建 v1.10.0（releasedAt 更晚）→ 排在 v1.0.0 之前；DELETE 已发布记录不带 force → 10001，带 force=true → 删除成功；version 重复/非法 → 10001
15. 草稿工作区与并发保护：`GET /admin/posts?tagId={标签id}&sort=updatedAt` → 仅返回带该标签的文章且按最后编辑倒序；PUT 更新文章带服务器返回的 `updatedAt` 作 `baseUpdatedAt` → 成功并返回新的 `updatedAt`；再用**旧** `baseUpdatedAt` 重发 → `10005` 且内容未被覆盖；不带 `baseUpdatedAt` 重发 → 成功（向后兼容）；`PUT /admin/posts/:id/status` 等状态变更后 `updatedAt` 应同步刷新
