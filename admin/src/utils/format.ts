import dayjs from 'dayjs'

/** RFC3339 → 本地展示时间，空值返回 '-' */
export function formatTime(value?: string | null, withTime = true): string {
  if (!value) return '-'
  const d = dayjs(value)
  if (!d.isValid()) return '-'
  return withTime ? d.format('YYYY-MM-DD HH:mm') : d.format('YYYY-MM-DD')
}

/** 字节数 → 可读大小 */
export function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index += 1
  }
  return `${value >= 100 ? Math.round(value) : value.toFixed(1)} ${units[index]}`
}

/**
 * 计数统一格式（表格/卡片共用，全站一套口径）：
 * - < 10000  → 千分位原值：1,280
 * - < 1e6    → 紧凑 k：12.8k（去掉多余的 .0）
 * - 其余     → 紧凑 M：1.23M
 * 需要精确值时由调用方把原始数字放进 title 悬浮提示。
 */
export function formatCount(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  const abs = Math.abs(value)
  if (abs < 10000) return value.toLocaleString('zh-CN')
  if (abs < 1_000_000) return `${trimZero(value / 1000, 1)}k`
  return `${trimZero(value / 1_000_000, 2)}M`
}

/** 千分位原值（不做紧凑化），用于需要精确读数的位置 */
export function formatNumber(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  return value.toLocaleString('zh-CN')
}

/** 保留固定小数位并去掉尾随 0：12.80 → 12.8，12.00 → 12 */
function trimZero(value: number, digits: number): string {
  return value.toFixed(digits).replace(/\.?0+$/, '')
}

/** 保留一位小数的百分比，带正负号：+12.4% / -8.0% / 0% */
export function formatPercent(ratio: number): string {
  if (!Number.isFinite(ratio)) return '-'
  const percent = ratio * 100
  const rounded = Math.abs(percent) < 0.05 ? 0 : percent
  const sign = rounded > 0 ? '+' : ''
  return `${sign}${rounded.toFixed(1)}%`
}

/** 复制文本到剪贴板，失败时降级 execCommand */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    try {
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      const ok = document.execCommand('copy')
      document.body.removeChild(textarea)
      return ok
    } catch {
      return false
    }
  }
}
