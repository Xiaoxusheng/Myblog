import dayjs from 'dayjs'
import type { AnalyticsRange, AnalyticsTrendPoint } from '@/types/api'

/**
 * 访问分析页的领域计算与文案映射。
 * 只做「真实数据的派生」，不产生任何占位/模拟数值——算不出来就返回 null 由 UI 降级展示。
 */

/** 环比结果 */
export interface Delta {
  /**
   * 变化率；null 表示「基期为 0 但有增长」——方向明确、百分比无法表达，
   * UI 应显示「新增」而非编造一个 ∞% 或 100%。
   */
  ratio: number | null
  direction: 'up' | 'down' | 'flat'
}

/**
 * 计算环比。previous 缺失（旧后端未返回 prevTotals）时返回 undefined，
 * 调用方据此隐藏对比区，而不是拿当前值假装对比。
 */
export function computeDelta(current: number, previous?: number | null): Delta | undefined {
  if (previous == null || !Number.isFinite(previous)) return undefined
  if (previous === 0) {
    if (current === 0) return { ratio: 0, direction: 'flat' }
    return { ratio: null, direction: 'up' }
  }
  const ratio = (current - previous) / previous
  if (!Number.isFinite(ratio)) return undefined
  return { ratio, direction: ratio > 0 ? 'up' : ratio < 0 ? 'down' : 'flat' }
}

/** 时间范围元信息：标签 / 环比基线文案 / 趋势天数 / 次级面板可用性 */
export interface RangeMeta {
  value: AnalyticsRange
  label: string
  /** 环比基线说明，直接展示给用户，让「和什么比」始终明确 */
  compareLabel: string
  days: number
  /** 站内搜索统计后端最小粒度为 7d */
  supportsSearch: boolean
}

export const RANGE_META: Record<AnalyticsRange, RangeMeta> = {
  today: {
    value: 'today',
    label: '今日',
    compareLabel: '较昨日同期',
    days: 1,
    supportsSearch: false,
  },
  '7d': { value: '7d', label: '近 7 天', compareLabel: '较前 7 天', days: 7, supportsSearch: true },
  '30d': {
    value: '30d',
    label: '近 30 天',
    compareLabel: '较前 30 天',
    days: 30,
    supportsSearch: true,
  },
  '90d': {
    value: '90d',
    label: '近 90 天',
    compareLabel: '较前 90 天',
    days: 90,
    supportsSearch: true,
  },
}

/** 时间范围选项（Segmented 用，顺序即展示顺序） */
export const RANGE_OPTIONS: { label: string; value: AnalyticsRange }[] = [
  RANGE_META.today,
  RANGE_META['7d'],
  RANGE_META['30d'],
  RANGE_META['90d'],
].map((meta) => ({ label: meta.label, value: meta.value }))

/** 趋势总览（全部来自真实 trend 数组的派生值） */
export interface TrendSummary {
  totalPv: number
  totalUv: number
  /** 日均 PV，保留一位小数 */
  averagePv: number
  /** 峰值日；trend 为空时为 null */
  peak: { date: string; pv: number } | null
  /** 是否存在任意非零数据（全 0 视为「暂无访问」空态） */
  hasData: boolean
}

export function summarizeTrend(trend: AnalyticsTrendPoint[]): TrendSummary {
  let totalPv = 0
  let totalUv = 0
  let peak: TrendSummary['peak'] = null
  for (const point of trend) {
    totalPv += point.pv
    totalUv += point.uv
    if (!peak || point.pv > peak.pv) peak = { date: point.date, pv: point.pv }
  }
  return {
    totalPv,
    totalUv,
    averagePv: trend.length > 0 ? Number((totalPv / trend.length).toFixed(1)) : 0,
    peak,
    hasData: totalPv > 0 || totalUv > 0,
  }
}

/** 坐标轴刻度：2026-09-14 → 09-14 */
export function axisDate(date: string): string {
  return date.length >= 10 ? date.slice(5) : date
}

/** 展示用日期：2026-09-14 → 9月14日 */
export function displayDate(date: string): string {
  const parsed = dayjs(date)
  return parsed.isValid() ? parsed.format('M月D日') : date
}

/** 范围的人话描述，用于空态与副标题：7d → 近 7 天 */
export function rangeDescription(range: AnalyticsRange): string {
  return RANGE_META[range].label
}

/** 本地「最后更新」时间戳文案 */
export function formatUpdatedAt(value: Date | null): string {
  if (!value) return ''
  return dayjs(value).format('HH:mm:ss')
}
