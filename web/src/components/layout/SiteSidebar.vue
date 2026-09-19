<template>
  <aside class="page-aside sidebar">
    <section v-if="site.settings.notice" class="side-card card">
      <h3 class="side-title">公告</h3>
      <p class="notice">{{ site.settings.notice }}</p>
    </section>

    <section class="side-card card">
      <h3 class="side-title">热门文章</h3>
      <div v-if="hotLoading" class="hot-skeletons" aria-hidden="true">
        <div v-for="i in 5" :key="i" class="hot-skeleton">
          <span class="skeleton sk-line"></span>
          <span class="skeleton sk-line sk-short"></span>
        </div>
      </div>
      <ErrorState v-else-if="hotError" small :message="hotError" @retry="loadHot" />
      <ol v-else-if="hotPosts.length" class="hot-list">
        <li v-for="(post, index) in hotPosts" :key="post.id" class="hot-row">
          <RouterLink :to="`/post/${post.slug}`" class="hot-item">
            <span class="hot-rank" :class="{ top: index < 3 }">{{ index + 1 }}</span>
            <span class="hot-title">{{ post.title }}</span>
          </RouterLink>
          <span class="hot-views">{{ formatNumber(post.viewCount) }}</span>
        </li>
      </ol>
      <p v-else class="side-empty">暂无热门文章</p>
    </section>

    <section v-if="cloudTags.length" class="side-card card">
      <h3 class="side-title">标签云</h3>
      <div class="tag-cloud">
        <RouterLink
          v-for="tag in cloudTags"
          :key="tag.id"
          :to="`/tag/${tag.slug}`"
          class="chip"
        >
          {{ tag.name }}
        </RouterLink>
      </div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { fetchPosts } from '@/api/post'
import { isRequestCanceled } from '@/api/http'
import { useSiteStore } from '@/stores/site'
import { formatNumber } from '@/utils/format'
import type { PostSummary } from '@/types'
import ErrorState from '@/components/common/ErrorState.vue'

const site = useSiteStore()
const hotPosts = ref<PostSummary[]>([])
const hotLoading = ref(true)
const hotError = ref('')
let controller: AbortController | null = null

const cloudTags = computed(() => site.tags.slice(0, 24))

async function loadHot(): Promise<void> {
  controller?.abort()
  const localController = new AbortController()
  controller = localController
  hotLoading.value = true
  hotError.value = ''
  try {
    const data = await fetchPosts({ sort: 'views', page: 1, pageSize: 5 }, localController.signal)
    if (controller !== localController) return
    hotPosts.value = data?.list ?? []
  } catch (e) {
    if (controller !== localController || isRequestCanceled(e)) return
    hotError.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    if (controller === localController) hotLoading.value = false
  }
}

onMounted(() => {
  void loadHot()
})

onBeforeUnmount(() => {
  controller?.abort()
})
</script>

<style scoped>
.side-card {
  padding: 16px 18px;
}

.side-title {
  position: relative;
  margin: 0 0 12px;
  padding-left: 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-1);
}

.side-title::before {
  content: '';
  position: absolute;
  left: 0;
  top: 3px;
  bottom: 3px;
  width: 3px;
  border-radius: 2px;
  background: var(--brand);
}

.notice {
  font-size: 13.5px;
  color: var(--text-2);
  line-height: 1.8;
  white-space: pre-wrap;
}

/* 热门文章 */
.hot-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.hot-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.hot-row + .hot-row {
  border-top: 1px dashed var(--border);
}

.hot-item {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 0;
}

.hot-rank {
  width: 18px;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
}

.hot-rank.top {
  color: var(--brand);
}

.hot-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13.5px;
  color: var(--text-1);
  transition: color var(--transition);
}

.hot-item:hover .hot-title {
  color: var(--brand);
}

.hot-views {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.hot-skeletons {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 2px 0;
}

.hot-skeleton {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sk-line {
  display: block;
  height: 12px;
  width: 88%;
}

.sk-short {
  width: 55%;
}

.side-empty {
  font-size: 13px;
  color: var(--text-3);
}

/* 标签云 */
.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
