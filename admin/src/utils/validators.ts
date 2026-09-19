/**
 * 后台表单共享校验常量(docs/09 §7.2)
 * Slug/URL 规则单一来源:分类/标签/文章/页面编辑共用,禁止各页复制正则。
 */

/** Slug:仅字母、数字、短横线和下划线(可空,是否必填由各页面语义决定) */
export const SLUG_PATTERN = /^[a-zA-Z0-9_-]*$/

/** Slug 必填版(页面必须指定唯一 slug) */
export const SLUG_REQUIRED_PATTERN = /^[a-zA-Z0-9_-]+$/

export const SLUG_PATTERN_MESSAGE = '仅支持字母、数字、短横线和下划线'

/** 站点 URL:http(s) 开头 */
export const SITE_URL_PATTERN = /^https?:\/\/\S*$/
