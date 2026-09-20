<template>
  <div class="container changelog-page">
    <header class="page-title-bar">
      <h1 class="page-heading">更新日志</h1>
      <p v-if="items.length" class="page-sub">共 {{ items.length }} 个版本</p>
    </header>

    <div v-if="loading" class="cl-skeleton" aria-hidden="true">
      <span v-for="i in 3" :key="i" class="skeleton sk-cl"></span>
    </div>

    <ErrorState v-else-if="error" :message="error" @retry="load" />

    <div v-else-if="items.length" class="cl-list">
      <article v-for="item in items" :key="item.id" v-reveal class="cl-item">
        <header class="cl-head">
          <div class="cl-version-row">
            <span class="cl-version">v{{ item.version }}</span>
            <h2 v-if="item.title" class="cl-title">{{ item.title }}</h2>
          </div>
          <time class="cl-date" :datetime="item.releasedAt">{{ formatDate(item.releasedAt) }}</time>
        </header>
        <div class="cl-content markdown-body" v-html="item.html"></div>
      </article>
    </div>

    <EmptyState v-else title="暂无版本记录" description="项目还没有发布过任何版本。" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchChangelog } from '@/api/content'
import { ApiError, isRequestCanceled } from '@/api/http'
import { renderMarkdown } from '@/utils/markdown'
import { formatDate } from '@/utils/format'
import type { ChangelogItem } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

interface ChangelogNode extends ChangelogItem {
  html: string
}

const items = ref<ChangelogNode[]>([])
const loading = ref(true)
const error = ref('')
let controller: AbortController | null = null

async function load(): Promise<void> {
  controller?.abort()
  const localController = new AbortController()
  controller = localController
  loading.value = true
  error.value = ''
  try {
    const list = await fetchChangelog(localController.signal)
    if (controller !== localController) return
    items.value = list.map((it) => ({ ...it, html: it.content ? renderMarkdown(it.content).html : '' }))
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
.changelog-page {
  padding-bottom: 56px;
}

.cl-skeleton {
  margin-top: 22px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.sk-cl {
  display: block;
  height: 120px;
}

.cl-list {
  margin-top: 22px;
  display: flex;
  flex-direction: column;
}

.cl-item {
  padding: 26px 0 22px;
  border-bottom: 1px solid var(--border);
}

.cl-item:last-child {
  border-bottom: none;
}

.cl-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 6px 16px;
}

.cl-version-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  min-width: 0;
}

.cl-version {
  flex-shrink: 0;
  padding: 2px 10px;
  border: 1px solid var(--brand-soft-border);
  border-radius: 999px;
  background: var(--brand-soft);
  color: var(--brand);
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.cl-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-1);
  line-height: 1.5;
}

.cl-date {
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
}

.cl-content {
  margin-top: 12px;
  max-width: var(--reading-width);
}
</style>
