<template>
  <div class="container categories-page">
    <header class="page-title-bar">
      <h1 class="page-heading">分类</h1>
      <p v-if="site.categories.length" class="page-sub">共 {{ site.categories.length }} 个分类</p>
    </header>

    <ErrorState
      v-if="site.loadError && !site.categories.length"
      message="站点信息加载失败"
      @retry="site.ensureLoaded(true)"
    />

    <div v-else-if="!site.loaded && !site.categories.length" class="cat-skeleton" aria-hidden="true">
      <span v-for="i in 4" :key="i" class="skeleton sk-cat"></span>
    </div>

    <div v-else-if="site.categories.length" class="cat-grid">
      <RouterLink
        v-for="category in site.categories"
        :key="category.id"
        :to="`/category/${category.slug}`"
        class="cat-card"
      >
        <div class="cat-head">
          <h2 class="cat-name">{{ category.name }}</h2>
          <span v-if="typeof category.postCount === 'number'" class="cat-count">
            {{ category.postCount }} 篇
          </span>
        </div>
        <p class="cat-desc">{{ category.description || '查看该分类下的全部文章' }}</p>
      </RouterLink>
    </div>

    <EmptyState v-else title="暂无分类" description="博主还没有创建任何分类。" />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useSiteStore } from '@/stores/site'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

const site = useSiteStore()

onMounted(() => {
  void site.ensureLoaded()
})
</script>

<style scoped>
.categories-page {
  padding-bottom: 56px;
}

.cat-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 48px;
  margin-top: 16px;
}

@media (max-width: 639px) {
  .cat-grid {
    grid-template-columns: 1fr;
  }
}

.cat-card {
  display: block;
  padding: 16px 0;
  border-bottom: 1px solid var(--border);
}

.cat-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.cat-name {
  font-size: 15.5px;
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
  font-size: 11.5px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.cat-desc {
  margin-top: 6px;
  font-size: 13px;
  color: var(--text-3);
  line-height: 1.7;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.cat-skeleton {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 48px;
  margin-top: 22px;
}

@media (max-width: 639px) {
  .cat-skeleton {
    grid-template-columns: 1fr;
  }
}

.sk-cat {
  display: block;
  height: 76px;
  margin-bottom: 14px;
}
</style>
