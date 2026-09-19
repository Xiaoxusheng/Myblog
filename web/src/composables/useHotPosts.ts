import { ref } from 'vue'
import { fetchPosts } from '@/api/post'
import { isRequestCanceled } from '@/api/http'
import type { PostSummary } from '@/types'

/**
 * 热门文章共享加载（模块级单例）：
 * 桌面端侧栏与移动端发现区共用同一份结果，一个页面会话只发一次请求。
 */
const hotPosts = ref<PostSummary[]>([])
const hotLoading = ref(false)
const hotError = ref('')
let loaded = false
let controller: AbortController | null = null

export function useHotPosts() {
  async function load(): Promise<void> {
    if (loaded && hotPosts.value.length) return
    controller?.abort()
    const local = new AbortController()
    controller = local
    hotLoading.value = true
    hotError.value = ''
    try {
      const data = await fetchPosts({ sort: 'views', page: 1, pageSize: 5 }, local.signal)
      if (controller !== local) return
      hotPosts.value = data?.list ?? []
      loaded = true
    } catch (e) {
      if (controller !== local || isRequestCanceled(e)) return
      hotError.value = e instanceof Error ? e.message : '加载失败'
    } finally {
      if (controller === local) hotLoading.value = false
    }
  }

  return { hotPosts, hotLoading, hotError, load }
}
