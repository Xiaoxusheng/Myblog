import { ref } from 'vue'

/** 命令面板单例状态：header 与全局快捷键共用（docs/06 §4） */
const isOpen = ref(false)
let lastTrigger: HTMLElement | null = null

export function useCommandPalette() {
  function open(): void {
    lastTrigger =
      document.activeElement instanceof HTMLElement ? document.activeElement : null
    isOpen.value = true
  }

  function close(): void {
    isOpen.value = false
    lastTrigger?.focus()
    lastTrigger = null
  }

  function toggle(): void {
    if (isOpen.value) close()
    else open()
  }

  return { isOpen, open, close, toggle }
}
