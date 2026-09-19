# MyBlog 数据库契约

GORM AutoMigrate 建表；表名复数小写下划线（GORM 默认）。所有模型含 `CreatedAt/UpdatedAt`（time.Time）。ID 均 uint 自增。

## users
| 列 | 类型 | 说明 |
|---|---|---|
| id | uint PK | |
| username | string uniqueIndex size:64 | |
| password | string size:128 | bcrypt |
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

写入时做环检测（沿 target 链回溯回到自身或超 10 层 → 拒绝）；文章 slug 变更且 `autoRedirectOnSlugChange` 开启时自动 upsert 旧→新 301。

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

## pages
id, title(size:200), slug(uniqueIndex size:200), content text, status tinyint (0/1)

## settings
key string PK(size:64), value text。存储 Settings 结构化对象的各字段（bool/number 转字符串存取）

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
| read | bool index | 默认 false |

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

## Seed（首次启动写入，幂等）

- 用户：`admin / admin123`（bcrypt）
- 分类：`未分类`；标签：`Go`、`Gin`、`Vue3`、`随笔`
- 文章：3 篇已发布示例（Markdown 含二级标题与 Go/JS 代码块，便于验证高亮/TOC）、1 篇草稿
- 评论：2 条已通过（1 条管理员回复）
- 友链：3 条；页面：`关于`（slug=about，已发布）
- Settings 默认值：siteName=My Blog、commentEnabled=true、postPageSize=10、其余空
