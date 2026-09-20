import http from './http'
import type {
  AnalyticsBrowserItem,
  AnalyticsData,
  AnalyticsDeviceItem,
  AnalyticsOsItem,
  AnalyticsRange,
  AnalyticsSourceItem,
  AnalyticsTopPost,
  AnalyticsTrendPoint,
  PostAnalyticsData,
  PostAnalyticsRange,
} from '@/types/api'

/**
 * 响应结构兜底：在接口边界把服务端返回收敛成契约形状。
 *
 * 动机：访问分析是数据密集型看板，字段缺失/结构异常不应让整页渲染失败
 * （历史上一次搜索统计接口返回了别的形状，就把整页 render 打挂了）。
 * 这里缺失即 0 / 空数组，且**保留「无 prevTotals」的语义**——
 * 环比基线缺失时前端应隐藏对比区，而不是拿 0 冒充一个 -100% 的变化。
 */
function toCount(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

function toList<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : []
}

function toText(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function normalizeAnalytics(
  raw: Partial<AnalyticsData> | null | undefined,
  fallbackRange: AnalyticsRange,
): AnalyticsData {
  const totals = (raw?.totals ?? {}) as { pv?: unknown; uv?: unknown }
  const prev = raw?.prevTotals as { pv?: unknown; uv?: unknown } | undefined

  return {
    range: (raw?.range ?? fallbackRange) as AnalyticsRange,
    totals: { pv: toCount(totals.pv), uv: toCount(totals.uv) },
    ...(prev ? { prevTotals: { pv: toCount(prev.pv), uv: toCount(prev.uv) } } : {}),
    trend: toList<AnalyticsTrendPoint>(raw?.trend).map((point) => ({
      date: toText(point?.date),
      pv: toCount(point?.pv),
      uv: toCount(point?.uv),
    })),
    topPosts: toList<AnalyticsTopPost>(raw?.topPosts).map((item) => ({
      postId: toCount(item?.postId),
      title: toText(item?.title),
      pv: toCount(item?.pv),
      uv: toCount(item?.uv),
      likeCount: toCount(item?.likeCount),
      commentCount: toCount(item?.commentCount),
    })),
    sources: toList<AnalyticsSourceItem>(raw?.sources),
    devices: toList<AnalyticsDeviceItem>(raw?.devices),
    browsers: toList<AnalyticsBrowserItem>(raw?.browsers),
    oses: toList<AnalyticsOsItem>(raw?.oses),
  }
}

/** 搜索统计条目（契约 #85） */
export interface SearchStatItem {
  keyword: string
  count: number
  noResultCount: number
}

function normalizeSearchStats(raw: unknown): SearchStatItem[] {
  const payload = raw as { list?: unknown } | null | undefined
  return toList<{ keyword?: unknown; count?: unknown; noResultCount?: unknown }>(payload?.list)
    .map((item) => ({
      keyword: toText(item?.keyword),
      count: toCount(item?.count),
      noResultCount: toCount(item?.noResultCount),
    }))
    // 关键词为空的条目没有展示意义
    .filter((item) => item.keyword.length > 0)
}

/**
 * GET /admin/analytics:站点访问分析（默认 7d）
 * @param signal 可选中止信号——时间范围快速切换时取消上一个在途请求
 */
export async function getAnalytics(
  range: AnalyticsRange,
  signal?: AbortSignal,
): Promise<AnalyticsData> {
  const raw: Partial<AnalyticsData> | null = await http.get('/admin/analytics', {
    params: { range },
    signal,
  })
  return normalizeAnalytics(raw, range)
}

/** GET /admin/analytics/posts/:id:单篇文章访问分析 */
export function getPostAnalytics(id: number, range: PostAnalyticsRange): Promise<PostAnalyticsData> {
  return http.get(`/admin/analytics/posts/${id}`, { params: { range } })
}

/** 站内搜索统计（契约 #85）；页面次级面板，失败由调用方降级展示、不弹全局 toast */
export async function getSearchStats(
  params: { range?: '7d' | '30d' | '90d' },
  signal?: AbortSignal,
): Promise<{ range: string; list: SearchStatItem[] }> {
  const raw: { range?: unknown } | null = await http.get('/admin/analytics/searches', {
    params,
    signal,
    silent: true,
  })
  return { range: toText(raw?.range), list: normalizeSearchStats(raw) }
}
