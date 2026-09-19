import { ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { THEME_STORAGE_KEY } from '@/utils/storage'

export type ThemeMode = 'light' | 'dark' | 'auto'

function readStoredMode(): ThemeMode {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY)
    if (value === 'light' || value === 'dark' || value === 'auto') return value
  } catch {
    /* ignore */
  }
  return 'auto'
}

/** 三态主题：亮 / 暗 / 跟随系统;html.dark 切换;localStorage 记忆 */
export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>(readStoredMode())
  const resolved = ref<'light' | 'dark'>('light')

  const media =
    typeof window.matchMedia === 'function'
      ? window.matchMedia('(prefers-color-scheme: dark)')
      : null

  function apply(): void {
    resolved.value =
      mode.value === 'auto' ? (media?.matches ? 'dark' : 'light') : mode.value
    document.documentElement.classList.toggle('dark', resolved.value === 'dark')
    try {
      localStorage.setItem(THEME_STORAGE_KEY, mode.value)
    } catch {
      /* ignore */
    }
  }

  if (media) {
    media.addEventListener('change', () => {
      if (mode.value === 'auto') apply()
    })
  }
  watch(mode, apply, { immediate: true })

  function cycle(): void {
    mode.value = mode.value === 'light' ? 'dark' : mode.value === 'dark' ? 'auto' : 'light'
  }

  return { mode, resolved, cycle }
})
