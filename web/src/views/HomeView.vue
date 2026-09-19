<template>
  <div class="container page-layout">
    <section class="page-main">
      <ListSkeleton v-if="loading" />
      <ErrorState v-else-if="error" :message="error" @retry="reload" />
      <template v-else>
        <div v-if="items.length" class="post-list">
          <ArticleCard v-for="post in items" :key="post.id" :post="post" />
        </div>
        <EmptyState
          v-else
          title="还没有文章"
          description="博主尚未发布内容，过段时间再来看看吧。"
        />
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
import { setSeo } from '@/utils/seo'
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

const list = usePagedList<PostSummary>(
  ({ page, pageSize, signal }) => fetchPosts({ page, pageSize }, signal),
  site.settings.postPageSize || 10
)
const { loading, error, items, total, page } = list

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / list.pageSize.value)))

async function reload(): Promise<void> {
  await site.ensureLoaded()
  list.pageSize.value = site.settings.postPageSize || 10
  await list.load()
}

function handlePageChange(next: number): void {
  list.goToPage(next)
  // 同步到 URL，便于分享与后退
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

onMounted(async () => {
  await site.ensureLoaded()
  list.pageSize.value = site.settings.postPageSize || 10
  const qp = Number(route.query.page) || 1
  if (qp > 1) list.page.value = qp
  await list.load()
  setSeo({
    description: site.settings.siteDescription,
    keywords: site.settings.siteKeywords
  })
})
</script>

<style scoped>
.page-main {
  min-width: 0;
}

.post-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
