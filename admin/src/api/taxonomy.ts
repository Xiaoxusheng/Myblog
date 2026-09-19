import http from './http'
import type { Category, PageResult, Tag } from '@/types/api'

// ---------- 分类 ----------

export function getCategories(params: { page?: number; pageSize?: number }): Promise<PageResult<Category>> {
  return http.get('/admin/categories', { params })
}

export function createCategory(payload: { name: string; slug?: string; description?: string }): Promise<Category> {
  return http.post('/admin/categories', payload)
}

export function updateCategory(
  id: number,
  payload: { name: string; slug?: string; description?: string },
): Promise<Category> {
  return http.put(`/admin/categories/${id}`, payload)
}

export function deleteCategory(id: number): Promise<null> {
  return http.delete(`/admin/categories/${id}`)
}

// ---------- 标签 ----------

export function getTags(params: { page?: number; pageSize?: number }): Promise<PageResult<Tag>> {
  return http.get('/admin/tags', { params })
}

export function createTag(payload: { name: string; slug?: string }): Promise<Tag> {
  return http.post('/admin/tags', payload)
}

export function updateTag(id: number, payload: { name: string; slug?: string }): Promise<Tag> {
  return http.put(`/admin/tags/${id}`, payload)
}

export function deleteTag(id: number): Promise<null> {
  return http.delete(`/admin/tags/${id}`)
}
