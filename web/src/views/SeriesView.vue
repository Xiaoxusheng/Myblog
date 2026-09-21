<template>
  <div class="container series-page">
    <header class="page-title-bar">
      <p class="kicker">SERIES · 成体系的连载</p>
      <h1 class="page-heading">专题</h1>
      <p class="page-sub">把零散的文章串成一条线，按顺序读会更顺。</p>
    </header>

    <ErrorState v-if="error" :message="error" @retry="load" />

    <div v-else-if="loading" class="series-skeleton" aria-hidden="true">
      <span v-for="i in 3" :key="i" class="skeleton sk-series"></span>
    </div>

    <template v-else-if="list.length">
      <h2 class="list-head">
        全部专题
        <span class="list-count">{{ list.length }} 个专题</span>
      </h2>

      <div class="series-list">
        <RouterLink
          v-for="(item, index) in list"
          :key="item.id"
          :to="`/series/${item.slug}`"
          class="series-card"
        >
          <span
            class="series-cover"
            :class="`tint-${TINTS[index % TINTS.length]}`"
          >
            <img v-if="item.cover" :src="item.cover" :alt="item.name" loading="lazy" />
            <span v-else class="cover-initial" aria-hidden="true">{{ initial(item.name) }}</span>
          </span>

          <div class="series-info">
            <div class="series-head">
              <h3 class="series-name">{{ item.name }}</h3>
            </div>
            <p class="series-desc" :class="{ placeholder: !item.description }">
              {{ item.description || '查看该专题下的全部文章' }}
            </p>
            <p v-if="typeof item.postCount === 'number'" class="series-meta">
              {{ item.postCount }} 篇文章
            </p>
          </div>

          <span class="series-arrow" aria-hidden="true">›</span>
        </RouterLink>
      </div>

      <!-- 底部 CTA：订阅引导（p07） -->
      <section class="cta-block">
        <div>
          <h2 class="cta-title">不想错过更新？</h2>
          <p class="cta-desc">订阅 RSS，或收藏本站，新文章发布后第一时间看到。</p>
        </div>
        <a class="btn btn-primary" href="/rss" target="_blank" rel="noopener noreferrer">订阅 RSS</a>
      </section>
    </template>

    <EmptyState v-else title="暂无专题" description="博主还没有创建任何专题。" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchSeriesList } from '@/api/series'
import { ApiError } from '@/api/http'
import { setSeo } from '@/utils/seo'
import type { Series } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

const TINTS = ['sky', 'mint', 'peach', 'lavender'] as const

const list = ref<Series[]>([])
const loading = ref(true)
const error = ref('')

/** 无封面时取名称首字符做占位 */
function initial(name: string): string {
  return name.trim().slice(0, 1).toUpperCase() || '#'
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    list.value = await fetchSeriesList()
    setSeo({ description: '博客专题连载合集' })
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<style scoped>
.series-page {
  padding-bottom: 64px;
}

.list-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-top: 28px;
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

/* ---------- 专题行卡片（p07） ---------- */
.series-list {
  margin-top: 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.series-card {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg);
  transition: border-color var(--transition), background var(--transition);
}

.series-card:hover {
  border-color: var(--border-strong);
  background: var(--surface);
}

.series-cover {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--surface);
}

.series-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cover-initial {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-2);
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

.series-info {
  flex: 1;
  min-width: 0;
}

.series-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.series-name {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--text-1);
  transition: color var(--transition);
}

.series-card:hover .series-name {
  color: var(--brand);
}

.series-desc {
  margin-top: 6px;
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.7;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.series-desc.placeholder {
  color: var(--text-3);
}

.series-meta {
  margin-top: 8px;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.series-arrow {
  flex-shrink: 0;
  color: var(--text-3);
  font-size: 20px;
  line-height: 1;
  transition: color var(--transition), transform var(--transition);
}

.series-card:hover .series-arrow {
  color: var(--brand);
  transform: translateX(2px);
}

/* ---------- 底部 CTA（p07） ---------- */
.cta-block {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-top: 32px;
  padding: 26px 28px;
  border-radius: var(--radius-lg);
  background: var(--surface);
}

.cta-title {
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: var(--fs-lg);
  line-height: var(--lh-heading-2);
  letter-spacing: -0.01em;
}

.cta-desc {
  margin-top: 6px;
  font-size: var(--fs-sm);
  color: var(--text-2);
}

.series-skeleton {
  margin-top: 22px;
  display: flex;
  flex-direction: column;
}

.sk-series {
  display: block;
  height: 100px;
  margin-bottom: 12px;
  border-radius: var(--radius-lg);
}

@media (max-width: 640px) {
  .cta-block {
    flex-direction: column;
    align-items: flex-start;
    padding: 20px;
  }

  .cta-block .btn {
    width: 100%;
  }
}
</style>
