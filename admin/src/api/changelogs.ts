import http from './http'
import type { ChangelogItem } from '@/types/api'

export interface ChangelogPayload {
  version: string
  title?: string
  content?: string
  releasedAt: string // RFC3339
  status: number
  sort?: number
}

export function getChangelogs(params: {
  page?: number
  pageSize?: number
  status?: number
}): Promise<{ list: ChangelogItem[]; total: number }> {
  return http.get('/admin/changelogs', { params })
}

export function createChangelog(payload: ChangelogPayload): Promise<{ item: ChangelogItem }> {
  return http.post('/admin/changelogs', payload)
}

export function updateChangelog(
  id: number,
  payload: ChangelogPayload,
): Promise<{ item: ChangelogItem }> {
  return http.put(`/admin/changelogs/${id}`, payload)
}

export function deleteChangelog(id: number, force = false): Promise<null> {
  return http.delete(`/admin/changelogs/${id}`, { params: force ? { force: 'true' } : {} })
}
