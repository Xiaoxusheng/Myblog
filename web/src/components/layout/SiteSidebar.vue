<template>
  <aside class="page-aside sidebar">
    <section v-if="site.settings.notice" v-reveal class="side-card notice-card">
      <h3 class="side-title">公告</h3>
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
            <span class="hot-rank" :class="{ top: index < 3 }">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="hot-title">{{ post.title }}</span>
          </RouterLink>
          <span class="hot-views">{{ formatNumber(post.viewCount) }}</span>
        </li>
      </ol>
      <p v-else class="side-empty">暂无热门文章</p>
    </section>

    <section v-if="navSeries.length" v-reveal="{ delay: 90 }" class="side-card">
      <h3 class="side-title">专题</h3>
      <ul class="series-list">
        <li v-for="item in navSeries" :key="item.id" class="series-row">
          <RouterLink :to="`/series/${item.slug}`" class="series-item">
            <span class="series-name">{{ item.name }}</span>
            <span class="series-count">{{ item.postCount ?? 0 }} 篇</span>
          </RouterLink>
        </li>
      </ul>
      <RouterLink to="/series" class="tag-more">全部专题 →</RouterLink>
    </section>

    <section v-if="cloudTags.length" v-reveal="{ delay: 120 }" class="side-card">
      <h3 class="side-title">标签</h3>
      <div class="tag-cloud">
        <RouterLink
          v-for="tag in cloudTags"
          :key="tag.id"
          :to="`/tag/${tag.slug}`"
          class="chip"
        >
          {{ tag.name }}
        </RouterLink>
      </div>
      <RouterLink v-if="site.tags.length > 24" to="/tags" class="tag-more">更多标签 →</RouterLink>
    </section>
  </aside>
</template>

<script lang="ts">
import type { Series } from '@/types'
import { fetchSeriesList } from '@/api/series'

/**
 * 专题导航模块级缓存：一个页面会话只发一次请求（与 useHotPosts 同一策略）
 */
const seriesCache: { loaded: boolean; list: Series[] } = { loaded: false, list: [] }

async function loadSeriesNav(): Promise<Series[]> {
  if (seriesCache.loaded) return seriesCache.list
  const list = await fetchSeriesList()
  seriesCache.list = list.slice(0, 3)
  seriesCache.loaded = true
  return seriesCache.list
}
</script>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useHotPosts } from '@/composables/useHotPosts'
import { useSiteStore } from '@/stores/site'
import { formatNumber } from '@/utils/format'
import ErrorState from '@/components/common/ErrorState.vue'

const site = useSiteStore()
const { hotPosts, hotLoading, hotError, load } = useHotPosts()

const cloudTags = computed(() => site.tags.slice(0, 24))

/** 侧栏专题导航：最多展示 3 个 */
const navSeries = ref<Series[]>([])

onMounted(() => {
  void load()
  loadSeriesNav()
    .then((list) => {
      navSeries.value = list
    })
    .catch(() => {
      // 侧栏专题导航加载失败静默隐藏，不影响主体内容
    })
})

onBeforeUnmount(() => {
  // 热门文章是模块级共享状态，离开页面不中断已完成的加载
})
</script>

<style scoped>
.side-card {
  padding: 16px 18px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

/* 公告：弱底色与列表卡区分层级 */
.notice-card {
  background: var(--brand-soft);
  border-color: transparent;
}

/* 编辑感小标题：字距 + 弱化色，替代左侧竖条 */
.side-title {
  margin: 0 0 12px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.12em;
  color: var(--text-3);
}

.notice {
  font-size: 13.5px;
  color: var(--text-2);
  line-height: 1.8;
  white-space: pre-wrap;
}

/* 热门文章：真正的列表结构 */
.hot-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.hot-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.hot-row + .hot-row {
  border-top: 1px solid var(--border);
}

.hot-item {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 9px 0;
}

.hot-rank {
  flex-shrink: 0;
  width: 22px;
  font-size: 12.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
}

.hot-rank.top {
  color: var(--brand);
}

.hot-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13.5px;
  color: var(--text-1);
  transition: color var(--transition);
}

.hot-item:hover .hot-title {
  color: var(--brand);
}

.hot-views {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
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

/* 专题：名称 + 篇数的轻量列表 */
.series-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.series-row + .series-row {
  border-top: 1px solid var(--border);
}

.series-item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 0;
}

.series-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13.5px;
  color: var(--text-1);
  transition: color var(--transition);
}

.series-item:hover .series-name {
  color: var(--brand);
}

.series-count {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.side-empty {
  font-size: 13px;
  color: var(--text-3);
}

/* 标签：轻量 chip + 更多入口 */
.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-more {
  display: inline-block;
  margin-top: 12px;
  font-size: 12.5px;
  color: var(--text-3);
}

.tag-more:hover {
  color: var(--brand);
}
</style>
