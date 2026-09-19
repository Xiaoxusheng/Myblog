/**
 * 与 docs/contracts/api.md 契约一一对应的类型定义(camelCase)
 */

/** 分类（汇总侧，含可选统计） */
export interface Category {
  id: number
  name: string
  slug: string
  description?: string
  postCount?: number
}

/** 标签（汇总侧，含可选统计） */
export interface Tag {
  id: number
  name: string
  slug: string
  postCount?: number
}

/** 文章内嵌的精简分类/标签 */
export interface PostCategoryRef {
  id: number
  name: string
  slug: string
}

export interface PostTagRef {
  id: number
  name: string
  slug: string
}

/** 文章列表项(PostSummary，不含 content) */
export interface PostSummary {
  id: number
  title: string
  slug: string
  summary: string
  cover: string
  viewCount: number
  likeCount: number
  status: number
  isTop: boolean
  createdAt: string
  publishedAt: string
  category: PostCategoryRef | null
  tags: PostTagRef[]
}

/** 文章详情 = PostSummary + content + updatedAt */
export interface PostDetail extends PostSummary {
  content: string
  updatedAt: string
}

/** 上一篇/下一篇导航项 */
export interface PostNav {
  id: number
  title: string
  slug: string
}

/** GET /posts/:slug 响应 */
export interface PostDetailData {
  post: PostDetail
  prev: PostNav | null
  next: PostNav | null
  related: PostSummary[]
}

/** 公开评论（仅已通过，两级树）;顶级评论 parentId 约定为 0，容忍后端返回 null */
export interface CommentPublic {
  id: number
  parentId: number | null
  nickname: string
  website: string
  content: string
  isAdmin: boolean
  createdAt: string
  children: CommentPublic[]
}

/** POST 评论返回的新评论（待审核，前台不直接展示） */
export interface CommentCreated {
  id: number
  parentId: number | null
  nickname: string
  website: string
  content: string
  createdAt: string
}

/** 评论提交 body */
export interface CommentPayload {
  parentId?: number
  nickname: string
  email: string
  website?: string
  content: string
}

/** 通用分页响应 */
export interface Paged<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

/** GET /posts 查询参数 */
export interface PostListParams {
  keyword?: string
  categoryId?: number
  tagId?: number
  page?: number
  pageSize?: number
  sort?: 'newest' | 'views' | 'likes'
}

/** 站点设置（结构化对象） */
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

/** GET /site 响应 */
export interface SiteData {
  settings: Settings
  categories: Category[]
  tags: Tag[]
}

/** GET /archive 响应项 */
export interface ArchiveItem {
  id: number
  title: string
  slug: string
  createdAt: string
}

export interface ArchiveYear {
  year: number
  items: ArchiveItem[]
}

/** 自定义页面 */
export interface CustomPage {
  id: number
  title: string
  slug: string
  content: string
  status: number
  createdAt: string
  updatedAt: string
}

/** 友链 */
export interface LinkItem {
  id: number
  name: string
  url: string
  logo: string
  description: string
  visible: boolean
  sort: number
  createdAt: string
}
