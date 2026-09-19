<template>
  <div class="container page-layout">
    <section class="page-main">
      <header v-reveal class="hero">
        <h1 class="hero-title">{{ site.settings.siteName || 'MyBlog' }}</h1>
        <p class="hero-desc">{{ site.settings.siteDescription || '记录、思考与分享' }}</p>
        <p class="hero-stats">
          <span><strong>{{ total }}</strong> POSTS</span>
          <span class="hero-sep"></span>
          <span><strong>{{ site.categories.length }}</strong> CATEGORIES</span>
          <span class="hero-sep"></span>
          <span><strong>{{ site.tags.length }}</strong> TAGS</span>
        </p>
      </header>

      <ListSkeleton v-if="loading" />
      <ErrorState v-else-if="error" :message="error" @retry="reload" />
      <template v-else>
        <!-- 置顶文章升级为 Featured 层级（仅第一页） -->
        <article v-if="featured" v-reveal class="featured" itemscope itemtype="https://schema.org/BlogPosting">
          <div class="featured-main">
            <div class="featured-flags">
              <span class="featured-kicker">FEATURED</span>
              <RouterLink
                v-if="featured.category"
                :to="`/category/${featured.category.slug}`"
                class="chip cat"
              >
                {{ featured.category.name }}
              </RouterLink>
            </div>
            <h2 class="featured-title" itemprop="headline">
              <RouterLink :to="`/post/${featured.slug}`">{{ featured.title }}</RouterLink>
            </h2>
            <p class="featured-summary" :class="{ placeholder: !featured.summary }">
              {{ featured.summary || '暂无摘要' }}
            </p>
            <div class="featured-meta">
              <span>{{ formatDate(featured.publishedAt || featured.createdAt) }}</span>
              <span class="meta-dot">·</span>
              <span>阅读 {{ formatNumber(featured.viewCount) }}</span>
              <span class="meta-dot">·</span>
              <span>{{ formatNumber(featured.likeCount) }} 赞</span>
            </div>
          </div>
          <RouterLink
            v-if="featured.cover"
            :to="`/post/${featured.slug}`"
            class="featured-cover"
            tabindex="-1"
            aria-hidden="true"
          >
            <img :src="featured.cover" :alt="featured.title" loading="lazy" />
          </RouterLink>
        </article>

        <div v-if="restPosts.length" class="post-list">
          <div
            v-for="(post, index) in restPosts"
            :key="post.id"
            v-reveal="{ delay: Math.min(index * 55, 275) }"
          >
            <ArticleCard :post="post" />
          </div>
        </div>
        <EmptyState
          v-else-if="!featured"
          title="这里还没有文章"
          description="博主尚未发布内容，过段时间再来看看吧。"
        />
        <Pagination
          v-if="items.length"
          :page="page"
          :total-pages="totalPages"
          @change="handlePageChange"
        />
        <MobileDiscover />
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
import MobileDiscover from '@/components/common/MobileDiscover.vue'
import { formatDate, formatNumber } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const site = useSiteStore()

const list = usePagedList<PostSummary>(
  ({ page, pageSize, signal }) => fetchPosts({ page, pageSize }, signal),
  site.settings.postPageSize || 10
)
const { loading, error, items, total, page } = list

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / list.pageSize.value)))

/** 第一页的首篇置顶文章作为 Featured；无置顶时为 null，列表保持原样 */
const featured = computed<PostSummary | null>(() =>
  page.value === 1 && items.value[0]?.isTop ? (items.value[0] ?? null) : null
)

const restPosts = computed<PostSummary[]>(() => (featured.value ? items.value.slice(1) : items.value))

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
  padding: 18px 0 30px;
  margin-bottom: 2px;
  border-bottom: 1px solid var(--border);
}

.hero-title {
  font-size: var(--fs-display);
  font-weight: 650;
  letter-spacing: -0.015em;
  line-height: 1.22;
}

.hero-desc {
  margin-top: 12px;
  max-width: 560px;
  font-size: 15.5px;
  color: var(--text-2);
  line-height: 1.75;
}

/* mono 大写统计行：编辑感弱信息 */
.hero-stats {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 20px;
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 500;
  letter-spacing: 0.14em;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.hero-stats strong {
  color: var(--text-2);
  font-weight: 600;
}

.hero-sep {
  width: 1px;
  height: 10px;
  background: var(--border-strong);
}

.post-list {
  display: flex;
  flex-direction: column;
}

/* Featured：置顶文章的第一层级 */
.featured {
  display: flex;
  gap: 28px;
  padding: 26px 0;
  border-bottom: 1px solid var(--border);
  align-items: stretch;
}

.featured-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.featured-flags {
  display: flex;
  align-items: center;
  gap: 10px;
}

.featured-kicker {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.16em;
  color: var(--brand);
}

.featured-title {
  margin-top: 10px;
  font-size: clamp(22px, 2.8vw, 26px);
  font-weight: 650;
  line-height: 1.4;
  letter-spacing: -0.01em;
}

.featured-title a {
  color: var(--text-1);
  transition: color var(--transition);
}

.featured-title a:hover {
  color: var(--brand);
}

.featured-summary {
  margin-top: 10px;
  font-size: 14.5px;
  line-height: 1.8;
  color: var(--text-2);
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.featured-summary.placeholder {
  color: var(--text-3);
}

.featured-meta {
  margin-top: auto;
  padding-top: 14px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 12.5px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.meta-dot {
  color: var(--border-strong);
}

.featured-cover {
  flex-shrink: 0;
  align-self: center;
  width: 344px;
  aspect-ratio: 16 / 10;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--surface-2);
}

.featured-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.4s var(--ease-out-quart);
}

.featured:hover .featured-cover img {
  transform: scale(1.03);
}

@media (max-width: 900px) {
  .featured {
    flex-direction: column-reverse;
    gap: 14px;
  }

  .featured-cover {
    width: 100%;
  }
}
</style>
