import { ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchSite } from '@/api/site'
import type { Category, Settings, Tag } from '@/types'

export const DEFAULT_SETTINGS: Settings = {
  siteName: 'MyBlog',
  siteDescription: '',
  siteKeywords: '',
  siteUrl: '',
  logo: '',
  notice: '',
  icp: '',
  footerText: '',
  commentEnabled: true,
  postPageSize: 10
}

/** 站点信息：App 启动时拉取一次，Header/Footer/侧栏/分页大小共用 */
export const useSiteStore = defineStore('site', () => {
  const settings = ref<Settings>({ ...DEFAULT_SETTINGS })
  const categories = ref<Category[]>([])
  const tags = ref<Tag[]>([])
  const loaded = ref(false)
  const loadError = ref(false)

  let pending: Promise<void> | null = null

  async function load(): Promise<void> {
    try {
      const data = await fetchSite()
      settings.value = { ...DEFAULT_SETTINGS, ...(data?.settings ?? {}) }
      if (!Number.isFinite(settings.value.postPageSize) || settings.value.postPageSize < 1) {
        settings.value.postPageSize = 10
      }
      categories.value = data?.categories ?? []
      tags.value = data?.tags ?? []
      loadError.value = false
    } catch {
      loadError.value = true
    } finally {
      loaded.value = true
      pending = null
    }
  }

  /** 幂等加载：并发调用共享同一请求;失败后再次调用会重试;force 强制刷新 */
  function ensureLoaded(force = false): Promise<void> {
    if (force || loadError.value || (!loaded.value && !pending)) {
      pending = load()
    }
    return pending ?? Promise.resolve()
  }

  return { settings, categories, tags, loaded, loadError, ensureLoaded }
})
