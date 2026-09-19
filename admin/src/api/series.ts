import http from './http'
import type { PageResult, Series, SeriesPostItem } from '@/types/api'

export interface SeriesPayload {
  name: string
  slug?: string
  description?: string
  cover?: string
  visible: boolean
  sort?: number
}

export function getSeriesList(params: {
  page?: number
  pageSize?: number
}): Promise<PageResult<Series>> {
  return http.get('/admin/series', { params })
}

export function createSeries(payload: SeriesPayload): Promise<{ series: Series }> {
  return http.post('/admin/series', payload)
}

export function updateSeries(id: number, payload: SeriesPayload): Promise<{ series: Series }> {
  return http.put(`/admin/series/${id}`, payload)
}

export function deleteSeries(id: number): Promise<null> {
  return http.delete(`/admin/series/${id}`)
}

/** 专题内文章（全部状态，按序号排列） */
export function getSeriesPosts(id: number): Promise<{ list: SeriesPostItem[] }> {
  return http.get(`/admin/series/${id}/posts`)
}

/** 批量调整专题内序号 */
export function reorderSeriesPosts(
  id: number,
  items: { postId: number; sort: number }[],
): Promise<null> {
  return http.put(`/admin/series/${id}/posts`, { items })
}
