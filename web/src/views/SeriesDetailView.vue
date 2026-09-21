<template>
  <div class="container page-layout">
    <section class="page-main">
      <ListSkeleton v-if="loading" />

      <ErrorState v-else-if="error" :message="error" @retry="load" />

      <EmptyState v-else-if="notFound" title="专题不存在" description="它可能已被删除，或者链接有误。">
        <RouterLink class="btn" to="/series">浏览全部专题</RouterLink>
      </EmptyState>

      <template v-else-if="series">
        <RouterLink to="/series" class="back-link">← 全部专题</RouterLink>

        <header class="series-head">
          <span class="series-mark" :class="`tint-${tintName}`" aria-hidden="true">
            <img v-if="series.cover" :src="series.cover" :alt="series.name" />
            <span v-else>{{ initial(series.name) }}</span>
          </span>
          <div class="series-head-main">
            <p class="kicker">SERIES · 连载中</p>
            <h1 class="page-heading">{{ series.name }}</h1>
            <p v-if="series.description" class="series-desc">{{ series.description }}</p>
            <p class="series-meta">
              共 {{ series.postCount ?? posts.length }} 篇
              <span class="dot">·</span>
              已发布 {{ posts.length }} 篇
            </p>
          </div>
        </header>

        <!-- 阅读进度：已发布 / 总数（p08 紫色进度条） -->
        <div
          class="progress-track"
          role="progressbar"
          :aria-valuenow="progressPercent"
          aria-valuemin="0"
          aria-valuemax="100"
        >
          <span class="progress-fill" :style="{ width: `${progressPercent}%` }"></span>
        </div>

        <div v-if="posts.length" class="series-posts">
          <RouterLink
            v-for="(post, index) in posts"
            :key="post.id"
            :to="`/post/${post.slug}`"
            class="sp-row"
          >
            <span class="sp-num">{{ pad(index + 1) }}</span>
            <span class="sp-main">
              <span class="sp-title">{{ post.title }}</span>
              <span class="sp-meta">
                {{ formatDate(post.publishedAt || post.createdAt) }}
                <span class="dot">·</span>
                阅读 {{ formatNumber(post.viewCount) }}
              </span>
            </span>
            <span class="sp-arrow" aria-hidden="true">›</span>
          </RouterLink>
        </div>
        <EmptyState v-else title="该专题下还没有已发布的文章" description="发布后就会按序出现在这里。" />

        <p v-if="remaining > 0" class="series-note">
          还剩 {{ remaining }} 篇在写，订阅 RSS，新的第一篇发布后会自动推给你。
        </p>
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
import { formatDate, formatNumber } from '@/utils/format'
import type { PostSummary, Series } from '@/types'
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

const TINTS = ['sky', 'mint', 'peach', 'lavender'] as const
const tintName = computed(() => TINTS[(series.value?.id ?? 0) % TINTS.length])

/** 专题内已发布文章数 / 总数：Total 未知时退化为 100% */
const progressPercent = computed(() => {
  const total = series.value?.postCount ?? posts.value.length
  if (!total) return 0
  return Math.min(100, Math.round((posts.value.length / total) * 100))
})

const remaining = computed(() => {
  const total = series.value?.postCount ?? posts.value.length
  return Math.max(0, total - posts.value.length)
})

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

function initial(name: string): string {
  return name.trim().slice(0, 1).toUpperCase() || '#'
}

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

.back-link {
  display: inline-block;
  margin-bottom: 18px;
  font-size: var(--fs-sm);
  color: var(--text-2);
  transition: color var(--transition);
}

.back-link:hover {
  color: var(--brand);
}

/* ---------- 专题头部（p08） ---------- */
.series-head {
  display: flex;
  gap: 18px;
  align-items: flex-start;
  margin-bottom: 20px;
}

.series-mark {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: var(--radius-md);
  overflow: hidden;
  font-size: 22px;
  font-weight: 700;
  color: var(--text-2);
}

.series-mark img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.tint-sky {
  background: var(--tint-sky);
}

.tint-mint {
  background: var(--tint-mint);
}

.tint-peach {
  background: var(--tint-peach);
}

.tint-lavender {
  background: var(--tint-lavender);
}

.series-head-main {
  min-width: 0;
}

.series-head .kicker {
  margin-bottom: 8px;
  color: var(--brand);
}

.series-desc {
  margin-top: 8px;
  font-size: var(--fs-base);
  color: var(--text-2);
  line-height: 1.75;
  max-width: var(--reading-width);
}

.series-meta {
  margin-top: 10px;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.dot {
  margin: 0 6px;
  color: var(--border-strong);
}

/* ---------- 进度条（p08） ---------- */
.progress-track {
  height: 6px;
  border-radius: var(--radius-pill);
  background: var(--surface);
  overflow: hidden;
}

.progress-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--brand);
  transition: width 0.5s var(--ease-out-quart);
}

/* ---------- 编号列表 ---------- */
.series-posts {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 22px;
}

.sp-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
  transition: border-color var(--transition), background var(--transition);
}

.sp-row:hover {
  border-color: var(--border-strong);
  background: var(--surface);
}

.sp-num {
  flex-shrink: 0;
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
  font-weight: 600;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
  transition: color var(--transition);
}

.sp-row:hover .sp-num {
  color: var(--brand);
}

.sp-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sp-title {
  font-size: var(--fs-base);
  color: var(--text-1);
  transition: color var(--transition);
}

.sp-row:hover .sp-title {
  color: var(--brand);
}

.sp-meta {
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.sp-arrow {
  flex-shrink: 0;
  color: var(--text-3);
  font-size: 18px;
  line-height: 1;
  transition: color var(--transition), transform var(--transition);
}

.sp-row:hover .sp-arrow {
  color: var(--brand);
  transform: translateX(2px);
}

.series-note {
  margin-top: 18px;
  padding: 14px 16px;
  border-radius: var(--radius-md);
  background: var(--surface);
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.7;
}

@media (max-width: 640px) {
  .series-head {
    flex-direction: column;
    gap: 12px;
  }
}
</style>
