<template>
  <aside class="page-aside sidebar">
    <section v-if="site.settings.notice" v-reveal class="side-card side-card--notice">
      <h3 class="side-title">NOTICE</h3>
      <p class="notice">{{ site.settings.notice }}</p>
    </section>

    <section v-reveal="{ delay: 60 }" class="side-card">
      <h3 class="side-title">热门文章</h3>
      <div v-if="hotLoading" class="hot-skeletons" aria-hidden="true">
        <div v-for="i in 5" :key="i" class="hot-skeleton">
          <span class="skeleton sk-line"></span>
          <span class="skeleton sk-line sk-short"></span>
        </div>
      </div>
      <ErrorState v-else-if="hotError" small :message="hotError" @retry="load" />
      <ol v-else-if="hotPosts.length" class="hot-list">
        <li v-for="(post, index) in hotPosts" :key="post.id" class="hot-row">
          <RouterLink :to="`/post/${post.slug}`" class="hot-item">
            <span class="hot-rank">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="hot-title">{{ post.title }}</span>
          </RouterLink>
        </li>
      </ol>
      <EmptyState v-else size="compact" title="暂无热门文章" />
    </section>

    <section v-if="cloudTags.length" v-reveal="{ delay: 90 }" class="side-card">
      <h3 class="side-title">标签云</h3>
      <div class="tag-cloud">
        <RouterLink
          v-for="tag in cloudTags"
          :key="tag.id"
          :to="`/tag/${tag.slug}`"
          class="chip"
        >
          {{ tag.name }}
          <span v-if="tag.postCount" class="tag-count">{{ tag.postCount }}</span>
        </RouterLink>
      </div>
      <RouterLink v-if="site.tags.length > 24" to="/tags" class="side-more">更多标签 →</RouterLink>
    </section>

    <section v-if="archiveYears.length" v-reveal="{ delay: 120 }" class="side-card">
      <h3 class="side-title">归档</h3>
      <ul class="archive-list">
        <li v-for="item in archiveYears" :key="item.year" class="archive-row">
          <RouterLink :to="{ path: '/archives', query: { year: item.year } }" class="archive-item">
            <span class="archive-year">{{ item.year }}年</span>
            <span class="archive-count">{{ item.count }} 篇</span>
          </RouterLink>
        </li>
      </ul>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useHotPosts } from '@/composables/useHotPosts'
import { useSiteStore } from '@/stores/site'
import { fetchArchive } from '@/api/content'
import ErrorState from '@/components/common/ErrorState.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const site = useSiteStore()
const { hotPosts, hotLoading, hotError, load } = useHotPosts()

const cloudTags = computed(() => site.tags.slice(0, 24))

/** 归档年份 + 篇数：取最近 4 个年份 */
const archiveYears = ref<{ year: number; count: number }[]>([])

onMounted(() => {
  void load()
  // 侧栏归档加载失败静默隐藏，不影响主体内容
  const controller = new AbortController()
  fetchArchive(controller.signal)
    .then((years) => {
      archiveYears.value = years.slice(0, 4).map((y) => ({ year: y.year, count: y.items.length }))
    })
    .catch(() => {})
})
</script>

<style scoped>
/* 侧栏：四张独立卡片，底部同色发丝描边（p02） */
.sidebar {
  gap: 16px;
}

.side-card {
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg);
}

/* 公告卡：浅黄底（p02 NOTICE） */
.side-card--notice {
  background: var(--notice-bg);
  border-color: transparent;
}

.side-title {
  margin: 0 0 14px;
  font-size: var(--fs-sm);
  font-weight: 600;
  color: var(--text-1);
}

.side-card--notice .side-title {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-2);
}

.notice {
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.75;
  white-space: pre-wrap;
}

/* 热门文章：mono 编号榜单 */
.hot-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.hot-row + .hot-row {
  margin-top: 10px;
}

.hot-item {
  display: flex;
  align-items: baseline;
  gap: 9px;
}

.hot-rank {
  flex-shrink: 0;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
}

.hot-title {
  flex: 1;
  min-width: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: var(--fs-sm);
  color: var(--text-1);
  line-height: 1.55;
  transition: color var(--transition);
}

.hot-item:hover .hot-title {
  color: var(--brand);
}

.hot-skeletons {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 2px 0;
}

.hot-skeleton {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sk-line {
  display: block;
  height: 12px;
  width: 88%;
}

.sk-short {
  width: 55%;
}

/* 标签云：浅底胶囊 chip + mono 计数 */
.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.tag-count {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
}

a.chip:hover .tag-count {
  color: var(--text-2);
}

/* 归档：年份 + mono 篇数 */
.archive-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.archive-row + .archive-row {
  margin-top: 2px;
}

.archive-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 5px 0;
  font-size: var(--fs-sm);
  color: var(--text-2);
  transition: color var(--transition);
}

.archive-item:hover {
  color: var(--brand);
}

.archive-count {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.side-more {
  display: inline-block;
  margin-top: 12px;
  font-size: var(--fs-xs);
  color: var(--text-3);
}

.side-more:hover {
  color: var(--brand);
}

/* 移动端：侧栏退到正文之后，卡片横滑（p15 首屏形态） */
@media (max-width: 1023px) {
  .sidebar {
    display: flex;
    flex-direction: row;
    gap: 12px;
    overflow-x: auto;
    padding-bottom: 6px;
    scroll-snap-type: x proximity;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
  }

  .sidebar::-webkit-scrollbar {
    display: none;
  }

  .sidebar .side-card {
    flex: 0 0 auto;
    width: 260px;
    scroll-snap-align: start;
  }

  /* 公告卡在横滑带里内容更窄，压一档内边距 */
  .sidebar .side-card--notice {
    width: 240px;
  }
}

/* 极窄屏：横滑带改为纵向堆叠，避免只剩一条缝 */
@media (max-width: 419px) {
  .sidebar {
    display: flex;
    flex-direction: column;
    overflow-x: visible;
  }

  .sidebar .side-card,
  .sidebar .side-card--notice {
    width: auto;
  }
}
</style>

