/** SEO meta 动态设置：canonical/OG/Twitter/JSON-LD（title 由 utils/title.ts 负责） */

interface SeoOptions {
  /** 页面标题（og/twitter 用；不传取 document.title） */
  pageTitle?: string
  description?: string
  keywords?: string
  /** 自定义 canonical（文章 SEO 字段，空=当前页地址） */
  canonical?: string
  /** 自定义 OG 图（文章 SEO 字段，空=用封面） */
  ogImage?: string
  /** OG 类型：文章用 article，其余用 website */
  ogType?: 'website' | 'article'
  /** 文章页额外信息，用于 JSON-LD Article */
  article?: {
    headline: string
    publishedAt?: string
    modifiedAt?: string
    author?: string
    cover?: string
  }
  /** FAQ 结构化数据（由 Markdown FAQ 小节推导） */
  faq?: { question: string; answer: string }[]
}

function upsertMeta(attr: 'name' | 'property', key: string, content: string): void {
  let el = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${key}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attr, key)
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

function upsertLink(rel: string, href: string): void {
  let el = document.head.querySelector<HTMLLinkElement>(`link[rel="${rel}"]`)
  if (!el) {
    el = document.createElement('link')
    el.setAttribute('rel', rel)
    document.head.appendChild(el)
  }
  el.setAttribute('href', href)
}

/** 当前页 canonical 地址（部署在任意域名下都成立） */
function currentCanonical(): string {
  return window.location.origin + window.location.pathname
}

/** 路由切换时调用；不传的项会清回空值，避免串页 */
export function setSeo(options: SeoOptions): void {
  const {
    description = '',
    keywords = '',
    ogType = 'website',
    article,
    pageTitle,
    canonical: customCanonical,
    ogImage: customOgImage,
    faq,
  } = options
  const pageName = pageTitle || document.title
  const canonical = customCanonical || currentCanonical()
  const ogImage = customOgImage || article?.cover || ''

  upsertLink('canonical', canonical)
  upsertMeta('name', 'description', description)
  upsertMeta('name', 'keywords', keywords)
  upsertMeta('property', 'og:title', pageName)
  upsertMeta('property', 'og:description', description)
  upsertMeta('property', 'og:type', ogType)
  upsertMeta('property', 'og:url', canonical)
  if (ogImage) {
    upsertMeta('property', 'og:image', ogImage)
  } else {
    document.head.querySelector<HTMLMetaElement>('meta[property="og:image"]')?.remove()
  }
  upsertMeta('name', 'twitter:card', ogImage ? 'summary_large_image' : 'summary')
  upsertMeta('name', 'twitter:title', pageName)
  upsertMeta('name', 'twitter:description', description)

  setJsonLd(article, canonical)
  setFaqJsonLd(faq)
}

/** 文章页注入 JSON-LD Article；非文章页移除，避免残留 */
function setJsonLd(article: SeoOptions['article'], canonical: string): void {
  const ID = 'ld-json-article'
  document.getElementById(ID)?.remove()
  if (!article) return

  const data: Record<string, unknown> = {
    '@context': 'https://schema.org',
    '@type': 'BlogPosting',
    headline: article.headline,
    mainEntityOfPage: { '@type': 'WebPage', '@id': canonical },
    ...(article.publishedAt ? { datePublished: article.publishedAt } : {}),
    ...(article.modifiedAt ? { dateModified: article.modifiedAt } : {}),
    ...(article.cover ? { image: [article.cover] } : {}),
    author: { '@type': 'Person', name: article.author || document.title.split('·').pop()?.trim() || '博主' }
  }
  const el = document.createElement('script')
  el.type = 'application/ld+json'
  el.id = ID
  el.textContent = JSON.stringify(data)
  document.head.appendChild(el)
}

/**
 * FAQ JSON-LD（契约十四）：由 Markdown 约定推导——「## FAQ」小节下的「### 问题」+ 紧随段落为答案。
 * 仅在确有 FAQ 结构时输出，不做编造数据。
 */
export function extractFaqFromMarkdown(markdown: string): { question: string; answer: string }[] {
  const lines = markdown.split('\n')
  const items: { question: string; answer: string }[] = []
  let inFaq = false
  let current: { question: string; answer: string } | null = null
  for (const raw of lines) {
    const line = raw.trim()
    if (/^##\s+/.test(line)) {
      // 进入/离开 FAQ 小节
      inFaq = /^##\s+faq/i.test(line) || /^##\s+常见问题/.test(line)
      if (current) {
        items.push(current)
        current = null
      }
      continue
    }
    if (!inFaq) continue
    if (/^###\s+/.test(line)) {
      if (current) items.push(current)
      current = { question: line.replace(/^###\s+/, '').trim(), answer: '' }
      continue
    }
    if (current && line !== '' && !/^#/.test(line)) {
      current.answer = current.answer ? `${current.answer} ${line}` : line
    }
  }
  if (current) items.push(current)
  return items.filter((item) => item.question && item.answer)
}

/** 注入/移除 FAQPage JSON-LD */
function setFaqJsonLd(faq?: { question: string; answer: string }[]): void {
  const ID = 'ld-json-faq'
  document.getElementById(ID)?.remove()
  if (!faq || faq.length === 0) return
  const data = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: faq.map((item) => ({
      '@type': 'Question',
      name: item.question,
      acceptedAnswer: { '@type': 'Answer', text: item.answer },
    })),
  }
  const el = document.createElement('script')
  el.type = 'application/ld+json'
  el.id = ID
  el.textContent = JSON.stringify(data)
  document.head.appendChild(el)
}
