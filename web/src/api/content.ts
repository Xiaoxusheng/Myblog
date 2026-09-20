import { api } from './http'
import type { ArchiveYear, ChangelogItem, CustomPage, LinkItem, TimelineEventItem } from '@/types'

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

/** GET /timeline:可见时间线节点（默认时间倒序） */
export function fetchTimeline(signal?: AbortSignal): Promise<TimelineEventItem[]> {
  return api
    .get<{ list: TimelineEventItem[] }>('/timeline', { signal })
    .then((d) => d?.list ?? [])
}

/** GET /changelog:已发布版本记录（默认版本倒序） */
export function fetchChangelog(signal?: AbortSignal): Promise<ChangelogItem[]> {
  return api
    .get<{ list: ChangelogItem[] }>('/changelog', { signal })
    .then((d) => d?.list ?? [])
}
