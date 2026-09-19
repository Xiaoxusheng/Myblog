import { useSiteStore } from '@/stores/site'

/** 组合页面标题：`页面 · 站点名` */
export function applyDocumentTitle(routeTitle?: string): void {
  const siteName = useSiteStore().settings.siteName || 'MyBlog'
  document.title = routeTitle ? `${routeTitle} · ${siteName}` : siteName
}
