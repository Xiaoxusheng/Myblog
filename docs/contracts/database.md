# MyBlog 数据库契约

GORM AutoMigrate 建表；表名复数小写下划线（GORM 默认）。所有模型含 `CreatedAt/UpdatedAt`（time.Time）。ID 均 uint 自增。

## users
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| username | string uniqueIndex size:64 | |
| password | string size:128 | bcrypt |
| token_version | int not null default:0 | 改密 +1；JWT claims `ver` 与之不符即 401（改密吊销旧 token） |
| nickname | string size:64 | |
| email | string size:128 | |
| avatar | string size:512 | 可空 |

## posts
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| title | string size:200 | 必填 |
| slug | string uniqueIndex size:200 | 可为空串，后端保证唯一 |
| summary | string size:1000 | |
| content | text | Markdown 原文 |
| cover | string size:512 | |
| category_id | uint index | 0=未分类 |
| view_count / like_count | int | 默认 0 |
| status | tinyint index | 0草稿 1已发布 2隐藏 3定时发布 |
| is_top | bool | |
| published_at | *time.Time | 首次置为已发布时写入；定时发布自动到点时写入计划时间 |
| seo_title | string size:200 | SEO 标题（空=用文章标题） |
| seo_description | string size:300 | SEO 描述（空=用摘要） |
| canonical | string size:512 | Canonical URL（空=默认规则） |
| og_image | string size:512 | OG 图（空=用封面） |
| publish_at | *time.Time index | 仅 status=3 有值：计划发布时间；调度器按 `status=3 AND publish_at<=now` 扫描 |
| series_id | uint index | 0=不属于专题（一文至多一专题） |
| series_sort | int | 专题内序号，升序，同序按 id |

## post_revisions（文章版本历史，随文章删除级联删除）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| post_id | uint index | |
| version | int | 文章内自 1 递增；唯一约束 (post_id, version) |
| title | string size:200 | 该版本保存时的标题 |
| slug | string size:200 | |
| summary | string size:1000 | |
| cover | string size:512 | |
| content | text | Markdown 原文快照 |
| category_id | uint | |
| is_top | bool | |
| status | tinyint | 保存时的文章状态 |
| remark | string size:200 | 变更说明（首次保存 / 修改标题、正文 / 恢复前快照 / 恢复自 vN） |

生成规则：仅当文章快照字段（title/slug/summary/cover/content/category_id/is_top/status）与最新版本不同才插入；`auto` 保存距最新版本 <120s 不插入（防抖）。

## series（专题）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| name | string uniqueIndex size:100 | |
| slug | string uniqueIndex size:100 | 空则由名称派生，冲突追加 -id |
| description | string size:500 | |
| cover | string size:512 | |
| visible | bool | 默认 true |
| sort | int | 专题展示顺序，升序 |

成员关系直接存 `posts.series_id` / `posts.series_sort`（一文一专题，免 join 表）；
删除专题时将成员文章 series_id 置 0。

## redirects（URL 重定向）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| source | string uniqueIndex size:512 | 旧站内路径（/post/xxx），唯一 |
| target | string size:512 | 新站内路径 |
| type | int | 301 永久 / 302 临时 |
| enabled | bool | 默认 true |

写入时做环检测（沿 target 链回溯回到自身或超 10 层 → 拒绝）；文章 slug 变更且 `autoRedirectOnSlugChange` 开启时自动 upsert `/post/旧slug → /post/新slug` 301；页面 slug 变更同理，upsert `/page/旧slug → /page/新slug` 301（复用同一张表与开关，不新增第二套重定向系统）。

## categories
id, name(uniqueIndex size:64), slug(size:64), description(size:500)

## tags
id, name(uniqueIndex size:64), slug(size:64)

## post_tags
post_id uint, tag_id uint，复合主键 (post_id, tag_id)，双向 index

## comments
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| post_id | uint index | |
| parent_id | uint index | 0=顶级 |
| nickname | string size:64 | |
| email | string size:128 | 不对公开展示 |
| website | string size:256 | |
| content | text | ≤1000 字校验 |
| status | tinyint index | 0待审 1通过 2拒绝 |
| ip | string size:64 | 仅管理端可见 |
| is_admin | bool | 管理员回复 |

