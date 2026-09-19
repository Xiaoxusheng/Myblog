<template>
  <!-- 仅移动端显示：把侧栏里有价值的信息转移到主列 -->
  <section class="mobile-discover">
    <div v-reveal class="discover-block">
      <div class="discover-head">
        <h2 class="discover-title">热门阅读</h2>
      </div>
      <div v-if="hotLoading" class="hot-scroll" aria-hidden="true">
        <div v-for="i in 3" :key="i" class="hot-card">
          <span class="skeleton" style="display: block; height: 14px; width: 80%"></span>
          <span class="skeleton" style="display: block; height: 12px; width: 45%"></span>
        </div>
      </div>
      <div v-else-if="hotPosts.length" class="hot-scroll">
        <RouterLink
          v-for="(post, index) in hotPosts"
          :key="post.id"
          :to="`/post/${post.slug}`"
          class="hot-card"
        >
          <span class="hot-rank" :class="{ top: index < 3 }">{{ String(index + 1).padStart(2, '0') }}</span>
          <span class="hot-name">{{ post.title }}</span>
          <span class="hot-meta">{{ formatNumber(post.viewCount) }} 阅读</span>
        </RouterLink>
      </div>
    </div>

    <div v-if="displayTags.length" v-reveal class="discover-block">
      <div class="discover-head">
        <h2 class="discover-title">标签</h2>
        <RouterLink v-if="site.tags.length > TAG_LIMIT" to="/tags" class="discover-more">
          全部标签 →
        </RouterLink>
      </div>
      <div class="tag-row">
        <RouterLink v-for="tag in displayTags" :key="tag.id" :to="`/tag/${tag.slug}`" class="chip">
          {{ tag.name }}
        </RouterLink>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useHotPosts } from '@/composables/useHotPosts'
import { useSiteStore } from '@/stores/site'
import { formatNumber } from '@/utils/format'

const TAG_LIMIT = 10
const site = useSiteStore()
const { hotPosts, hotLoading, load } = useHotPosts()

const displayTags = computed(() => site.tags.slice(0, TAG_LIMIT))

onMounted(() => {
  void load()
})
</script>

<style scoped>
.mobile-discover {
  display: none;
  margin-top: 36px;
  flex-direction: column;
  gap: 28px;
}

.discover-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 12px;
}

.discover-title {
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
}

.discover-more {
  font-size: 12.5px;
  color: var(--text-3);
}

.discover-more:hover {
  color: var(--brand);
}

/* 热门横滑卡片 */
.hot-scroll {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(220px, 64%);
  gap: 10px;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  padding-bottom: 4px;
  margin: 0 -20px;
  padding-left: 20px;
  padding-right: 20px;
}

.hot-card {
  scroll-snap-align: start;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
  transition: border-color var(--transition);
}

.hot-card:hover {
  border-color: var(--border-strong);
}

.hot-rank {
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
}

.hot-rank.top {
  color: var(--brand);
}

.hot-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-1);
  line-height: 1.55;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.hot-meta {
  font-size: 12px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

@media (max-width: 1023px) {
  .mobile-discover {
    display: flex;
  }
}
</style>
