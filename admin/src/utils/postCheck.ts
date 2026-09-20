/**
 * 发布前内容检查的纯函数工具（模块五）。
 * 只做静态分析，不发起请求，便于单测与编辑器复用。
 */

/** Markdown 链接/图片引用扫描结果 */
export interface MarkdownRefScan {
  /** 行内链接数量 [](...) */
  linkCount: number
  /** 图片引用数量 ![](...) */
  imageCount: number
  /** 有问题的引用（空 URL 等），上限 5 条 */
  issues: MarkdownRefIssue[]
}

export interface MarkdownRefIssue {
  /** 引用类型 */
  kind: 'link' | 'image'
  /** 出问题的原文片段（供 Ctrl+F 定位） */
  snippet: string
  /** 问题原因 */
  reason: 'empty-url'
}

/** issues 数量上限：避免长文一次性列出几十条淹没真正的问题 */
const MAX_ISSUES = 5

/** 匹配行内链接与图片：![alt](url) 或 [text](url)，url 部分不含右括号 */
const REF_PATTERN = /(!?)\[([^\]\n]*)\]\(([^)\n]*)\)/g

/**
 * 扫描 Markdown 中的链接与图片引用。
 * 目前只判定「URL 为空」这一种确定性错误（作者常写了 `[]()` 占位却忘记填充）。
 */
export function scanMarkdownRefs(content: string): MarkdownRefScan {
  const result: MarkdownRefScan = { linkCount: 0, imageCount: 0, issues: [] }
  if (!content) return result

  // 去掉围栏代码块内容，避免把代码里的示例链接算进来
  const stripped = stripFencedCode(content)

  REF_PATTERN.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = REF_PATTERN.exec(stripped)) !== null) {
    const isImage = match[1] === '!'
    const url = match[3].trim()
    if (isImage) {
      result.imageCount += 1
    } else {
      result.linkCount += 1
    }
    if (url === '' && result.issues.length < MAX_ISSUES) {
      result.issues.push({
        kind: isImage ? 'image' : 'link',
        snippet: match[0],
        reason: 'empty-url',
      })
    }
  }
  return result
}

/** 移除 ``` 与 ~~~ 围栏代码块（含内容），保留其余文本 */
export function stripFencedCode(markdown: string): string {
  const lines = markdown.split('\n')
  const out: string[] = []
  let fence: string | null = null
  for (const line of lines) {
    const trimmed = line.trimStart()
    const marker = trimmed.startsWith('```') ? '```' : trimmed.startsWith('~~~') ? '~~~' : null
    if (fence === null) {
      if (marker) {
        fence = marker
        continue
      }
      out.push(line)
    } else if (marker === fence) {
      fence = null
    }
  }
  return out.join('\n')
}

/** 中文字数统计口径：汉字按 1 字，连续 ASCII 单词按 1 字，其余符号不计 */
export function countWords(content: string): number {
  if (!content) return 0
  const stripped = stripFencedCode(content)
    // 去掉 Markdown 语法噪声，避免把标记算成字
    .replace(/!?\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/[#>*_`~\-|]/g, ' ')
  const han = stripped.match(/[\u4e00-\u9fa5]/g)?.length ?? 0
  const words = stripped.match(/[A-Za-z0-9]+/g)?.length ?? 0
  return han + words
}

/** 预计阅读时长（分钟，向上取整，最少 1 分钟） */
export function estimateReadingMinutes(words: number): number {
  if (words <= 0) return 0
  return Math.max(1, Math.ceil(words / 400))
}
