<template>
  <div class="container archives-page">
    <header class="page-title-bar">
      <h1 class="page-heading">归档</h1>
      <p v-if="totalCount > 0" class="page-sub">共 {{ totalCount }} 篇文章，时间是最忠实的读者。</p>
    </header>

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
          {{ group.year }}
          <span class="year-count">{{ group.items.length }} 篇</span>
        </h2>
        <ul class="timeline">
          <li v-for="item in group.items" :key="item.id" class="timeline-item">
            <span class="tl-date">{{ formatMonthDay(item.createdAt) }}</span>
            <RouterLink :to="`/post/${item.slug}`" class="tl-title">{{ item.title }}</RouterLink>
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
import { formatMonthDay } from '@/utils/format'
import type { ArchiveYear } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

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
})
</script>

<style scoped>
.archives-page {
  padding-bottom: 56px;
}

.year-group {
  margin-top: 26px;
}

.year-title {
  font-family: var(--font-mono);
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.year-count {
  margin-left: 10px;
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
}

/* 桌面端年份粘性：随分组滚动吸附在报头下，渐变遮罩避免与时间轴硬叠（docs/06 §8.1） */
@media (min-width: 768px) {
  .year-group {
    margin-top: 34px;
  }

  .year-title {
    position: sticky;
    top: calc(var(--header-height) + 10px);
    z-index: 5;
    display: flex;
    align-items: baseline;
    margin: -10px -12px 10px;
    padding: 10px 12px 14px;
    background: linear-gradient(to bottom, var(--bg) 72%, transparent);
  }
}

.timeline {
  list-style: none;
  margin: 14px 0 0;
  padding: 4px 0 4px 22px;
  border-left: 1px solid var(--border);
}

.timeline-item {
  position: relative;
  display: flex;
  align-items: baseline;
  gap: 16px;
  padding: 9px 0;
}

.timeline-item::before {
  content: '';
  position: absolute;
  left: -26px;
  top: 50%;
  transform: translateY(-50%);
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--border-strong);
  transition: background var(--transition), transform var(--transition);
}

.timeline-item:hover::before {
  background: var(--brand);
  transform: translateY(-50%) scale(1.25);
}

.tl-date {
  flex-shrink: 0;
  width: 48px;
  font-family: var(--font-mono);
  font-size: 12.5px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.tl-title {
  color: var(--text-1);
  font-size: 14.5px;
  line-height: 1.6;
}

.tl-title:hover {
  color: var(--brand);
}

/* 行 hover：日期与圆点同步走 accent（docs/06 §8.2） */
.timeline-item:hover .tl-date {
  color: var(--brand);
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
</style>
