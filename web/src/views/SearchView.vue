<template>
  <div class="container search-page">
    <header class="search-head">
      <h1 class="page-heading">搜索</h1>
      <form class="search-big" role="search" @submit.prevent="submitSearch">
        <svg viewBox="0 0 24 24" width="17" height="17" fill="currentColor" aria-hidden="true">
          <path
            d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 1 0-.7.7l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0A4.5 4.5 0 1 1 14 9.5 4.5 4.5 0 0 1 9.5 14z"
          />
        </svg>
        <input
          v-model="input"
          type="search"
          placeholder="输入关键词，回车或点击搜索"
          aria-label="搜索关键词"
        />
        <button class="btn btn-primary" type="submit">搜索</button>
      </form>
      <p v-if="keyword" class="page-sub result-stat">
        “{{ keyword }}” 的搜索结果：共 {{ total }} 条<span v-if="elapsedMs !== null">，耗时 {{ elapsedText }}</span>
      </p>
      <p v-else class="page-sub">支持搜索文章标题、摘要与正文内容</p>
    </header>

    <ListSkeleton v-if="loading" />
    <ErrorState v-else-if="error" :message="error" @retry="list.load()" />

    <template v-else>
      <div v-if="keyword && items.length" class="post-list">
        <ArticleCard v-for="post in items" :key="post.id" :post="post" :keyword="keyword" />
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

const list = usePagedList<PostSummary>(
  ({ page, pageSize, signal }) => fetchPosts({ page, pageSize, keyword: keyword.value }, signal),
  site.settings.postPageSize || 10
)
const { loading, error, items, total, page } = list

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / list.pageSize.value)))

/** 客户端感知的搜索耗时（请求发出到当前列表就绪） */
const elapsedMs = ref<number | null>(null)
const elapsedText = computed(() =>
  elapsedMs.value === null ? '' : elapsedMs.value >= 1000 ? `${(elapsedMs.value / 1000).toFixed(1)} 秒` : `${elapsedMs.value} 毫秒`
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
  // page 追加到 URL（keyword 仍为主键，docs/08 §7.4）
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
  padding-bottom: 56px;
}

.search-head {
  padding-top: 28px;
}

.search-big {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 18px;
  padding: 6px 6px 6px 14px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--surface);
  transition: border-color var(--transition);
}

.search-big:focus-within {
  border-color: var(--text-3);
}

.search-big svg {
  flex-shrink: 0;
  color: var(--text-3);
}

.search-big input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  font-size: 15px;
  color: var(--text-1);
}

.search-big input::placeholder {
  color: var(--text-3);
}

.result-stat {
  margin-top: 12px;
}

.post-list {
  display: flex;
  flex-direction: column;
  margin-top: 8px;
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
