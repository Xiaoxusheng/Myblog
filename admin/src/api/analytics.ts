import http from './http'
import type {
  AnalyticsData,
  AnalyticsRange,
  PostAnalyticsData,
  PostAnalyticsRange,
} from '@/types/api'

/** GET /admin/analytics:站点访问分析（默认 7d） */
export function getAnalytics(range: AnalyticsRange): Promise<AnalyticsData> {
  return http.get('/admin/analytics', { params: { range } })
}

/** GET /admin/analytics/posts/:id:单篇文章访问分析 */
export function getPostAnalytics(id: number, range: PostAnalyticsRange): Promise<PostAnalyticsData> {
  return http.get(`/admin/analytics/posts/${id}`, { params: { range } })
}
