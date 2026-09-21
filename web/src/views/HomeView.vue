<template>
  <div class="home">
    <!-- 通栏深藏蓝 Hero（p02）：装饰几何 + 居中标题 + 双按钮 + 分隔线 + mono 统计行 -->
    <section class="hero">
      <div class="hero-deco" aria-hidden="true">
        <span class="deco deco-pink"></span>
        <span class="deco deco-yellow"></span>
        <span class="deco deco-orange"></span>
        <span class="deco deco-cyan"></span>
        <span class="deco deco-white"></span>
        <span class="deco-square deco-square-1"></span>
        <span class="deco-square deco-square-2"></span>
        <svg class="deco-orbit" viewBox="0 0 1200 420" preserveAspectRatio="none">
          <path d="M-40 300 Q 300 120 640 210 T 1240 150" fill="none" stroke="rgba(255,255,255,0.09)" stroke-width="1.5" />
        </svg>
      </div>

      <div class="hero-inner container">
        <p class="hero-kicker">{{ heroKicker }}</p>
        <h1 class="hero-title">{{ site.settings.siteName || 'MyBlog' }}</h1>
        <p class="hero-desc">
          {{ site.settings.siteDescription || '一个关于 Go、Vue 与工程实践的长期笔记站点。' }}
        </p>
        <div class="hero-actions">
          <a class="btn btn-primary" href="#latest">阅读最新文章</a>
          <a class="btn btn-hero-ghost" href="/rss" target="_blank" rel="noopener noreferrer">
            订阅 RSS
          </a>
        </div>
        <p class="hero-stats">
          <span>{{ total }} 篇文章</span>
          <span class="hero-sep">·</span>
          <span>{{ site.categories.length }} 个分类</span>
          <span class="hero-sep">·</span>
          <span>{{ site.tags.length }} 个标签</span>
          <span class="hero-sep">·</span>
          <span>持续更新中</span>
        </p>
      </div>
    </section>

    <div class="container page-layout">
      <section class="page-main">
        <header class="list-head">
          <h2 class="list-title">
            最新文章
            <span class="list-count">共 {{ total }} 篇</span>
          </h2>
          <RouterLink to="/archives" class="list-more">查看归档 →</RouterLink>
        </header>

        <ListSkeleton v-if="loading" />
        <ErrorState v-else-if="error" :message="error" @retry="reload" />
        <template v-else>
          <div v-if="items.length" id="latest" class="post-list">
            <div
              v-for="(post, index) in items"
              :key="post.id"
              v-reveal="{ delay: Math.min(index * 45, 225) }"
            >
              <ArticleCard :post="post" :tint-index="index" />
            </div>
          </div>
          <EmptyState
            v-else
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

const route = useRoute()
const router = useRouter()
const site = useSiteStore()

const list = usePagedList<PostSummary>(
  ({ page, pageSize, signal }) => fetchPosts({ page, pageSize }, signal),
  site.settings.postPageSize || 10
)
const { loading, error, items, total, page } = list

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / list.pageSize.value)))

/** Hero kicker：站点关键词优先，兜底为固定 slogan */
const heroKicker = computed(() => {
  const kw = site.settings.siteKeywords?.trim()
  return kw ? `PERSONAL BLOG · ${kw}` : 'PERSONAL BLOG · GO / VUE / 工程实践'
})

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
.home {
  padding-bottom: 8px;
}

/* ---------- Hero：通栏深藏蓝（p02） ---------- */
.hero {
  position: relative;
  overflow: hidden;
  background: var(--brand-navy);
  color: #fff;
}

/* 装饰几何：粉彩圆点 + 方块 + 一条低对比弧线 */
.hero-deco {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.deco {
  position: absolute;
  border-radius: 50%;
}

.deco-pink {
  top: 32%;
  right: 16%;
  width: 16px;
  height: 16px;
  background: #f06bb0;
}

.deco-yellow {
  top: 66%;
  left: 14%;
  width: 14px;
  height: 14px;
  background: #f0c85a;
}

.deco-orange {
  top: 18%;
  right: 26%;
  width: 7px;
  height: 7px;
  background: #f0995a;
}

.deco-cyan {
  top: 70%;
  right: 8%;
  width: 11px;
  height: 11px;
  background: #4ecfc4;
}

.deco-white {
  top: 44%;
  left: 27%;
  width: 5px;
  height: 5px;
  background: rgba(255, 255, 255, 0.85);
}

.deco-square {
  position: absolute;
  border-radius: 3px;
}

.deco-square-1 {
  bottom: 30%;
  left: 8%;
  width: 17px;
  height: 17px;
  background: #f0c85a;
  transform: rotate(8deg);
}

.deco-square-2 {
  top: 26%;
  right: 8%;
  width: 13px;
  height: 13px;
  border-radius: 50%;
  background: #7c6ce0;
}

.deco-orbit {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.hero-inner {
  position: relative;
  padding-top: 76px;
  padding-bottom: 56px;
  text-align: center;
}

/* mono 大写 kicker：Hero 顶部弱信息 */
.hero-kicker {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-weight: 500;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.62);
}

.hero-title {
  margin-top: 22px;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: var(--fs-display);
  line-height: var(--lh-display);
  letter-spacing: -0.02em;
  color: #fff;
  overflow-wrap: anywhere;
}

.hero-desc {
  margin: 16px auto 0;
  max-width: 620px;
  font-size: var(--fs-base);
  line-height: 1.75;
  color: rgba(255, 255, 255, 0.72);
  overflow-wrap: anywhere;
}

.hero-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-top: 30px;
}

/* 深色背景上的描边按钮：白描边 + 白字 */
.btn-hero-ghost {
  border-color: rgba(255, 255, 255, 0.42);
  background: transparent;
  color: #fff;
}

.btn-hero-ghost:hover {
  border-color: rgba(255, 255, 255, 0.9);
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

/* 分隔线 + mono 统计行 */
.hero-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 34px;
  padding-top: 22px;
  border-top: 1px solid rgba(255, 255, 255, 0.14);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  letter-spacing: 0.06em;
  color: rgba(255, 255, 255, 0.6);
  font-variant-numeric: tabular-nums;
}

.hero-sep {
  color: rgba(255, 255, 255, 0.28);
}

/* ---------- 正文列表 ---------- */
/* 栏头：左「最新文章 · 共 N 篇」，右上「查看归档 →」 */
.list-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.list-title {
  display: flex;
  align-items: baseline;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: var(--fs-xl);
  line-height: var(--lh-heading-2);
  letter-spacing: -0.01em;
}

.list-count {
  font-family: var(--font-sans);
  font-size: var(--fs-sm);
  font-weight: 400;
  color: var(--text-3);
}

.list-more {
  flex-shrink: 0;
  font-size: var(--fs-sm);
  color: var(--text-2);
  transition: color var(--transition);
}

.list-more:hover {
  color: var(--brand);
}

.post-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 20px;
}

@media (max-width: 900px) {
  .hero-inner {
    padding-top: 56px;
    padding-bottom: 44px;
  }

  .hero-stats {
    gap: 8px;
  }
}

@media (max-width: 560px) {
  .hero-inner.container {
    padding-left: 24px;
    padding-right: 24px;
  }

  .hero-kicker {
    font-size: 11px;
    letter-spacing: 0.12em;
    line-height: 1.7;
  }

  .hero-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .hero-actions .btn {
    width: 100%;
  }

  .hero-stats {
    margin-top: 28px;
    padding-top: 18px;
    font-size: 11px;
    gap: 6px;
  }
}
</style>
