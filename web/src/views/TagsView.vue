<template>
  <div class="container tags-page">
    <header class="page-title-bar">
      <h1 class="page-heading">标签</h1>
      <p v-if="site.tags.length" class="page-sub">共 {{ site.tags.length }} 个标签</p>
    </header>

    <ErrorState
      v-if="site.loadError && !site.tags.length"
      message="站点信息加载失败"
      @retry="site.ensureLoaded(true)"
    />

    <div v-else-if="!site.loaded && !site.tags.length" class="tag-skeleton" aria-hidden="true">
      <span v-for="i in 10" :key="i" class="skeleton sk-tag" :style="{ width: `${52 + ((i * 37) % 60)}px` }"></span>
    </div>

    <div v-else-if="site.tags.length" class="tag-cloud">
      <RouterLink
        v-for="tag in site.tags"
        :key="tag.id"
        :to="`/tag/${tag.slug}`"
        class="tag-item"
      >
        {{ tag.name }}
        <sup v-if="typeof tag.postCount === 'number'" class="tag-count">{{ tag.postCount }}</sup>
      </RouterLink>
    </div>

    <EmptyState v-else title="暂无标签" description="博主还没有创建任何标签。" />
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
.tags-page {
  padding-bottom: 56px;
}

.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 26px;
}

.tag-item {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  padding: 7px 16px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-2);
  font-size: 14px;
  transition: color var(--transition), border-color var(--transition),
    background var(--transition), transform var(--transition);
}

.tag-item:hover {
  color: var(--brand);
  border-color: var(--brand-soft-border);
  background: var(--brand-soft);
  transform: translateY(-2px);
}

.tag-count {
  font-size: 11.5px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.tag-item:hover .tag-count {
  color: var(--brand);
}

.tag-skeleton {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 26px;
}

.sk-tag {
  display: block;
  height: 34px;
  border-radius: 999px;
}
</style>
