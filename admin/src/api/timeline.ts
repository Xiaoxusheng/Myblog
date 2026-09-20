import http from './http'
import type { TimelineEventAdmin } from '@/types/api'

export interface TimelinePayload {
  title: string
  content?: string
  eventDate: string // RFC3339
  image?: string
  postId?: number
  projectName?: string
  projectUrl?: string
  visible: boolean
  sort?: number
}

export function getTimeline(params: {
  page?: number
  pageSize?: number
}): Promise<{ list: TimelineEventAdmin[]; total: number }> {
  return http.get('/admin/timeline', { params })
}

export function createTimeline(payload: TimelinePayload): Promise<{ item: TimelineEventAdmin }> {
  return http.post('/admin/timeline', payload)
}

export function updateTimeline(
  id: number,
  payload: TimelinePayload,
): Promise<{ item: TimelineEventAdmin }> {
  return http.put(`/admin/timeline/${id}`, payload)
}

export function deleteTimeline(id: number): Promise<null> {
  return http.delete(`/admin/timeline/${id}`)
}
