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

/** post.status：0 草稿 1 已发布 2 隐藏 */
export type PostStatus = 0 | 1 | 2
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

/** 管理端文章对象：PostSummary + content + categoryId + tagNames + commentCount */
export interface AdminPostItem extends PostSummary {
  content: string
  categoryId: number
  tagNames: string[]
  commentCount: number
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

export interface Stats {
  postCount: number
  draftCount: number
  commentCount: number
  pendingCommentCount: number
  viewCount: number
  likeCount: number
  linkCount: number
  trend: TrendPoint[]
  recentComments: RecentComment[]
}
