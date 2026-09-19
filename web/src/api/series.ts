import { api } from './http'
import type { Series, SeriesDetailData } from '@/types'

/** GET /series:可见专题列表（sort 升序，postCount 为已发布文章数） */
export function fetchSeriesList(signal?: AbortSignal): Promise<Series[]> {
  return api
    .get<{ list: Series[] }>('/series', { signal })
    .then((d) => d?.list ?? [])
}

/** GET /series/:slug:专题详情 + 专题内已发布文章（按专题内序号排列） */
export function fetchSeriesDetail(slug: string, signal?: AbortSignal): Promise<SeriesDetailData> {
  return api.get<SeriesDetailData>(`/series/${encodeURIComponent(slug)}`, { signal })
}
