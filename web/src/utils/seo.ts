/** SEO meta 动态设置：description/keywords/OG（title 由 utils/title.ts 负责） */

interface SeoOptions {
  description?: string
  keywords?: string
  /** OG 类型：文章用 article，其余用 website */
  ogType?: 'website' | 'article'
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

/** 路由切换时调用；不传的项会清回空值，避免串页 */
export function setSeo(options: SeoOptions): void {
  const { description = '', keywords = '', ogType = 'website' } = options
  const siteName = document.title

  upsertMeta('name', 'description', description)
  upsertMeta('name', 'keywords', keywords)
  upsertMeta('property', 'og:title', siteName)
  upsertMeta('property', 'og:description', description)
  upsertMeta('property', 'og:type', ogType)
}
