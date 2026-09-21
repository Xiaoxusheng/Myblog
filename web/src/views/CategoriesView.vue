<template>
  <div class="container tax-page">
    <header class="page-title-bar">
      <p class="kicker">TAXONOMY · 内容索引</p>
      <h1 class="page-heading">分类与标签</h1>
      <p class="page-sub">按主题归类阅读，或从标签直接跳到关心的技术点。</p>
    </header>

    <ErrorState
      v-if="site.loadError && !site.loaded"
      message="站点信息加载失败"
      @retry="site.ensureLoaded(true)"
    />

    <template v-else>
      <SegmentedTabs v-model="tab" :tabs="tabs" class="tax-tabs" />

      <!-- 分类：3 列卡片网格（p06） -->
      <section v-if="tab === 'category'" class="tax-section">
        <h2 class="tax-head">
          全部分类
          <span class="tax-count">{{ site.categories.length }} 个分类</span>
        </h2>

        <div v-if="!site.loaded && !site.categories.length" class="cat-grid" aria-hidden="true">
          <span v-for="i in 6" :key="i" class="skeleton sk-cat"></span>
        </div>

        <div v-else-if="site.categories.length" class="cat-grid">
          <RouterLink
            v-for="category in site.categories"
            :key="category.id"
            :to="`/category/${category.slug}`"
            class="cat-card"
          >
            <div class="cat-head">
              <h3 class="cat-name">{{ category.name }}</h3>
              <span v-if="typeof category.postCount === 'number'" class="cat-count">
                {{ category.postCount }} 篇
              </span>
            </div>
            <p class="cat-desc">{{ category.description || '查看该分类下的全部文章' }}</p>
          </RouterLink>
        </div>

        <EmptyState v-else title="暂无分类" description="博主还没有创建任何分类。" />
      </section>

      <!-- 标签：chip 云（p06） -->
      <section v-else class="tax-section">
        <h2 class="tax-head">
          全部标签
          <span class="tax-count">{{ site.tags.length }} 个标签</span>
        </h2>

        <div v-if="!site.loaded && !site.tags.length" class="tag-cloud" aria-hidden="true">
          <span
            v-for="i in 10"
            :key="i"
            class="skeleton sk-tag"
            :style="{ width: `${56 + ((i * 37) % 62)}px` }"
          ></span>
        </div>

        <div v-else-if="site.tags.length" class="tag-cloud">
          <RouterLink
            v-for="tag in site.tags"
            :key="tag.id"
            :to="`/tag/${tag.slug}`"
            class="chip tag-item"
          >
            {{ tag.name }}
            <span v-if="typeof tag.postCount === 'number'" class="tag-count">{{ tag.postCount }}</span>
          </RouterLink>
        </div>

        <EmptyState v-else title="暂无标签" description="博主还没有创建任何标签。" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSiteStore } from '@/stores/site'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'

const route = useRoute()
const router = useRouter()
const site = useSiteStore()

/** Tab 状态与 URL 同步：/categories（分类）与 /tags（标签）共用一个视图 */
const tab = ref<'category' | 'tag'>(route.path.startsWith('/tags') ? 'tag' : 'category')

const tabs = computed(() => [
  { value: 'category', label: '分类', count: site.categories.length },
  { value: 'tag', label: '标签', count: site.tags.length }
])

// tab 变化 → 同步 URL；URL 变化（含浏览器后退）→ 同步 tab
watch(tab, (next) => {
  const target = next === 'tag' ? '/tags' : '/categories'
  if (route.path !== target) void router.replace(target)
})

watch(
  () => route.path,
  (path) => {
    const next = path.startsWith('/tags') ? 'tag' : 'category'
    if (next !== tab.value) tab.value = next
  }
)

onMounted(() => {
  void site.ensureLoaded()
})
</script>

<style scoped>
.tax-page {
  padding-bottom: 64px;
}

.tax-tabs {
  margin-top: 24px;
}

.tax-section {
  margin-top: 28px;
}

.tax-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: var(--fs-xl);
  line-height: var(--lh-heading-2);
  letter-spacing: -0.01em;
}

.tax-count {
  font-family: var(--font-sans);
  font-size: var(--fs-sm);
  font-weight: 400;
  color: var(--text-3);
}

/* ---------- 分类卡片：3 列（p06） ---------- */
.cat-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 18px;
}

.cat-card {
  display: flex;
  flex-direction: column;
  padding: 20px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg);
  transition: border-color var(--transition), background var(--transition);
}

.cat-card:hover {
  border-color: var(--border-strong);
  background: var(--surface);
}

.cat-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.cat-name {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--text-1);
  transition: color var(--transition);
}

.cat-card:hover .cat-name {
  color: var(--brand);
}

.cat-count {
  flex-shrink: 0;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.cat-desc {
  margin-top: 8px;
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.7;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.sk-cat {
  display: block;
  height: 96px;
  border-radius: var(--radius-lg);
}

/* ---------- 标签云（p06） ---------- */
.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 9px;
  margin-top: 18px;
}

.tag-item {
  padding: 5px 12px;
  font-size: var(--fs-sm);
  color: var(--text-1);
}

.tag-count {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

a.tag-item:hover .tag-count {
  color: var(--text-2);
}

.sk-tag {
  display: block;
  height: 30px;
  border-radius: var(--radius-pill);
}

@media (max-width: 900px) {
  .cat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .cat-grid {
    grid-template-columns: 1fr;
  }
}
</style>
