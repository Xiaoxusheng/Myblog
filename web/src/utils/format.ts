/** 日期与数字格式化(RFC3339 输入） */

function toDate(iso?: string | null): Date | null {
  if (!iso) return null
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? null : d
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n)
}

/** 2026-09-19 */
export function formatDate(iso?: string | null): string {
  const d = toDate(iso)
  if (!d) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 2026-09-19 12:00 */
export function formatDateTime(iso?: string | null): string {
  const d = toDate(iso)
  if (!d) return ''
  return `${formatDate(iso)} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** MM-DD（归档时间线用） */
export function formatMonthDay(iso?: string | null): string {
  const d = toDate(iso)
  if (!d) return ''
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 相对时间：刚刚 / n 分钟前 / n 小时前 / n 天前，超过 30 天回退到完整日期 */
export function formatRelative(iso?: string | null): string {
  const d = toDate(iso)
  if (!d) return ''
  const diff = Date.now() - d.getTime()
  if (diff < 0) return formatDate(iso)
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  if (diff < minute) return '刚刚'
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`
  if (diff < 30 * day) return `${Math.floor(diff / day)} 天前`
  return formatDate(iso)
}

/** 千分位：1234 -> 1,234 */
export function formatNumber(n?: number | null): string {
  const v = Number(n ?? 0)
  if (!Number.isFinite(v)) return '0'
  return v.toLocaleString('en-US')
}

/** 简单邮箱格式校验 */
export function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim())
}
