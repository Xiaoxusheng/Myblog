/**
 * Markdown 渲染：markdown-it + highlight.js（按需注册语言）
 * - 不放行内嵌 HTML(html:false)，内容来自后台管理员，仍然默认转义，防 XSS
 * - heading_open 覆写：自动生成标题 id（支持中文），同时收集目录数据
 * - 外链统一新窗口打开，图片统一懒加载
 */
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import type { LanguageFn } from 'highlight.js'
import bash from 'highlight.js/lib/languages/bash'
import css from 'highlight.js/lib/languages/css'
import diff from 'highlight.js/lib/languages/diff'
import go from 'highlight.js/lib/languages/go'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import markdownLang from 'highlight.js/lib/languages/markdown'
import python from 'highlight.js/lib/languages/python'
import sql from 'highlight.js/lib/languages/sql'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import yaml from 'highlight.js/lib/languages/yaml'

export interface TocItem {
  id: string
  text: string
  level: number
}

const languages: Array<[string, LanguageFn]> = [
  ['bash', bash],
  ['css', css],
  ['diff', diff],
  ['go', go],
  ['java', java],
  ['javascript', javascript],
  ['json', json],
  ['markdown', markdownLang],
  ['python', python],
  ['sql', sql],
  ['typescript', typescript],
  ['xml', xml],
  ['yaml', yaml]
]
for (const [name, lang] of languages) {
  hljs.registerLanguage(name, lang)
}

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: false
})

/** 生成标题锚点 id:保留中文/字母/数字/连字符 */
function slugify(text: string): string {
  const slug = text
    .trim()
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s-]/gu, '')
    .replace(/\s+/g, '-')
  return slug || 'heading'
}

// 渲染是同步单线程的，模块级状态在每次 renderMarkdown 时重置即可
let tocResult: TocItem[] = []
let idCount: Record<string, number> = {}

function uniqueId(raw: string): string {
  const count = (idCount[raw] ?? 0) + 1
  idCount[raw] = count
  return count === 1 ? raw : `${raw}-${count}`
}

md.renderer.rules.heading_open = (tokens, idx) => {
  const token = tokens[idx]
  const level = Number(token.tag.slice(1)) || 1
  const inlineToken = tokens[idx + 1]
  const text = (inlineToken?.children ?? [])
    .map((child) => child.content)
    .join('')
    .trim()
  const id = uniqueId(slugify(text))
  token.attrSet('id', id)
  if (level <= 4) {
    tocResult.push({ id, text: text || `标题 ${tocResult.length + 1}`, level })
  }
  return md.renderer.renderToken(tokens, idx, md.options)
}

// fence 覆写：代码块包一层带语言标签的 .code-block（语言名取自 info 串，未知语言原样显示）
const LANG_LABEL: Record<string, string> = {
  go: 'Go',
  js: 'JavaScript',
  javascript: 'JavaScript',
  ts: 'TypeScript',
  typescript: 'TypeScript',
  vue: 'Vue',
  html: 'HTML',
  xml: 'XML',
  css: 'CSS',
  scss: 'SCSS',
  json: 'JSON',
  yaml: 'YAML',
  yml: 'YAML',
  toml: 'TOML',
  bash: 'Shell',
  sh: 'Shell',
  shell: 'Shell',
  sql: 'SQL',
  python: 'Python',
  py: 'Python',
  java: 'Java',
  c: 'C',
  cpp: 'C++',
  rust: 'Rust',
  dockerfile: 'Dockerfile',
  diff: 'Diff',
  markdown: 'Markdown'
}

md.renderer.rules.fence = (tokens, idx) => {
  const token = tokens[idx]
  const info = (token.info || '').trim().split(/\s+/)[0].toLowerCase()
  const label = LANG_LABEL[info] ?? (info || '')
  const langLabel = label ? `<span class="code-lang">${md.utils.escapeHtml(label)}</span>` : ''
  // highlight 选项已在 md.options.highlight 中处理高亮
  const highlighted = md.options.highlight?.(token.content, token.info, '') ?? ''
  const cls = info ? ` class="language-${md.utils.escapeHtml(info)}"` : ''
  return `<div class="code-block">${langLabel}<button class="code-copy" type="button">复制</button><pre><code${cls}>${highlighted}</code></pre></div>
`
}

// 代码高亮：命中已注册语言时返回标记片段，由 fence 规则包裹 <pre><code class="language-xx">
md.options.highlight = (code, lang) => {
  if (lang && hljs.getLanguage(lang)) {
    try {
      return hljs.highlight(code, { language: lang, ignoreIllegals: true }).value
    } catch {
      /* 高亮失败回退到默认转义输出 */
    }
  }
  return ''
}

export interface RenderedMarkdown {
  html: string
  toc: TocItem[]
}

export function renderMarkdown(source: string): RenderedMarkdown {
  tocResult = []
  idCount = {}
  let html = md.render(source ?? '')
  // 外链新窗口;图片懒加载（渲染后的确定性字符串处理）
  html = html.replace(/<a\s+href=/g, '<a target="_blank" rel="noopener noreferrer" href=')
  html = html.replace(/<img\s/g, '<img loading="lazy" ')
  return { html, toc: tocResult }
}
