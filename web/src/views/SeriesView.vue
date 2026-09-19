<template>
  <div class="container series-page">
    <header class="page-title-bar">
      <h1 class="page-heading">专题</h1>
      <p v-if="!loading && !error && list.length" class="page-sub">
        共 {{ list.length }} 个专题
      </p>
    </header>

    <ErrorState v-if="error" :message="error" @retry="load" />

    <div v-else-if="loading" class="series-skeleton" aria-hidden="true">
      <span v-for="i in 3" :key="i" class="skeleton sk-series card"></span>
    </div>

    <div v-else-if="list.length" class="series-grid">
      <RouterLink
        v-for="item in list"
        :key="item.id"
        :to="`/series/${item.slug}`"
        class="series-card card"
      >
        <span v-if="item.cover" class="series-cover">
          <img :src="item.cover" :alt="item.name" loading="lazy" />
        </span>
        <span v-else class="series-cover series-cover--fallback" aria-hidden="true">
          {{ initial(item.name) }}
        </span>
        <div class="series-info">
          <div class="series-head">
            <h2 class="series-name">{{ item.name }}</h2>
            <span v-if="typeof item.postCount === 'number'" class="series-count">
              {{ item.postCount }} 篇
            </span>
          </div>
          <p class="series-desc" :class="{ placeholder: !item.description }">
            {{ item.description || '查看该专题下的全部文章' }}
          </p>
        </div>
      </RouterLink>
    </div>

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
  padding-bottom: 56px;
}

.series-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
  margin-top: 22px;
}

@media (min-width: 700px) {
  .series-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1000px) {
  .series-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

.series-card {
  display: flex;
  gap: 14px;
  padding: 18px 20px;
  transition: border-color var(--transition), box-shadow 0.2s ease-out, transform 0.2s ease-out;
}

.series-card:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.series-cover {
  flex-shrink: 0;
  display: block;
  width: 56px;
  height: 56px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--surface-2);
}

.series-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* 无封面：名称首字符占位，保持克制的单色块 */
.series-cover--fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  font-weight: 600;
  color: var(--text-3);
}

.series-info {
  flex: 1;
  min-width: 0;
}

.series-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.series-name {
  font-size: 15.5px;
  font-weight: 600;
  color: var(--text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color var(--transition);
}

.series-card:hover .series-name {
  color: var(--brand);
}

.series-count {
  flex-shrink: 0;
  font-size: 12.5px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.series-desc {
  margin-top: 6px;
  font-size: 13px;
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

.series-skeleton {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 22px;
}

@media (max-width: 999px) {
  .series-skeleton {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 699px) {
  .series-skeleton {
    grid-template-columns: 1fr;
  }
}

.sk-series {
  display: block;
  height: 92px;
}
</style>
