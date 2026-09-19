import type { Directive, DirectiveBinding } from 'vue'

interface RevealOptions {
  /** 进场延迟（ms），用于列表交错进场 */
  delay?: number
}

const REDUCED_MOTION = '(prefers-reduced-motion: reduce)'

function prefersReducedMotion(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia(REDUCED_MOTION).matches
}

/** 观察到元素进入视口后加 is-revealed（只触发一次），由全局 CSS 完成过渡 */
let observer: IntersectionObserver | null = null

function getObserver(): IntersectionObserver | null {
  if (typeof IntersectionObserver === 'undefined') return null
  if (!observer) {
    observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          entry.target.classList.add('is-revealed')
          observer?.unobserve(entry.target)
        }
      },
      { threshold: 0.05, rootMargin: '0px 0px -24px 0px' }
    )
  }
  return observer
}

export const vReveal: Directive<HTMLElement, RevealOptions | undefined> = {
  // beforeMount：元素插入 DOM 前先隐藏，避免挂载瞬间闪现
  beforeMount(el: HTMLElement, binding: DirectiveBinding<RevealOptions | undefined>) {
    const delay = binding.value?.delay ?? 0
    if (delay > 0) el.style.setProperty('--reveal-delay', `${delay}ms`)
    // 不支持 IntersectionObserver 或用户偏好减少动效：直接可见
    if (!getObserver() || prefersReducedMotion()) return
    el.classList.add('reveal')
  },
  mounted(el: HTMLElement) {
    if (!el.classList.contains('reveal')) return
    getObserver()?.observe(el)
  },
  unmounted(el: HTMLElement) {
    observer?.unobserve(el)
  }
}
