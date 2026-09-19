import type { CommentStatus, PostStatus } from '@/types/api'

export interface StatusMeta {
  text: string
  color: string
}

/** post.status → Tag 展示 */
export const POST_STATUS_MAP: Record<PostStatus, StatusMeta> = {
  0: { text: '草稿', color: 'default' },
  1: { text: '已发布', color: 'success' },
  2: { text: '隐藏', color: 'warning' },
}

/** comment.status → Tag 展示 */
export const COMMENT_STATUS_MAP: Record<CommentStatus, StatusMeta> = {
  0: { text: '待审核', color: 'processing' },
  1: { text: '已通过', color: 'success' },
  2: { text: '已拒绝', color: 'error' },
}

export const TOKEN_KEY = 'blog_admin_token'
