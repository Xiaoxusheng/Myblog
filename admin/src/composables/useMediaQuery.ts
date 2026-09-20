import { onBeforeUnmount, ref, type Ref } from 'vue'

/**
 * 响应式媒体查询（SSR 无关的轻封装）。
 * 挂载时同步当前命中状态并监听变化，卸载时移除监听，避免事件残留。
 */
export function useMediaQuery(query: string): Ref<boolean> {
  const supported = typeof window !== 'undefined' && typeof window.matchMedia === 'function'
  const matches = ref(supported ? window.matchMedia(query).matches : false)
  let mql: MediaQueryList | null = null

  function onChange(event: MediaQueryListEvent): void {
    matches.value = event.matches
  }

  if (supported) {
    mql = window.matchMedia(query)
    matches.value = mql.matches
    mql.addEventListener('change', onChange)
  }

  onBeforeUnmount(() => {
    mql?.removeEventListener('change', onChange)
    mql = null
  })

  return matches
}
