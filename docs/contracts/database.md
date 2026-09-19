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
