import { ref } from 'vue'

/**
 * Markdown 渲染区域的交互增强（事件委托，v-html 内容上的点击统一在此处理）：
 * - .code-copy 按钮 → 复制对应代码块
 * - 点击图片 → 打开 lightbox 预览
 * 与 Lightbox 组件配套使用。
 */

function fallbackCopy(text: string): boolean {
  // http 内网环境没有 navigator.clipboard，退回 execCommand
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  document.body.removeChild(textarea)
  return ok
}

async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      /* 落到 fallback */
    }
  }
  return fallbackCopy(text)
}

export function useMarkdownActions() {
  const preview = ref<{ src: string; alt: string } | null>(null)

  async function onContentClick(e: MouseEvent): Promise<void> {
    const target = e.target as HTMLElement

    const copyBtn = target.closest<HTMLButtonElement>('.code-copy')
    if (copyBtn) {
      const code = copyBtn.parentElement?.querySelector('pre')?.textContent ?? ''
      const ok = await copyText(code)
      const original = '复制'
      copyBtn.textContent = ok ? '已复制' : '复制失败'
      copyBtn.classList.toggle('copied', ok)
      window.setTimeout(() => {
        copyBtn.textContent = original
        copyBtn.classList.remove('copied')
      }, 1500)
      return
    }

    if (target instanceof HTMLImageElement && target.src) {
      preview.value = { src: target.src, alt: target.alt }
    }
  }

  function closePreview(): void {
    preview.value = null
  }

  return { preview, onContentClick, closePreview }
}
