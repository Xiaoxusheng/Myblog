import http from './http'
import type {
  PageBatchAction,
  PageItem,
  PageListResult,
  PagePayload,
  PageResult,
  PageRevisionDetail,
  PageRevisionItem,
  PageStatusPayload,
} from '@/types/api'

/** 页面列表（契约 #38）：支持 keyword / status 筛选，附带 meta 概览 */
export function getPages(params: {
  keyword?: string
  status?: number
  page?: number
  pageSize?: number
}): Promise<PageListResult> {
  return http.get('/admin/pages', { params })
}

export function getPage(id: number): Promise<{ page: PageItem }> {
  return http.get(`/admin/pages/${id}`)
}

export function createPage(payload: PagePayload): Promise<{ page: PageItem }> {
  return http.post('/admin/pages', payload)
}

export function updatePage(id: number, payload: PagePayload): Promise<{ page: PageItem }> {
  return http.put(`/admin/pages/${id}`, payload)
}

/** 快速修改状态（契约 #44） */
export function updatePageStatus(id: number, payload: PageStatusPayload): Promise<{ page: PageItem }> {
  return http.put(`/admin/pages/${id}/status`, payload)
}

/** 复制页面（契约 #44）：副本强制为草稿 */
export function copyPage(id: number): Promise<{ page: PageItem }> {
  return http.post(`/admin/pages/${id}/copy`)
}

/** 批量操作（契约 #104） */
export function batchPages(payload: {
  action: PageBatchAction
  ids: number[]
}): Promise<{ updated: number }> {
  return http.post('/admin/pages/batch', payload)
}

export function deletePage(id: number): Promise<null> {
  return http.delete(`/admin/pages/${id}`)
}

/** 未发布页面的后台预览数据源（契约 #105） */
export function getPagePreview(id: number): Promise<{ page: PageItem }> {
  return http.get(`/admin/pages/preview/${id}`)
}

/** 版本历史（契约 #101） */
export function getPageRevisions(
  id: number,
  params: { page?: number; pageSize?: number } = {},
): Promise<PageResult<PageRevisionItem>> {
  return http.get(`/admin/pages/${id}/revisions`, { params })
}

/** 版本详情（契约 #102） */
export function getPageRevision(id: number, version: number): Promise<{ revision: PageRevisionDetail }> {
  return http.get(`/admin/pages/${id}/revisions/${version}`)
}

/** 恢复版本（契约 #103）：恢复会生成新版本，不覆盖历史 */
export function restorePageRevision(id: number, version: number): Promise<{ page: PageItem }> {
  return http.post(`/admin/pages/${id}/revisions/${version}/restore`)
}
