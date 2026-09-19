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
