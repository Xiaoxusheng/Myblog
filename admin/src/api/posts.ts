import http from './http'
import type {
  AdminPostItem,
  PageResult,
  PostPayload,
  PostRevisionDetail,
  PostRevisionItem,
  PostStatus,
} from '@/types/api'

export interface PostListParams {
  keyword?: string
  status?: PostStatus | ''
  categoryId?: number | ''
  /** 按标签筛选（服务端 JOIN post_tags，不产生重复计数） */
  tagId?: number | ''
  /** 排序：updatedAt=按最后编辑时间倒序；缺省按创建时间倒序 */
  sort?: 'updatedAt'
  page?: number
  pageSize?: number
}

export function getPosts(params: PostListParams): Promise<PageResult<AdminPostItem>> {
  return http.get('/admin/posts', { params })
}

export function getPost(id: number): Promise<{ post: AdminPostItem }> {
  return http.get(`/admin/posts/${id}`)
}

export function createPost(payload: PostPayload): Promise<{ post: AdminPostItem }> {
  return http.post('/admin/posts', payload)
}

export function updatePost(id: number, payload: PostPayload): Promise<{ post: AdminPostItem }> {
  return http.put(`/admin/posts/${id}`, payload)
}

export function updatePostStatus(id: number, status: PostStatus): Promise<null> {
  return http.put(`/admin/posts/${id}/status`, { status })
}

export function deletePost(id: number): Promise<null> {
  return http.delete(`/admin/posts/${id}`)
}

// ---------- 版本历史 ----------

export function getRevisions(
  postId: number,
  params: { page?: number; pageSize?: number },
): Promise<PageResult<PostRevisionItem>> {
  return http.get(`/admin/posts/${postId}/revisions`, { params })
}

export function getRevision(postId: number, version: number): Promise<{ revision: PostRevisionDetail }> {
  return http.get(`/admin/posts/${postId}/revisions/${version}`)
}

/** 恢复版本：服务端先快照当前内容，再应用目标版本 */
export function restoreRevision(postId: number, version: number): Promise<{ post: AdminPostItem }> {
  return http.post(`/admin/posts/${postId}/revisions/${version}/restore`)
}
