import http from './http'

/** 站点公开信息（契约 #1 `GET /site`）；登录页需要未认证即可读取站点名 */
export interface PublicSiteInfo {
  siteName: string
  siteDescription: string
  logo: string
}

/**
 * 读取站点公开信息。
 * 注意：该接口无需鉴权（契约 #1 为公开接口），登录页可用它渲染真实站点名。
 * 失败时静默降级为默认名称，绝不能因为读站点名失败而挡住登录。
 */
export function fetchPublicSite(): Promise<PublicSiteInfo> {
  return http.get('/site')
}
