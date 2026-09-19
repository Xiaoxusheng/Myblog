import http from './http'
import type { PageResult, RedirectItem } from '@/types/api'

export interface RedirectPayload {
  source: string
  target: string
  type: 301 | 302
  enabled: boolean
}

export function getRedirects(params: {
  page?: number
  pageSize?: number
}): Promise<PageResult<RedirectItem>> {
  return http.get('/admin/redirects', { params })
}

export function createRedirect(payload: RedirectPayload): Promise<{ redirect: RedirectItem }> {
  return http.post('/admin/redirects', payload)
}

export function updateRedirect(
  id: number,
  payload: RedirectPayload,
): Promise<{ redirect: RedirectItem }> {
  return http.put(`/admin/redirects/${id}`, payload)
}

export function deleteRedirect(id: number): Promise<null> {
  return http.delete(`/admin/redirects/${id}`)
}
