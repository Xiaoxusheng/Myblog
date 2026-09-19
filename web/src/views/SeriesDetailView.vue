<template>
  <div class="container page-layout">
    <section class="page-main">
      <ListSkeleton v-if="loading" />

      <ErrorState v-else-if="error" :message="error" @retry="load" />

      <EmptyState v-else-if="notFound" title="专题不存在" description="它可能已被删除，或者链接有误。">
        <RouterLink class="btn" to="/series">浏览全部专题</RouterLink>
      </EmptyState>

      <template v-else-if="series">
        <header class="series-head">
          <p class="kicker">SERIES</p>
          <h1 class="page-heading">{{ series.name }}</h1>
          <p v-if="series.description" class="series-desc">{{ series.description }}</p>
          <p class="page-sub">{{ posts.length }} 篇文章</p>
        </header>

        <div v-if="posts.length" class="post-list">
          <ArticleCard v-for="post in posts" :key="post.id" :post="post" />
        </div>
        <EmptyState v-else title="该专题下还没有已发布的文章" description="发布后就会按序出现在这里。" />
      </template>
    </section>
    <SiteSidebar />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { fetchSeriesDetail } from '@/api/series'
import { ApiError } from '@/api/http'
import { applyDocumentTitle } from '@/utils/title'
import { setSeo } from '@/utils/seo'
import type { PostSummary, Series } from '@/types'
import ArticleCard from '@/components/common/ArticleCard.vue'
import ListSkeleton from '@/components/common/ListSkeleton.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'
import SiteSidebar from '@/components/layout/SiteSidebar.vue'

const route = useRoute()
const slug = computed(() => String(route.params.slug))

const series = ref<Series | null>(null)
const posts = ref<PostSummary[]>([])
const loading = ref(true)
const error = ref('')
const notFound = ref(false)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  try {
    const data = await fetchSeriesDetail(slug.value)
    series.value = data.series
    posts.value = data.posts ?? []
    applyDocumentTitle(data.series.name)
    setSeo({
      description: data.series.description || `专题「${data.series.name}」的全部文章`,
    })
  } catch (e) {
    series.value = null
    posts.value = []
    if (e instanceof ApiError && e.code === 10004) {
      notFound.value = true
    } else {
      error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
    }
    applyDocumentTitle('专题')
    setSeo({})
  } finally {
    loading.value = false
  }
}

// 专题切换（组件复用）时重新加载
watch(slug, () => void load(), { immediate: true })
</script>

<style scoped>
.page-main {
  min-width: 0;
}

.series-head {
  margin-bottom: 18px;
}

.series-head .kicker {
  margin-bottom: 10px;
  color: var(--brand);
}

.series-desc {
  margin-top: 10px;
  font-size: 14px;
  color: var(--text-2);
  line-height: 1.75;
  max-width: var(--reading-width);
}

.series-head .page-sub {
  margin-top: 8px;
}

.post-list {
  display: flex;
  flex-direction: column;
}
</style>
