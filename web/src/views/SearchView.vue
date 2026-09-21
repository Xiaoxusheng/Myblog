<template>
  <div class="container search-page">
    <header class="search-head">
      <form class="search-big" role="search" @submit.prevent="submitSearch">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden="true">
          <path
            d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 1 0-.7.7l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0A4.5 4.5 0 1 1 14 9.5 4.5 4.5 0 0 1 9.5 14z"
          />
        </svg>
        <input
          ref="inputEl"
          v-model="input"
          type="search"
          placeholder="搜索文章标题、摘要与正文…"
          aria-label="搜索关键词"
          @keydown.esc="inputEl?.blur()"
        />
        <span class="kbd search-esc">ESC</span>
      </form>

      <div v-if="keyword" class="result-head">
        <p class="result-stat">找到 {{ total }} 篇与「{{ keyword }}」相关的文章</p>
        <span v-if="elapsedMs !== null" class="result-time">耗时 {{ elapsedText }}</span>
      </div>
      <p v-else class="result-hint">输入关键词后回车开始搜索</p>
    </header>

    <ListSkeleton v-if="loading" />
    <ErrorState v-else-if="error" :message="error" @retry="list.load()" />

    <template v-else>
      <div v-if="keyword && items.length" class="post-list">
        <ArticleCard
          v-for="(post, index) in items"
          :key="post.id"
          :post="post"
          :keyword="keyword"
          :tint-index="index"
        />
      </div>
      <EmptyState
        v-else-if="keyword"
        title="没有找到相关文章"
        description="没有找到与关键词相关的内容，换个关键词，或从下面继续浏览。"
      >
        <div class="empty-actions">
          <RouterLink class="btn" to="/">查看全部文章</RouterLink>
          <RouterLink class="btn btn-ghost" to="/categories">浏览分类</RouterLink>
        </div>
      </EmptyState>
      <EmptyState
        v-else
        title="输入关键词开始搜索"
        description="例如：Markdown、Go、部署……"
      />

      <Pagination
        v-if="keyword && items.length"
        :page="page"
        :total-pages="totalPages"
        @change="handlePageChange"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchPosts } from '@/api/post'
import { usePagedList } from '@/composables/usePagedList'
import { useSiteStore } from '@/stores/site'
import type { PostSummary } from '@/types'
import ArticleCard from '@/components/common/ArticleCard.vue'
import ListSkeleton from '@/components/common/ListSkeleton.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'
import Pagination from '@/components/common/Pagination.vue'

const route = useRoute()
const router = useRouter()
const site = useSiteStore()

const keyword = computed(() => String(route.query.keyword ?? '').trim())
const input = ref(keyword.value)
const inputEl = ref<HTMLInputElement | null>(null)

const list = usePagedList<PostSummary>(
  ({ page, pageSize, signal }) => fetchPosts({ page, pageSize, keyword: keyword.value }, signal),
  site.settings.postPageSize || 10
)
const { loading, error, items, total, page } = list

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / list.pageSize.value)))

/** 客户端感知的搜索耗时（请求发出到当前列表就绪） */
const elapsedMs = ref<number | null>(null)
const elapsedText = computed(() =>
  elapsedMs.value === null ? '' : elapsedMs.value >= 1000 ? `${(elapsedMs.value / 1000).toFixed(1)} 秒` : `${elapsedMs.value} ms`
)

let searchStartedAt = 0
watch(loading, (isLoading) => {
  if (isLoading) {
    searchStartedAt = performance.now()
  } else if (searchStartedAt) {
    elapsedMs.value = Math.max(1, Math.round(performance.now() - searchStartedAt))
    searchStartedAt = 0
  }
})

function submitSearch(): void {
  const kw = input.value.trim()
  if (!kw || kw === keyword.value) return
  void router.push({ path: '/search', query: { keyword: kw } })
}

function handlePageChange(next: number): void {
  list.goToPage(next)
  // page 追加到 URL（keyword 仍为主键）
  void router.replace({
    query: { ...route.query, page: next > 1 ? String(next) : undefined }
  })
}

// 关键词变化（含浏览器前进/后退）时重新搜索
watch(
  keyword,
  (kw) => {
    input.value = kw
    if (kw) {
      list.reset(site.settings.postPageSize || 10)
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.search-page {
  max-width: 860px;
  padding-bottom: 64px;
}

.search-head {
  padding-top: 40px;
}

/* 大搜索框：紫色描边 + 右侧 ESC 徽标（p09） */
.search-big {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 8px 8px 16px;
  border: 1.5px solid var(--brand);
  border-radius: var(--radius-md);
  background: var(--bg);
  transition: box-shadow var(--transition);
}

.search-big:focus-within {
  box-shadow: 0 0 0 3px var(--brand-soft);
}

.search-big svg {
  flex-shrink: 0;
  color: var(--text-3);
}

.search-big input {
  flex: 1;
  min-width: 0;
  height: 34px;
  border: none;
  outline: none;
  background: transparent;
  font-size: var(--fs-base);
  color: var(--text-1);
}

.search-big input::placeholder {
  color: var(--text-3);
}

.search-esc {
  flex-shrink: 0;
  margin-right: 4px;
}

.result-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  margin-top: 18px;
}

.result-stat {
  font-size: var(--fs-base);
  color: var(--text-1);
}

.result-stat::first-letter {
  font-weight: 600;
}

.result-time {
  flex-shrink: 0;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.result-hint {
  margin-top: 18px;
  font-size: var(--fs-sm);
  color: var(--text-3);
}

.post-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 20px;
}

.empty-actions {
  display: flex;
  gap: 10px;
}

/* 关键词命中标记：低饱和品牌底，避免刺眼 */
:deep(mark) {
  padding: 0 1px;
  border-radius: 3px;
  background: var(--brand-soft);
  color: var(--brand);
  font-weight: 600;
}
</style>
