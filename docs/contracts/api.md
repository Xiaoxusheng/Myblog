# MyBlog API 契约（唯一真相来源）

后端与两个前端（web/admin）均以本文档为准。字段命名一律 **camelCase**。

## 通用约定

- Base URL：`/api/v1`；RSS 特例 `GET /rss`（根路径，返回 XML）
- 所有业务响应 HTTP 200，body：`{"code":0,"message":"ok","data":...}`
- HTTP 401：未登录 / token 失效（前端统一跳登录）；HTTP 500：服务器错误
- 错误码：`0` 成功；`10001` 参数错误；`10002` 未认证；`10003` 无权限；`10004` 资源不存在；`20001` 用户名或密码错误；`20002` 评论已关闭；`20003` 操作过于频繁
- 分页：query `page`(默认1)、`pageSize`(默认10，上限50)；响应 data：`{"list":[...],"total":123,"page":1,"pageSize":10}`
- 时间：RFC3339 字符串（如 `2026-09-19T12:00:00+08:00`）
- 鉴权：请求头 `Authorization: Bearer <token>`，仅 `/admin/*` 需要
- 枚举：post.status `0`草稿 `1`已发布 `2`隐藏；comment.status `0`待审核 `1`已通过 `2`已拒绝

## 对象结构

**PostSummary（列表项）**：`id,title,slug,summary,cover,viewCount,likeCount,status,isTop,createdAt,publishedAt,category:{id,name,slug},tags:[{id,name,slug}]`（不含 content）

**PostDetail**：PostSummary 全部字段 + `content`(Markdown 原文) + `updatedAt`

**AdminPostItem**：PostSummary + `content` + `categoryId` + `tagNames:[string]` + `commentCount`

**Tag**：`{id,name,slug,postCount?}`；**Category**：`{id,name,slug,description?,postCount?}`

**公开评论 CommentPublic**：`{id,parentId,nickname,website,content,isAdmin,createdAt,children:[CommentPublic]}`（仅已通过；两级树，children 升序，顶级倒序）

**管理评论 CommentAdmin**：`{id,postId,postTitle,parentId,nickname,email,website,content,status,ip,isAdmin,createdAt}`

**Link**：`{id,name,url,logo,description,visible,sort,createdAt}`

**Page**：`{id,title,slug,content,status,createdAt,updatedAt}`（status 0 草稿 1 已发布）

**Upload**：`{id,url,filename,size,mime,createdAt}`，url 形如 `/uploads/202609/xxx.png`

**User**：`{id,username,nickname,email,avatar}`

**Settings（结构化对象，不是 map）**：`{siteName,siteDescription,siteKeywords,siteUrl,logo,notice,icp,footerText,commentEnabled(bool),postPageSize(number)}`

---

## 公开接口（无需登录）

| # | 方法+路径 | 说明 |
|---|---|---|
| 1 | GET `/site` | 站点信息：Settings + `categories:[Category]` + `tags:[Tag]` |
| 2 | GET `/posts?keyword=&categoryId=&tagId=&page=&pageSize=&sort=` | 文章分页，仅已发布；`sort`=`newest`(默认)/`views`/`likes`；置顶优先；keyword 模糊匹配标题/摘要/内容（转义 %_\） |
| 3 | GET `/posts/:slug` | 详情：`{post:PostDetail,prev:{id,title,slug}\|null,next:...\|null,related:[PostSummary]}`；:slug 同时接受数字 id 或 slug；浏览量 +1；非已发布 → 10004 |
| 4 | GET `/posts/:slug/comments` | `{list:[CommentPublic]}` 仅已通过 |
| 5 | POST `/posts/:slug/comments` | body `{parentId?,nickname,email,website?,content}`；昵称/邮箱/内容必填；评论开关关闭→20002；限流→20003；新评论 status=0；返回 `{comment:{id,parentId,nickname,website,content,createdAt}}` |
| 6 | POST `/posts/:slug/like` | likeCount +1，返回 `{likeCount}` |
| 7 | GET `/archive` | `[{year:2026,items:[{id,title,slug,createdAt}]}]` 按年倒序 |
| 8 | GET `/pages/:slug` | 已发布自定义页面 `{page:Page}` |
| 9 | GET `/links` | `{list:[Link]}` 仅 visible |
| 10 | GET `/rss`（根路径） | RSS 2.0 XML，最近 20 篇，链接用 settings.siteUrl |
| 48 | GET `/sitemap.xml`（根路径） | sitemap 0.9 XML：首页+已发布文章+已发布页面+全部分类/标签；siteUrl 为空回退请求 Host |
| 49 | GET `/robots.txt`（根路径） | 纯文本：Allow 全站、Disallow /api/ 与 /admin、声明 Sitemap 地址 |

## 管理接口（Bearer）

### 认证
| 11 | POST `/admin/auth/login` | `{username,password,remember?}` → `{token,user:User}`；remember=true 签发 7 天，否则 24h；同 IP 连续失败 5 次锁定 15 分钟→20003 |
| 12 | GET `/admin/auth/me` | `{user:User}` |
| 13 | PUT `/admin/auth/password` | `{oldPassword,newPassword}`(≥6位) → data:null |
| 14 | PUT `/admin/auth/profile` | `{nickname,email,avatar}` → `{user:User}` |

### 仪表盘
| 15 | GET `/admin/stats` | `{postCount,draftCount,commentCount,pendingCommentCount,viewCount,likeCount,linkCount,trend:[{date:"2026-09-13",posts,comments}](近7天，按 created_at),recentComments:[{id,postTitle,nickname,content,status,createdAt}](5条)}` |

### 文章
| 16 | GET `/admin/posts?keyword=&status=&categoryId=&page=&pageSize=` | 分页 AdminPostItem，最新在前 |
| 17 | POST `/admin/posts` | `{title,slug?,summary?,content,cover?,categoryId,tags:[名称字符串],status,isTop}` → `{post:AdminPostItem}`；slug 空/重复则自动生成（post-{id} 或追加 -id）；tags 按 name upsert |
| 18 | GET `/admin/posts/:id` | `{post:AdminPostItem}` |
| 19 | PUT `/admin/posts/:id` | 同 17，全量更新 |
| 20 | PUT `/admin/posts/:id/status` | `{status}` → data:null |
| 21 | DELETE `/admin/posts/:id` | data:null；同时删除其 post_tags 与评论 |

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
| 30 | GET `/admin/comments?status=&postId=&page=&pageSize=` | 分页 CommentAdmin，最新在前 |
| 31 | PUT `/admin/comments/:id/status` | `{status:1\|2}` → data:null |
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

## 联调冒烟清单（QA 用）

1. `go run .` 启动后 `curl :8080/api/v1/site` → code 0
2. 登录拿 token → 创建文章（含标签）→ 公开列表可见 → 详情 viewCount+1
3. 游客评论 → admin 待审 1 条 → 通过 → 公开可见且为树
4. 上传图片 → 返回 url 可访问
5. `curl :8080/rss` → XML
