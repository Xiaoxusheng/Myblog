<template>
  <div class="container page-layout">
    <section class="page-main">
      <header class="list-head">
        <h1 class="page-heading">标签：{{ tag?.name || '加载中' }}</h1>
      </header>

      <ListSkeleton v-if="loading" />
      <ErrorState v-else-if="error" :message="error" @retry="list.load()" />
      <ErrorState
        v-else-if="site.loadError && !tag"
        message="站点信息加载失败，请稍后重试"
        @retry="initSlug"
      />

      <template v-else>
        <div v-if="items.length" class="post-list">
          <ArticleCard v-for="post in items" :key="post.id" :post="post" />
        </div>
        <EmptyState
          v-else-if="notFound"
          title="标签不存在"
          description="它可能已被删除，去标签页看看其他内容吧。"
        >
          <RouterLink class="btn" to="/tags">浏览全部标签</RouterLink>
        </EmptyState>
        <EmptyState v-else title="该标签下还没有文章" description="博主正在努力创作中…" />

        <Pagination
          v-if="items.length"
          :page="page"
          :total-pages="totalPages"
          @change="handlePageChange"
        />
      </template>
    </section>
    <SiteSidebar />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
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
import SiteSidebar from '@/components/layout/SiteSidebar.vue'

const route = useRoute()
const router = useRouter()
const site = useSiteStore()

const slug = computed(() => String(route.params.slug))
const tag = computed(() => site.tags.find((t) => t.slug === slug.value))
const notFound = computed(
  () => site.loaded && !site.loadError && site.tags.length > 0 && !tag.value
)

const list = usePagedList<PostSummary>(
  ({ page, pageSize, signal }) => fetchPosts({ page, pageSize, tagId: tag.value?.id }, signal),
  site.settings.postPageSize || 10
)
const { loading, error, items, total, page } = list

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / list.pageSize.value)))

function handlePageChange(next: number): void {
  list.goToPage(next)
  // 同步到 URL，便于分享与后退（与首页一致，docs/08 §7.4）
  void router.replace({
    query: { ...route.query, page: next > 1 ? String(next) : undefined }
  })
}

// 浏览器前进/后退时同步页码
watch(
  () => route.query.page,
  () => {
    const qp = Number(route.query.page) || 1
    if (qp !== list.page.value) {
      list.page.value = qp
      void list.load()
    }
  }
)

/** 确保站点信息就绪后，再按标签加载文章（重试入口复用） */
async function initSlug(): Promise<void> {
  await site.ensureLoaded()
  if (tag.value) {
    list.page.value = 1
    await list.load()
  }
}

// 标签切换（组件复用）时重新加载
watch(slug, () => void initSlug(), { immediate: true })

onMounted(() => {
  void site.ensureLoaded()
})
</script>

<style scoped>
.page-main {
  min-width: 0;
}

.list-head {
  margin-bottom: 18px;
}

.post-list {
  display: flex;
  flex-direction: column;
}
</style>
