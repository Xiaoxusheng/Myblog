import { reactive } from 'vue'

export type ToastType = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
}

// 模块级单例：ToastHost 渲染 state，任意组件调用 useToast() 弹提示
const state = reactive<{ items: ToastItem[] }>({ items: [] })
let seq = 0

function push(type: ToastType, message: string, duration: number): void {
  const id = ++seq
  state.items.push({ id, type, message })
  window.setTimeout(() => dismiss(id), duration)
}

function dismiss(id: number): void {
  const index = state.items.findIndex((t) => t.id === id)
  if (index >= 0) state.items.splice(index, 1)
}

export function useToast() {
  return {
    state,
    dismiss,
    success: (message: string) => push('success', message, 2600),
    error: (message: string) => push('error', message, 3600),
    info: (message: string) => push('info', message, 2600)
  }
}
