import { computed, ref } from 'vue'
import { theme as antTheme } from 'ant-design-vue'
import { antdDarkTokens, antdTokens } from './tokens'

/**
 * 管理端主题状态(docs/09 §8.1)
 * 机制:AntD darkAlgorithm + 显式 token 双表(显式色值会压过 algorithm,故按主题切换整表);
 * 布局层 --admin-* 由 main.css 的 html.dark 分支接管;偏好存 localStorage,缺省跟随系统。
 */

const THEME_KEY = 'blog_admin_theme'

export type AdminTheme = 'light' | 'dark'

function initialTheme(): AdminTheme {
  try {
    const saved = localStorage.getItem(THEME_KEY)
    if (saved === 'dark' || saved === 'light') return saved
  } catch {
    /* localStorage 不可用时跟随系统 */
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

const current = ref<AdminTheme>(initialTheme())

function applyToDom(value: AdminTheme) {
  document.documentElement.classList.toggle('dark', value === 'dark')
}
applyToDom(current.value)

/** 图表等非 setup 上下文也可读取的主题响应式引用 */
export const adminTheme = current

export function useAdminTheme() {
  const isDark = computed(() => current.value === 'dark')

  function toggle() {
    current.value = current.value === 'dark' ? 'light' : 'dark'
    try {
      localStorage.setItem(THEME_KEY, current.value)
    } catch {
      /* 存储失败不影响本次切换 */
    }
    applyToDom(current.value)
  }

  const themeConfig = computed(() => ({
    algorithm: current.value === 'dark' ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
    token: current.value === 'dark' ? { ...antdDarkTokens } : { ...antdTokens },
  }))

  return { isDark, toggle, themeConfig }
}
