import http from './http'
import type { AuditLogItem, PageResult } from '@/types/api'

export function getAuditLogs(params: {
  action?: string
  page?: number
  pageSize?: number
}): Promise<PageResult<AuditLogItem>> {
  return http.get('/admin/audit-logs', { params })
}