## links
id, name(size:100), url(size:500), logo(size:512), description(size:500), visible bool, sort int

## pages（自定义页面；契约 #38-42 / #101-109）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| title | string size:200 | 必填 |
| slug | string uniqueIndex size:200 | 必填，后端保证唯一 |
| content | text | Markdown 原文 |
| status | tinyint index | 0草稿 1已发布 2隐藏 3定时发布（与 posts.status 同一枚举；3 到点由调度器自动置 1） |
| page_type | string size:32 | 页面类型 default/about/links/contact；当前仅 default 有 UI，其余为后续模板预留，不为其新增字段 |
| published_at | *time.Time | 首次置为已发布时写入；定时发布到点时写入计划时间 |
| publish_at | *time.Time index | 仅 status=3 有值：计划发布时间；调度器按 `status=3 AND publish_at<=now` 扫描 |
| seo_title | string size:200 | SEO 标题（空=用页面标题） |
| seo_description | string size:300 | SEO 描述（空=用正文摘要） |
| canonical | string size:512 | Canonical URL（空=默认规则） |
| og_image | string size:512 | OG 图（空=无） |

## page_revisions（页面版本历史，随页面删除级联删除；契约 #101-103）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| page_id | uint index | |
| version | int | 页面内自 1 递增；唯一约束 (page_id, version) |
| title | string size:200 | 该版本保存时的标题 |
| slug | string size:200 | |
| content | text | Markdown 原文快照 |
| status | tinyint | 保存时的页面状态 |
| remark | string size:200 | 变更说明（首次保存 / 修改标题、正文 / 恢复前快照 / 恢复自 vN） |

生成规则与 post_revisions 一致：仅当快照字段（title/slug/content/status）与最新版本不同才插入；`auto` 保存距最新版本 <120s 不插入（防抖）；无变化不插入。
说明：`post_revisions` 与 `page_revisions` 为两张独立表（分别由 `post_id`/`page_id` 关联），不合并为多态表——保持与既有 `PostRevision` 一致的实现范式，便于各自独立演进与索引。

## settings
key string PK(size:64), value text。存储 Settings 结构化对象的各字段（bool/number 转字符串存取）

**`key` 是 MySQL 8 保留字**：模型必须写 `gorm:"column:key"`，手写 SQL 必须用 `` `key` `` 包裹。
错误被读取方（`autoRedirectOnSlugChangeEnabled`）吞掉回退默认值 → 表现为「设置改了不生效」且日志无痕，
属最隐蔽的一类；SQLite 不保留该词，本地测不出来。

## uploads
id, filename(size:255), path(size:512), url(size:512), size int64, mime(size:100)

## search_logs（站内搜索统计）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| keyword | string index size:200 | 搜索词（归一化：trim，≤200） |
| result_count | int | 该次搜索命中的文章数（0=无结果） |
| created_at | time index | |

## comment_blacklist（评论防护黑名单）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| type | string size:16 | ip / email / keyword |
| value | string size:200 | IP、邮箱或关键词；命中 → 评论直接 status=3 |
| created_at | time | 唯一约束 (type, value) |

## notifications（后台通知中心，轻量轮询）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| type | string size:32 | comment_pending / comment_spam / post_published / backup |
| title | string size:200 | |
| content | string size:500 | |
| link | string size:500 | 管理端跳转路径（如 /comments?status=0） |
| read | bool index | 默认 false。**列名是 MySQL 8 保留字**：模型必须写 `gorm:"column:read"`，手写 SQL 必须用 `` `read` `` 包裹，否则 MySQL 报 Error 1064（SQLite 不保留该词，本地测不出来） |

## audit_logs（操作日志；禁止记录密码/JWT/完整请求体）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| action | string size:50 index | 如 post.create / post.update / comment.batch / backup.create |
| resource_type | string size:50 | post / comment / upload / setting / backup / system |
| resource_id | string size:50 | |
| description | string size:500 | 摘要说明（截断，不含敏感内容） |
| ip_hash | string size:64 | 管理员 IP 的 SHA-256 哈希 |
| created_at | time index | |

