import http from './http'
import type { HealthInfo } from '@/types/api'

export function getHealth(): Promise<HealthInfo> {
  return http.get('/admin/health')
}
