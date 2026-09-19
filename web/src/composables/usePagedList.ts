import { ref, shallowRef } from 'vue'
import { ApiError, isRequestCanceled } from '@/api/http'
import type { Paged } from '@/types'

export interface PagedRequest {
  page: number
  pageSize: number
  signal: AbortSignal
}

/**
 * 分页列表通用状态机：loading / error / items / total / 分页跳转
 * - 连续请求自动取消前一个(abort)，过期响应不覆盖新状态
 * - fetcher 由调用方注入（首页/分类/标签/搜索各自拼参数）
 */
export function usePagedList<T>(
  fetcher: (req: PagedRequest) => Promise<Paged<T>>,
  initialPageSize = 10
) {
  const loading = ref(false)
  const error = ref('')
  const items = shallowRef<T[]>([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(initialPageSize)

  let controller: AbortController | null = null

  async function load(): Promise<void> {
    controller?.abort()
    const localController = new AbortController()
    controller = localController

    loading.value = true
    error.value = ''
    try {
      const data = await fetcher({
        page: page.value,
        pageSize: pageSize.value,
        signal: localController.signal
      })
      if (controller !== localController) return // 已被更新的请求取代
      items.value = data?.list ?? []
      total.value = data?.total ?? 0
      if (data?.page) page.value = data.page
      if (data?.pageSize) pageSize.value = data.pageSize
    } catch (e) {
      if (controller !== localController) return
      if (isRequestCanceled(e)) return
      error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
      items.value = []
    } finally {
      if (controller === localController) loading.value = false
    }
  }

  function goToPage(next: number): void {
    if (next === page.value) return
    page.value = next
    // 翻页统一回顶（首页经 router scrollBehavior 回顶，此处幂等）
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    window.scrollTo({ top: 0, behavior: reduce ? 'auto' : 'smooth' })
    void load()
  }

  function reset(nextPageSize?: number): void {
    page.value = 1
    if (nextPageSize && nextPageSize > 0) pageSize.value = nextPageSize
    void load()
  }

  return { loading, error, items, total, page, pageSize, load, goToPage, reset }
}
