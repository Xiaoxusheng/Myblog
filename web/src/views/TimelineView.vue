<template>
  <div class="container timeline-page">
    <header class="page-title-bar">
      <h1 class="page-heading">时间线</h1>
      <p v-if="events.length" class="page-sub">共 {{ events.length }} 个节点</p>
    </header>

    <div v-if="loading" class="tl-skeleton" aria-hidden="true">
      <span v-for="i in 4" :key="i" class="skeleton sk-tl"></span>
    </div>

    <ErrorState v-else-if="error" :message="error" @retry="load" />

    <div v-else-if="groups.length" class="tl-body">
      <section v-for="group in groups" :key="group.year" class="tl-year-group">
        <h2 class="tl-year">{{ group.year }}</h2>
        <ol class="tl-list">
          <li v-for="item in group.items" :key="item.id" v-reveal class="tl-node">
            <span class="tl-dot" aria-hidden="true"></span>
            <div class="tl-card">
              <div class="tl-head">
                <time class="tl-date" :datetime="item.eventDate">{{ formatMonthDay(item.eventDate) }}</time>
                <h3 class="tl-title">{{ item.title }}</h3>
              </div>
              <div v-if="item.content" class="tl-content markdown-body" v-html="item.html"></div>
              <img
                v-if="item.image"
                class="tl-image"
                :src="item.image"
                :alt="item.title"
                loading="lazy"
                decoding="async"
              />
              <div class="tl-links">
                <RouterLink v-if="item.post" :to="`/post/${item.post.slug}`" class="tl-link">
                  文章：{{ item.post.title }}<span class="tl-link-arrow">→</span>
                </RouterLink>
                <a
                  v-if="item.projectUrl"
                  class="tl-link ext-link"
                  :href="item.projectUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {{ item.projectName || '项目' }}<span class="tl-link-arrow">↗</span>
                </a>
              </div>
            </div>
          </li>
        </ol>
      </section>
    </div>

    <EmptyState v-else title="暂无时间线" description="博主还没有记录任何节点。" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchTimeline } from '@/api/content'
import { ApiError, isRequestCanceled } from '@/api/http'
import { renderMarkdown } from '@/utils/markdown'
import { formatMonthDay } from '@/utils/format'
import type { TimelineEventItem } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

interface TimelineNode extends TimelineEventItem {
  html: string
}

const events = ref<TimelineNode[]>([])
const loading = ref(true)
const error = ref('')
let controller: AbortController | null = null

/** 按年分组（数据本身已按时间倒序） */
const groups = computed(() => {
  const out: Array<{ year: number; items: TimelineNode[] }> = []
  for (const ev of events.value) {
    const year = new Date(ev.eventDate).getFullYear()
    const last = out[out.length - 1]
    if (last && last.year === year) {
      last.items.push(ev)
    } else {
      out.push({ year, items: [ev] })
    }
  }
  return out
})

async function load(): Promise<void> {
  controller?.abort()
  const localController = new AbortController()
  controller = localController
  loading.value = true
  error.value = ''
  try {
    const list = await fetchTimeline(localController.signal)
    if (controller !== localController) return
    events.value = list.map((ev) => ({ ...ev, html: ev.content ? renderMarkdown(ev.content).html : '' }))
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
.timeline-page {
  padding-bottom: 56px;
}

.tl-skeleton {
  margin-top: 22px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.sk-tl {
  display: block;
  height: 96px;
}

.tl-body {
  margin-top: 22px;
}

.tl-year {
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.14em;
  color: var(--text-3);
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border);
}

.tl-list {
  list-style: none;
  margin: 0;
  padding: 6px 0 8px 20px;
  /* 竖向 hairline：贯穿节点圆点 */
  border-left: 1px solid var(--border);
}

.tl-node {
  position: relative;
  padding: 18px 0 6px;
}

.tl-dot {
  position: absolute;
  left: -24.5px;
  top: 27px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--surface);
  border: 1.5px solid var(--brand);
}

.tl-head {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.tl-date {
  flex-shrink: 0;
  font-family: var(--font-mono);
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
  color: var(--brand);
}

.tl-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-1);
  line-height: 1.5;
}

.tl-content {
  margin-top: 6px;
  max-width: var(--reading-width);
}

.tl-content :deep(p) {
  margin: 0.4em 0;
}

.tl-image {
  display: block;
  max-width: min(560px, 100%);
  margin-top: 10px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
}

.tl-links {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 18px;
  margin-top: 8px;
}

.tl-link {
  font-size: 13px;
  color: var(--text-2);
  transition: color var(--transition);
}

.tl-link:hover {
  color: var(--brand);
}

.tl-link-arrow {
  display: inline-block;
  margin-left: 3px;
  transition: transform var(--transition);
}

.tl-link:hover .tl-link-arrow {
  transform: translateX(2px);
}

@media (max-width: 640px) {
  .tl-list {
    padding-left: 16px;
  }

  .tl-dot {
    left: -20.5px;
  }

  .tl-head {
    flex-direction: column;
    gap: 2px;
  }
}
</style>
