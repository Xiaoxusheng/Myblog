<template>
  <div class="container archives-page">
    <header class="page-title-bar">
      <p class="kicker">ARCHIVE · 全部文章</p>
      <h1 class="page-heading">归档</h1>
      <p v-if="totalCount > 0" class="page-sub">
        按时间倒序排列，共 {{ totalCount }} 篇文章，跨越 {{ years.length }} 年。
      </p>
    </header>

    <!-- 统计卡：时间跨度 / 文章总数 / 分类数量（p05） -->
    <div v-if="!loading && !error && totalCount > 0" class="archive-stats">
      <div class="stat-card">
        <p class="stat-value">{{ years.length }}</p>
        <p class="stat-label">时间跨度（年）</p>
      </div>
      <div class="stat-card">
        <p class="stat-value">{{ totalCount }}</p>
        <p class="stat-label">文章总数</p>
      </div>
      <div class="stat-card">
        <p class="stat-value">{{ site.categories.length }}</p>
        <p class="stat-label">分类数量</p>
      </div>
    </div>

    <div v-if="loading" class="archive-skeleton" aria-hidden="true">
      <template v-for="i in 2" :key="i">
        <span class="skeleton sk-year"></span>
        <div v-for="j in 4" :key="j" class="skeleton sk-row" :style="{ width: j % 2 ? '62%' : '48%' }"></div>
      </template>
    </div>

    <ErrorState v-else-if="error" :message="error" @retry="load" />

    <template v-else>
      <section v-for="group in years" :key="group.year" v-reveal class="year-group">
        <h2 class="year-title">
          <span class="year-num">{{ group.year }}</span>
          <span class="year-rule"></span>
          <span class="year-count">{{ group.items.length }} 篇</span>
        </h2>
        <ul class="archive-list">
          <li v-for="item in group.items" :key="item.id" class="archive-item">
            <RouterLink :to="`/post/${item.slug}`" class="archive-link">
              <span class="ai-date">{{ formatMonthDay(item.createdAt) }}</span>
              <span class="ai-title">{{ item.title }}</span>
              <span class="ai-arrow" aria-hidden="true">›</span>
            </RouterLink>
          </li>
        </ul>
      </section>

      <EmptyState
        v-if="years.length === 0"
        title="暂无归档"
        description="还没有发布任何文章。"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchArchive } from '@/api/content'
import { ApiError, isRequestCanceled } from '@/api/http'
import { useSiteStore } from '@/stores/site'
import { formatMonthDay } from '@/utils/format'
import type { ArchiveYear } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

const site = useSiteStore()
const years = ref<ArchiveYear[]>([])
const loading = ref(true)
const error = ref('')
let controller: AbortController | null = null

const totalCount = computed(() =>
  years.value.reduce((sum, group) => sum + group.items.length, 0)
)

async function load(): Promise<void> {
  controller?.abort()
  const localController = new AbortController()
  controller = localController
  loading.value = true
  error.value = ''
  try {
    const data = await fetchArchive(localController.signal)
    if (controller !== localController) return
    years.value = data ?? []
  } catch (e) {
    if (controller !== localController || isRequestCanceled(e)) return
    error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
  } finally {
    if (controller === localController) loading.value = false
  }
}

onMounted(() => {
  void load()
  void site.ensureLoaded()
})
</script>

<style scoped>
.archives-page {
  padding-bottom: 64px;
}

/* ---------- 统计卡（p05） ---------- */
.archive-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 28px;
}

.stat-card {
  padding: 20px 22px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
}

.stat-value {
  font-family: var(--font-display);
  font-size: var(--fs-2xl);
  font-weight: var(--display-weight);
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}

.stat-label {
  margin-top: 4px;
  font-size: var(--fs-sm);
  color: var(--text-3);
}

/* ---------- 年份分组：年份 + 长横线 + 篇数（p05） ---------- */
.year-group {
  margin-top: 36px;
}

.year-title {
  display: flex;
  align-items: center;
  gap: 14px;
}

.year-num {
  font-family: var(--font-mono);
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.year-rule {
  flex: 1;
  height: 1px;
  background: var(--border);
}

.year-count {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

/* 桌面端年份粘性：随分组滚动吸附在报头下，渐变遮罩避免与列表硬叠 */
@media (min-width: 768px) {
  .year-title {
    position: sticky;
    top: calc(var(--header-height) + 10px);
    z-index: 5;
    margin: -10px -12px 4px;
    padding: 12px 12px 14px;
    background: linear-gradient(to bottom, var(--bg) 74%, transparent);
  }
}

/* ---------- 归档行：轻卡（p05） ---------- */
.archive-list {
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.archive-link {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
  transition: border-color var(--transition), background var(--transition);
}

.archive-link:hover {
  border-color: var(--border-strong);
  background: var(--surface);
}

.ai-date {
  flex-shrink: 0;
  width: 44px;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
  transition: color var(--transition);
}

.archive-link:hover .ai-date {
  color: var(--brand);
}

.ai-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--fs-base);
  color: var(--text-1);
  transition: color var(--transition);
}

.archive-link:hover .ai-title {
  color: var(--brand);
}

.ai-arrow {
  flex-shrink: 0;
  color: var(--text-3);
  font-size: 18px;
  line-height: 1;
  transition: color var(--transition), transform var(--transition);
}

.archive-link:hover .ai-arrow {
  color: var(--brand);
  transform: translateX(2px);
}

.archive-skeleton {
  margin-top: 28px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.sk-year {
  width: 90px;
  height: 26px;
  border-radius: 8px;
  margin-top: 12px;
}

.sk-row {
  height: 15px;
  margin-left: 22px;
}

@media (max-width: 640px) {
  .archive-stats {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .stat-card {
    padding: 16px 18px;
  }
}
</style>

