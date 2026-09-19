import http from './http'
import type { AdminPostItem, PageResult, PostPayload, PostStatus } from '@/types/api'

export interface PostListParams {
  keyword?: string
  status?: PostStatus | ''
  categoryId?: number | ''
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
