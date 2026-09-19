import http from './http'
import type { PageItem, PagePayload, PageResult } from '@/types/api'

export function getPages(params: { page?: number; pageSize?: number }): Promise<PageResult<PageItem>> {
  return http.get('/admin/pages', { params })
}

export function getPage(id: number): Promise<{ page: PageItem }> {
  return http.get(`/admin/pages/${id}`)
}

export function createPage(payload: PagePayload): Promise<PageItem> {
  return http.post('/admin/pages', payload)
}

export function updatePage(id: number, payload: PagePayload): Promise<PageItem> {
  return http.put(`/admin/pages/${id}`, payload)
}

export function deletePage(id: number): Promise<null> {
  return http.delete(`/admin/pages/${id}`)
}
