/** 搜索关键词高亮：HTML 转义后把命中片段包上 <mark>（关键词来自 URL，必须先转义） */

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export function highlightText(text: string, keyword: string): string {
  const safeText = escapeHtml(text ?? '')
  const kw = (keyword ?? '').trim()
  if (!kw) return safeText
  // 转义正则元字符后全局匹配（大小写不敏感）
  const pattern = kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  try {
    return safeText.replace(new RegExp(pattern, 'gi'), (m) => `<mark>${m}</mark>`)
  } catch {
    return safeText
  }
}
