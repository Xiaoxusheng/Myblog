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
        <!-- eslint-disable-next-line vue/no-v-html：内容已 HTML 转义，仅注入 <mark> -->
        <RouterLink :to="`/post/${post.slug}`" v-html="titleHtml"></RouterLink>
      </h2>
      <p class="post-card-summary" :class="{ placeholder: !post.summary }">
        <!-- eslint-disable-next-line vue/no-v-html：同上 -->
        <span v-html="summaryHtml"></span>
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
      <img ref="coverEl" :src="post.cover" :alt="post.title" loading="lazy" @load="onImgLoad" />
    </RouterLink>
  </article>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { PostSummary } from '@/types'
import { formatDate, formatNumber } from '@/utils/format'
import { highlightText } from '@/utils/highlight'

const props = withDefaults(defineProps<{ post: PostSummary; keyword?: string }>(), {
  keyword: ''
})

const titleHtml = computed(() => highlightText(props.post.title, props.keyword))
const summaryHtml = computed(() => highlightText(props.post.summary || '暂无摘要', props.keyword))

const displayDate = computed(() => formatDate(props.post.publishedAt || props.post.createdAt))

// 封面加载完成后淡入（缓存图在 mounted 前可能已 complete，需补一次）
const coverEl = ref<HTMLImageElement | null>(null)

function onImgLoad(): void {
  coverEl.value?.classList.add('loaded')
}

onMounted(() => {
  if (coverEl.value?.complete) coverEl.value.classList.add('loaded')
})
</script>

<style scoped>
.post-card {
  display: flex;
  gap: 20px;
  padding: 20px;
  transition: border-color var(--transition);
}

/* hover 克制：不整卡上浮，只做边界/标题/封面的联动反馈 */
.post-card:hover {
  border-color: var(--border-strong);
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
  font-size: 18px;
  font-weight: 600;
  line-height: 1.5;
  letter-spacing: 0.1px;
}

.post-card-title a {
  color: var(--text-1);
  background-image: linear-gradient(var(--brand), var(--brand));
  background-size: 0% 2px;
  background-repeat: no-repeat;
  background-position: 0 calc(100% - 1px);
  transition: background-size 0.3s var(--ease-out-quart), color var(--transition);
}

.post-card-title a:hover {
  color: var(--brand);
  background-size: 100% 2px;
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
  opacity: 0;
  transform: scale(1);
  transition: opacity 0.45s ease-out, transform 0.35s var(--ease-out-quart);
}

.post-card-cover img.loaded {
  opacity: 1;
}

.post-card:hover .post-card-cover img.loaded {
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