备份文件存于 `BLOG_BACKUP_DIR`（默认 backups/，不对外提供静态访问）；
运行中服务的数据库恢复不在 API 内执行（避免热替换损坏），按部署文档手动操作。

## page_views（访问统计；隐私：不存明文 IP，仅 SHA-256 哈希）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| path | string size:512 | 被访问路径（/post/xxx、/about 等） |
| post_id | uint index | 0=非文章页 |
| visitor_hash | string index size:64 | SHA-256(ip+UA+密钥)，UV 去重键 |
| ip_hash | string size:64 | SHA-256(ip+密钥)，滥用审计用 |
| user_agent | string size:512 | 原始 UA（解析后存 device/browser/os，原始值留诊断） |
| referer | string size:512 | 原始 referer |
| referer_source | string size:16 | direct/search/github/social/other |
| device_type | string size:16 | desktop/mobile/tablet |
| browser / os | string size:32 | UA 粗解析 |
| created_at | time index | 事件时间（不可变行，无 updated_at） |

## banned_ips（IP 封禁；自动/手动封禁均落库，重启后由防护中间件重新加载）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| ip | string size:64 uniqueIndex | 封禁的 IP（明文：需精确等值匹配，同 comment_blacklist 的 ip 值） |
| reason | string size:200 | 封禁原因（WAF 特征 / 限流超限 / 管理员备注） |
| source | string size:16 | auto（自动）/ manual（手动） |
| created_at | time | |
| expires_at | time *nullable* | nil = 永久；过期的行在防护加载时顺手清理 |

## timeline_events（技术时间线；契约 #91-94 / #99）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| title | varchar(100) | 节点标题（必填） |
| content | text | 节点描述（Markdown 原文，前台安全渲染） |
| event_date | time | 节点日期（必填；默认排序依据之一） |
| image | varchar(512) | 配图 URL（可空） |
| post_id | uint index | 关联文章 id（0=无；弱关联不设外键，文章删除后 post 字段自然为 null） |
| project_name | varchar(100) | 关联项目名称（可空） |
| project_url | varchar(512) | 关联项目链接（可空） |
| visible | bool | 显隐（无 default 标签，规避 GORM 零值坑） |
| sort | int | 排序值（升序在前，默认 0，负值置顶；同 sort 按 event_date 倒序） |
| created_at / updated_at | time | |

## changelogs（版本发布记录；契约 #95-98 / #100）
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| version | varchar(32) uniqueIndex | 版本号（SemVer 兼容，存储统一去 v 前缀） |
| title | varchar(100) | 发布标题（可空） |
| content | text | 更新说明（Markdown 原文） |
| released_at | time | 发布日期 |
| status | int8 | 0 草稿 1 已发布 |
| sort | int | 排序值（升序在前，默认 0；同 sort 按 released_at 倒序，即默认版本倒序） |
| created_at / updated_at | time | |

回滚说明：两表均为全新表、不影响现有表结构；回滚执行 `DROP TABLE timeline_events; DROP TABLE changelogs;` 即可（先确认无业务数据）。

## Seed（首次启动写入，幂等）

- 用户：`admin / admin123`（bcrypt）
- 分类：`未分类`；标签：`Go`、`Gin`、`Vue3`、`随笔`
- 文章：3 篇已发布示例（Markdown 含二级标题与 Go/JS 代码块，便于验证高亮/TOC）、1 篇草稿
- 评论：2 条已通过（1 条管理员回复）
- 友链：3 条；页面：`关于`（slug=about，已发布，page_type=about）
- Settings 默认值：siteName=My Blog、commentEnabled=true、postPageSize=10、autoRedirectOnSlugChange=true、其余空

## 迁移与回滚（页面模块升级，2026-09-20）

增量变更（AutoMigrate 自动完成，均向后兼容）：
- `pages` 新增列：`page_type`、`published_at`、`publish_at`(+index)、`seo_title`、`seo_description`、`canonical`、`og_image`；`status` 由 0/1 扩为 0/1/2/3（列类型不变）
- 新增表 `page_revisions`
- 既有 `pages.status=1` 行语义不变（1 仍为已发布）

回滚：`DROP TABLE page_revisions;` 并删除 `pages` 的新增列即可（`pages` 原有 6 列未改动，数据可直接沿用）。
