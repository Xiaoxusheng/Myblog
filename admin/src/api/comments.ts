import http from './http'
import type { CommentAdmin, CommentStatus, PageResult } from '@/types/api'

export interface CommentListParams {
  status?: CommentStatus | ''
  postId?: number
  page?: number
  pageSize?: number
}

export function getComments(params: CommentListParams): Promise<PageResult<CommentAdmin>> {
  return http.get('/admin/comments', { params })
}

/** status 仅允许 1 已通过 / 2 已拒绝 */
export function updateCommentStatus(id: number, status: 1 | 2): Promise<null> {
  return http.put(`/admin/comments/${id}/status`, { status })
}

/** 以管理员身份回复（直接通过） */
export function replyComment(id: number, content: string): Promise<{ comment: CommentAdmin }> {
  return http.post(`/admin/comments/${id}/reply`, { content })
}

export function deleteComment(id: number): Promise<null> {
  return http.delete(`/admin/comments/${id}`)
}
