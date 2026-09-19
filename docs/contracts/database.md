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
| status | tinyint index | 0草稿 1已发布 2隐藏 |
| is_top | bool | |
| published_at | *time.Time | 首次置为已发布时写入 |

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

## Seed（首次启动写入，幂等）

- 用户：`admin / admin123`（bcrypt）
- 分类：`未分类`；标签：`Go`、`Gin`、`Vue3`、`随笔`
- 文章：3 篇已发布示例（Markdown 含二级标题与 Go/JS 代码块，便于验证高亮/TOC）、1 篇草稿
- 评论：2 条已通过（1 条管理员回复）
- 友链：3 条；页面：`关于`（slug=about，已发布）
- Settings 默认值：siteName=My Blog、commentEnabled=true、postPageSize=10、其余空
