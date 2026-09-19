<template>
  <div class="container page-layout">
    <section class="page-main">
      <header v-reveal class="hero">
        <p class="hero-kicker">PERSONAL BLOG</p>
        <h1 class="hero-title">{{ site.settings.siteName || 'MyBlog' }}</h1>
        <p class="hero-desc">{{ site.settings.siteDescription || '记录、思考与分享' }}</p>
        <div class="hero-stats">
          <span class="hero-stat"><strong>{{ total }}</strong> 文章</span>
          <span class="hero-sep"></span>
          <span class="hero-stat"><strong>{{ site.categories.length }}</strong> 分类</span>
          <span class="hero-sep"></span>
          <span class="hero-stat"><strong>{{ site.tags.length }}</strong> 标签</span>
        </div>
      </header>

      <ListSkeleton v-if="loading" />
      <ErrorState v-else-if="error" :message="error" @retry="reload" />
      <template v-else>
        <div v-if="items.length" class="post-list">
          <div
            v-for="(post, index) in items"
            :key="post.id"
            v-reveal="{ delay: Math.min(index * 55, 275) }"
          >
            <ArticleCard :post="post" />
          </div>
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

.hero {
  position: relative;
  padding: 10px 0 30px;
  margin-bottom: 28px;
  border-bottom: 1px solid var(--border);
}

/* 品牌色的一抹柔光，只作为 Hero 的背景点缀 */
.hero::before {
  content: '';
  position: absolute;
  inset: -28px -20px auto;
  height: 180px;
  background: radial-gradient(
    480px 160px at 12% 0%,
    var(--brand-soft),
    transparent 72%
  );
  pointer-events: none;
}

.hero > * {
  position: relative;
}

.hero-kicker {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.18em;
  color: var(--brand);
}

.hero-title {
  margin-top: 8px;
  font-size: clamp(26px, 4vw, 32px);
  font-weight: 700;
  letter-spacing: 0.3px;
  line-height: 1.3;
}

.hero-desc {
  margin-top: 8px;
  max-width: 560px;
  font-size: 15px;
  color: var(--text-2);
  line-height: 1.75;
}

.hero-stats {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 16px;
  font-size: 13px;
  color: var(--text-3);
}

.hero-stat strong {
  color: var(--text-1);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.hero-sep {
  width: 1px;
  height: 12px;
  background: var(--border-strong);
}

.post-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
