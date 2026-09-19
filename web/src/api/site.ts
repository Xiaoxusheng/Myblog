import { api } from './http'
import type { SiteData } from '@/types'

/** GET /site:站点设置 + 分类 + 标签 */
export function fetchSite(signal?: AbortSignal): Promise<SiteData> {
  return api.get<SiteData>('/site', { signal })
}
