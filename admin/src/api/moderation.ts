import http from './http'
import type { BlacklistItem, PageResult } from '@/types/api'

export function getBlacklist(params: {
  type?: 'ip' | 'email' | 'keyword' | ''
  page?: number
  pageSize?: number
}): Promise<PageResult<BlacklistItem>> {
  return http.get('/admin/comment-blacklist', { params })
}

export function createBlacklist(payload: { type: 'ip' | 'email' | 'keyword'; value: string }): Promise<{ item: BlacklistItem }> {
  return http.post('/admin/comment-blacklist', payload)
}

export function deleteBlacklist(id: number): Promise<null> {
  return http.delete(`/admin/comment-blacklist/${id}`)
}
