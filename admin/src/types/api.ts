/**
 * MyBlog API 契约类型定义
 * 唯一真相来源：docs/contracts/api.md（字段一律 camelCase）
 */

/** 统一分页响应结构 */
export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

/** post.status：0 草稿 1 已发布 2 隐藏 3 定时发布 */
export type PostStatus = 0 | 1 | 2 | 3
/** comment.status：0 待审核 1 已通过 2 已拒绝 */
export type CommentStatus = 0 | 1 | 2
/** page.status：0 草稿 1 已发布 */
export type PageStatus = 0 | 1

export interface Category {
  id: number
  name: string
  slug: string
  description?: string
  postCount?: number
}

export interface Tag {
  id: number
  name: string
  slug: string
  postCount?: number
}

export interface CategoryRef {
  id: number
  name: string
  slug: string
}

export interface TagRef {
  id: number
  name: string
  slug: string
}

/** 文章列表项（不含 content） */
export interface PostSummary {
  id: number
  title: string
  slug: string
  summary: string
  cover: string
  viewCount: number
  likeCount: number
  status: PostStatus
  isTop: boolean
  createdAt: string
  publishedAt: string | null
  category: CategoryRef | null
  tags: TagRef[]
}

/** 文章详情（公开接口） */
export interface PostDetail extends PostSummary {
  content: string
  updatedAt: string
}

/** 管理端文章对象：PostSummary + content + categoryId + tagNames + commentCount + publishAt */
export interface AdminPostItem extends PostSummary {
  content: string
  categoryId: number
  tagNames: string[]
  commentCount: number
  publishAt: string | null
  updatedAt?: string
}

/** 创建/更新文章入参（tags 为名称字符串数组，按 name upsert） */
export interface PostPayload {
  title: string
  slug?: string
  summary?: string
  content: string
  cover?: string
  categoryId: number
  tags: string[]
  status: PostStatus
  isTop: boolean
  /** 定时发布计划时间（RFC3339）；status=3 必填，其余状态忽略 */
  publishAt?: string | null
  /** 自动保存标记：服务端据此做版本生成防抖 */
  auto?: boolean
  /** 所属专题 id；0/缺省=移出专题 */
  seriesId?: number
  /** 专题内序号；0=自动排到末尾（已成员则保持原序号） */
  seriesSort?: number
}

/** 文章版本列表项（不含 content） */
export interface PostRevisionItem {
  id: number
  postId: number
  version: number
  /** 变更说明：首次保存 / 修改标题、正文 / 恢复前快照 / 恢复自 vN */
  remark: string
  createdAt: string
}

/** 文章版本详情：该版本保存时的文章全量快照 */
export interface PostRevisionDetail extends PostRevisionItem {
  title: string
  slug: string
  summary: string
  cover: string
  content: string
  categoryId: number
  isTop: boolean
  status: PostStatus
}

/** 管理端评论 */
export interface CommentAdmin {
  id: number
  postId: number
  postTitle: string
  parentId: number
  nickname: string
  email: string
  website: string
  content: string
  status: CommentStatus
  ip: string
  isAdmin: boolean
  createdAt: string
}

/** 友链 */
export interface Link {
  id: number
  name: string
  url: string
  logo: string
  description: string
  visible: boolean
  sort: number
  createdAt: string
}

/** 自定义页面 */
export interface PageItem {
  id: number
  title: string
  slug: string
  content: string
  status: PageStatus
  createdAt: string
  updatedAt: string
}

export interface PagePayload {
  title: string
  slug: string
  content: string
  status: PageStatus
}

/** 上传对象，url 形如 /uploads/202609/xxx.png */
export interface UploadItem {
  id: number
  url: string
  filename: string
  size: number
  mime: string
  createdAt: string
}

export interface User {
  id: number
  username: string
  nickname: string
  email: string
  avatar: string
}

export interface LoginResult {
  token: string
  user: User
}

/** 系统设置（结构化对象，不是 map） */
export interface Settings {
  siteName: string
  siteDescription: string
  siteKeywords: string
  siteUrl: string
  logo: string
  notice: string
  icp: string
  footerText: string
  commentEnabled: boolean
  postPageSize: number
  /** 文章 slug 变更时自动创建旧→新 301 重定向 */
  autoRedirectOnSlugChange: boolean
}

/** 专题 */
export interface Series {
  id: number
  name: string
  slug: string
  description: string
  cover: string
  visible: boolean
  sort: number
  postCount?: number
  createdAt: string
  updatedAt: string
}

/** 专题内文章项（管理端排序用） */
export interface SeriesPostItem {
  id: number
  title: string
  slug: string
  status: PostStatus
  sort: number
}

/** URL 重定向 */
export interface RedirectItem {
  id: number
  source: string
  target: string
  type: 301 | 302
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface ProfilePayload {
  nickname: string
  email: string
  avatar: string
}

/** 仪表盘近 7 天趋势点 */
export interface TrendPoint {
  date: string
  posts: number
  comments: number
}

export interface RecentComment {
  id: number
  postTitle: string
  nickname: string
  content: string
  status: CommentStatus
  createdAt: string
}

/** 计划发布中的文章（仪表盘用） */
export interface ScheduledPost {
  id: number
  title: string
  publishAt: string
}

export interface Stats {
  postCount: number
  draftCount: number
  scheduledCount: number
  commentCount: number
  pendingCommentCount: number
  viewCount: number
  likeCount: number
  linkCount: number
  trend: TrendPoint[]
  recentComments: RecentComment[]
  scheduledPosts: ScheduledPost[]
}
