/**
 * 读取 :root 上的 px 值 CSS 变量，供 JS 滚动定位消费
 * （JS 不再硬编码与 CSS 重复的魔法数，docs/08 §7.2）
 * 变量未定义或非数值时返回 fallback
 */
export function readPxVar(varName: string, fallback: number): number {
  const raw = getComputedStyle(document.documentElement).getPropertyValue(varName)
  const parsed = parseFloat(raw)
  return Number.isFinite(parsed) ? parsed : fallback
}
