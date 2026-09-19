import http from './http'
import type { Link } from '@/types/api'

export interface LinkPayload {
  name: string
  url: string
  logo?: string
  description?: string
  visible: boolean
  sort?: number
}

export function getLinks(params: { page?: number; pageSize?: number }): Promise<{ list: Link[]; total: number }> {
  return http.get('/admin/links', { params })
}

export function createLink(payload: LinkPayload): Promise<Link> {
  return http.post('/admin/links', payload)
}

export function updateLink(id: number, payload: LinkPayload): Promise<Link> {
  return http.put(`/admin/links/${id}`, payload)
}

export function deleteLink(id: number): Promise<null> {
  return http.delete(`/admin/links/${id}`)
}
