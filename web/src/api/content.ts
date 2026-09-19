import { api } from './http'
import type { ArchiveYear, CustomPage, LinkItem } from '@/types'

/** GET /archive:按年倒序的归档 */
export function fetchArchive(signal?: AbortSignal): Promise<ArchiveYear[]> {
  return api.get<ArchiveYear[]>('/archive', { signal })
}

/** GET /pages/:slug:已发布自定义页面 */
export function fetchPage(slug: string, signal?: AbortSignal): Promise<CustomPage> {
  return api
    .get<{ page: CustomPage }>(`/pages/${encodeURIComponent(slug)}`, { signal })
    .then((d) => d.page)
}

/** GET /links:可见友链 */
export function fetchLinks(signal?: AbortSignal): Promise<LinkItem[]> {
  return api
    .get<{ list: LinkItem[] }>('/links', { signal })
    .then((d) => d?.list ?? [])
}
