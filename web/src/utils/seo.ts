/** SEO meta 动态设置：canonical/OG/Twitter/JSON-LD（title 由 utils/title.ts 负责） */

interface SeoOptions {
  description?: string
  keywords?: string
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
  const { description = '', keywords = '', ogType = 'website', article } = options
  const siteName = document.title
  const canonical = currentCanonical()

  upsertLink('canonical', canonical)
  upsertMeta('name', 'description', description)
  upsertMeta('name', 'keywords', keywords)
  upsertMeta('property', 'og:title', siteName)
  upsertMeta('property', 'og:description', description)
  upsertMeta('property', 'og:type', ogType)
  upsertMeta('property', 'og:url', canonical)
  upsertMeta('name', 'twitter:card', article?.cover ? 'summary_large_image' : 'summary')
  upsertMeta('name', 'twitter:title', siteName)
  upsertMeta('name', 'twitter:description', description)

  setJsonLd(article, canonical)
}

/** 文章页注入 JSON-LD Article；非文章页移除，避免残留 */
function setJsonLd(
  article: SeoOptions['article'],
  canonical: string
): void {
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
