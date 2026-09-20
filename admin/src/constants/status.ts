import type { CommentStatus, PostStatus, PageStatus } from '@/types/api'

export interface StatusMeta {
  text: string
  color: string
}

/** post.status → Tag 展示 */
export const POST_STATUS_MAP: Record<PostStatus, StatusMeta> = {
  0: { text: '草稿', color: 'default' },
  1: { text: '已发布', color: 'success' },
  2: { text: '隐藏', color: 'warning' },
  3: { text: '定时发布', color: 'processing' },
}

/** page.status → Tag 展示（与 post 同一枚举，展示文案保持一致） */
export const PAGE_STATUS_MAP: Record<PageStatus, StatusMeta> = {
  0: { text: '草稿', color: 'default' },
  1: { text: '已发布', color: 'success' },
  2: { text: '隐藏', color: 'warning' },
  3: { text: '定时发布', color: 'processing' },
}

/** 页面状态下拉筛选项（列表工具栏用） */
export const PAGE_STATUS_OPTIONS = [
  { label: '全部状态', value: undefined },
  { label: '草稿', value: 0 },
  { label: '已发布', value: 1 },
  { label: '隐藏', value: 2 },
  { label: '定时发布', value: 3 },
] as const

/** 页面类型展示文案 */
export const PAGE_TYPE_LABELS: Record<string, string> = {
  default: '普通页面',
  about: '关于页',
  links: '友链说明',
  contact: '联系页',
}

/** comment.status → Tag 展示 */
export const COMMENT_STATUS_MAP: Record<CommentStatus, StatusMeta> = {
  0: { text: '待审核', color: 'processing' },
  1: { text: '已通过', color: 'success' },
  2: { text: '已拒绝', color: 'error' },
  3: { text: '垃圾', color: 'volcano' },
  4: { text: '回收站', color: 'default' },
}

export const TOKEN_KEY = 'blog_admin_token'
