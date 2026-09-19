import http from './http'
import type { Stats } from '@/types/api'

export function getStats(): Promise<Stats> {
  return http.get('/admin/stats')
}
