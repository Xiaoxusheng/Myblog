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

/** status 允许 0 待审（恢复）/ 1 通过 / 2 拒绝 / 3 垃圾 / 4 回收站 */
export function updateCommentStatus(id: number, status: CommentStatus): Promise<null> {
  return http.put(`/admin/comments/${id}/status`, { status })
}

/** 批量操作（契约 #72） */
export function batchComments(
  action: 'approve' | 'reject' | 'spam' | 'delete',
  ids: number[],
): Promise<{ updated: number }> {
  return http.post('/admin/comments/batch', { action, ids })
}

/** 以管理员身份回复（直接通过） */
export function replyComment(id: number, content: string): Promise<{ comment: CommentAdmin }> {
  return http.post(`/admin/comments/${id}/reply`, { content })
}

export function deleteComment(id: number): Promise<null> {
  return http.delete(`/admin/comments/${id}`)
}
