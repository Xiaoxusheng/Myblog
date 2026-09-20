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

// heading_close：h2-h4 闭合前注入悬停锚点（# 悬挂于列外，不入正文文字流，选中复制不受影响）
// token 流为 heading_open → inline → heading_close，向前查找最近的 heading_open 取 id
md.renderer.rules.heading_close = (tokens, idx) => {
  const level = Number(tokens[idx].tag.slice(1)) || 1
  let id: string | null = null
  for (let i = idx - 1; i >= 0; i--) {
    if (tokens[i].type === 'heading_open') {
      id = tokens[i].attrGet('id')
      break
    }
  }
  if (level >= 2 && level <= 4 && id) {
    return `<a class="header-anchor" href="#${id}" aria-label="标题锚点">#</a>${md.renderer.renderToken(tokens, idx, md.options)}`
  }
  return md.renderer.renderToken(tokens, idx, md.options)
}

// fence 覆写：代码块包一层带语言标签/可选文件名标题/行号的 .code-block
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

/** 高亮语言别名归一：不依赖 highlight.js 内建别名表是否随 core 注册 */
const LANG_ALIAS: Record<string, string> = {
  js: 'javascript',
  jsx: 'javascript',
  mjs: 'javascript',
  ts: 'typescript',
  sh: 'bash',
  shell: 'bash',
  py: 'python',
  yml: 'yaml',
  golang: 'go',
  md: 'markdown',
  html: 'xml'
}

/** 解析 fence info 串：首 token 为语言，其后支持 title=main.go（引号可选）标注文件名 */
function parseFenceInfo(raw: string): { lang: string; title: string } {
  const info = (raw || '').trim()
  if (!info) return { lang: '', title: '' }
  const lang = info.split(/\s+/)[0].toLowerCase()
  const m = info.match(/(?:^|\s)title=(?:"([^"]*)"|'([^']*)'|(\S+))/i)
  const title = m ? (m[1] ?? m[2] ?? m[3] ?? '') : ''
  return { lang, title }
}

/** 高亮返回空（未注册语言/无语言）时回退转义原文，保证代码块始终可见而非空块 */
function highlightFence(code: string, rawInfo: string): string {
  const first = (rawInfo || '').trim().split(/\s+/)[0].toLowerCase()
  const name = LANG_ALIAS[first] ?? first
  if (name && hljs.getLanguage(name)) {
    try {
      return hljs.highlight(code, { language: name, ignoreIllegals: true }).value
    } catch {
      /* 高亮失败回退到默认转义输出 */
    }
  }
  return ''
}

/**
 * 将高亮后的 HTML 按行拆分并逐行包上 .code-line（供 CSS 计数器生成行号）。
 * hljs 的 span 可能跨行，拆分时在行尾补闭合、行首重开，保证每行标签自洽；
 * 行间保留真实换行符，使 pre.textContent 与原始代码一致（复制不带行号、不丢换行）。
 */
function splitHighlightedLines(html: string): string[] {
  const parts = html.split('\n')
  if (parts.length > 1 && parts[parts.length - 1] === '') parts.pop()
  const openStack: string[] = []
  const out: string[] = []
  for (const part of parts) {
    const prefix = openStack.join('')
    const tagRe = /<span\b[^>]*>|<\/span>/g
    let m: RegExpExecArray | null
    while ((m = tagRe.exec(part)) !== null) {
      if (m[0][1] === '/') openStack.pop()
      else openStack.push(m[0])
    }
    out.push(prefix + part + '</span>'.repeat(openStack.length))
  }
  return out
}

md.renderer.rules.fence = (tokens, idx) => {
  const token = tokens[idx]
  const { lang, title } = parseFenceInfo(token.info)
  const label = LANG_LABEL[lang] ?? lang
  const titleHtml = title
    ? `<span class="code-title">${md.utils.escapeHtml(title)}</span>`
    : ''
  const langLabel = label ? `<span class="code-lang">${md.utils.escapeHtml(label)}</span>` : ''
  const highlighted = highlightFence(token.content, token.info)
  const codeHtml = highlighted || md.utils.escapeHtml(token.content)
  const cls = lang ? ` class="language-${md.utils.escapeHtml(lang)}"` : ''
  const body = splitHighlightedLines(codeHtml.replace(/\n$/, ''))
    .map((line) => `<span class="code-line">${line}</span>`)
    .join('\n')
  return `<div class="code-block${title ? ' has-title' : ''}">${titleHtml}${langLabel}<button class="code-copy" type="button">复制</button><pre><code${cls}>${body}</code></pre></div>
`
}

export interface RenderedMarkdown {
  html: string
  toc: TocItem[]
}

export function renderMarkdown(source: string): RenderedMarkdown {
  tocResult = []
  idCount = {}
  let html = md.render(source ?? '')
  // 仅 http(s) 外链新窗口 + ↗ 标识；站内相对链接交由 SPA 接管、页内锚点原地跳转
  html = html.replace(
    /<a href="(https?:\/\/[^"]*)"([^>]*)>/g,
    '<a class="ext-link" target="_blank" rel="noopener noreferrer" href="$1"$2>'
  )
  // 图片懒加载 + 异步解码（渲染后的确定性字符串处理）
  html = html.replace(/<img\s/g, '<img loading="lazy" decoding="async" ')
  return { html, toc: tocResult }
}
