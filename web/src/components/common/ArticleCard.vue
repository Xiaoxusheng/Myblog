<template>
  <article class="post-card card">
    <div class="post-card-main">
      <div class="post-card-flags">
        <span v-if="post.isTop" class="pin">置顶</span>
        <RouterLink v-if="post.category" :to="`/category/${post.category.slug}`" class="chip cat">
          {{ post.category.name }}
        </RouterLink>
      </div>
      <h2 class="post-card-title">
        <RouterLink :to="`/post/${post.slug}`">{{ post.title }}</RouterLink>
      </h2>
      <p class="post-card-summary" :class="{ placeholder: !post.summary }">
        {{ post.summary || '暂无摘要' }}
      </p>
      <div class="post-card-meta">
        <span class="meta-item">{{ displayDate }}</span>
        <span class="meta-dot">·</span>
        <span class="meta-item">阅读 {{ formatNumber(post.viewCount) }}</span>
        <span class="meta-dot">·</span>
        <span class="meta-item">{{ formatNumber(post.likeCount) }} 赞</span>
        <span v-if="post.tags.length" class="meta-tags">
          <RouterLink
            v-for="tag in post.tags"
            :key="tag.id"
            :to="`/tag/${tag.slug}`"
            class="meta-tag"
          >
            # {{ tag.name }}
          </RouterLink>
        </span>
      </div>
    </div>
    <RouterLink
      v-if="post.cover"
      :to="`/post/${post.slug}`"
      class="post-card-cover"
      tabindex="-1"
      aria-hidden="true"
    >
      <img :src="post.cover" :alt="post.title" loading="lazy" />
    </RouterLink>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { PostSummary } from '@/types'
import { formatDate, formatNumber } from '@/utils/format'

const props = defineProps<{ post: PostSummary }>()

const displayDate = computed(() => formatDate(props.post.publishedAt || props.post.createdAt))
</script>

<style scoped>
.post-card {
  display: flex;
  gap: 20px;
  padding: 20px;
  transition: border-color var(--transition), box-shadow 0.2s ease-out, transform 0.2s ease-out;
}

.post-card:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.post-card-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.post-card-flags {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.pin {
  padding: 0 8px;
  border: 1px solid var(--brand-soft-border);
  border-radius: 6px;
  background: var(--brand-soft);
  color: var(--brand);
  font-size: 12px;
  line-height: 1.7;
}

.post-card-title {
  font-size: 17.5px;
  font-weight: 600;
  line-height: 1.5;
}

.post-card-title a {
  color: var(--text-1);
}

.post-card-title a:hover {
  color: var(--brand);
}

.post-card-summary {
  margin-top: 8px;
  color: var(--text-2);
  font-size: 14px;
  line-height: 1.75;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.post-card-summary.placeholder {
  color: var(--text-3);
}

.post-card-meta {
  margin-top: auto;
  padding-top: 12px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 12.5px;
  color: var(--text-3);
}

.meta-dot {
  color: var(--border-strong);
}

.meta-tag {
  margin-left: 8px;
  color: var(--text-3);
}

.meta-tag:hover {
  color: var(--brand);
}

.post-card-cover {
  flex-shrink: 0;
  align-self: center;
  width: 168px;
  height: 112px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: var(--surface-2);
}

.post-card-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s ease-out;
}

.post-card:hover .post-card-cover img {
  transform: scale(1.04);
}

@media (max-width: 640px) {
  .post-card {
    flex-direction: column-reverse;
    gap: 12px;
    padding: 16px;
  }

  .post-card-cover {
    width: 100%;
    height: 150px;
    align-self: stretch;
  }
}
</style>
