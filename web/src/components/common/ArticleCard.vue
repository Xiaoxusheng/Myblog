<template>
  <article class="post-row">
    <div class="post-row-main">
      <h2 class="post-row-title">
        <span v-if="post.isTop" class="pin">置顶</span>
        <!-- eslint-disable-next-line vue/no-v-html：内容已 HTML 转义，仅注入 <mark> -->
        <RouterLink :to="`/post/${post.slug}`" v-html="titleHtml"></RouterLink>
      </h2>
      <p class="post-row-summary" :class="{ placeholder: !post.summary }">
        <!-- eslint-disable-next-line vue/no-v-html：同上 -->
        <span v-html="summaryHtml"></span>
      </p>
      <div class="post-row-meta">
        <span class="meta-date">{{ displayDate }}</span>
        <RouterLink
          v-if="post.category"
          :to="`/category/${post.category.slug}`"
          class="meta-cat"
        >
          {{ post.category.name }}
        </RouterLink>
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
        <span class="meta-tail">
          <span>阅读 {{ formatNumber(post.viewCount) }}</span>
          <span class="meta-dot">·</span>
          <span>{{ formatNumber(post.likeCount) }} 赞</span>
        </span>
      </div>
    </div>
    <RouterLink
      v-if="post.cover"
      :to="`/post/${post.slug}`"
      class="post-row-cover"
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
/* 编辑部式列表行：无卡片，hairline 分隔 */
.post-row {
  display: flex;
  gap: 24px;
  padding: 22px 0;
  border-bottom: 1px solid var(--border);
}

.post-row-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.post-row-title {
  font-size: 17px;
  font-weight: 600;
  line-height: 1.5;
}

.pin {
  margin-right: 8px;
  padding: 1px 8px;
  border: 1px solid var(--brand-soft-border);
  border-radius: 999px;
  background: var(--brand-soft);
  color: var(--brand);
  font-size: 11.5px;
  font-weight: 500;
  line-height: 1.6;
  vertical-align: 2px;
}

.post-row-title a {
  color: var(--text-1);
  transition: color var(--transition);
}

.post-row-title a:hover {
  color: var(--brand);
}

.post-row-summary {
  margin-top: 7px;
  color: var(--text-2);
  font-size: 13.5px;
  line-height: 1.75;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.post-row-summary.placeholder {
  color: var(--text-3);
}

.post-row-meta {
  margin-top: 10px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12.5px;
  color: var(--text-3);
}

/* 日期走 mono + 等宽数字，编辑感；行 hover 时与归档时间轴同语言变 accent（docs/07 §6.4） */
.meta-date {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.01em;
  transition: color var(--transition);
}

.post-row:hover .meta-date {
  color: var(--brand);
}

.meta-cat {
  color: var(--text-2);
}

.meta-cat:hover {
  color: var(--brand);
}

.meta-tags {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 8px;
}

.meta-tag {
  color: var(--text-3);
}

.meta-tag:hover {
  color: var(--brand);
}

.meta-tail {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-variant-numeric: tabular-nums;
}

.meta-dot {
  color: var(--border-strong);
}

.post-row-cover {
  flex-shrink: 0;
  align-self: center;
  width: 136px;
  height: 90px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--surface-2);
}

.post-row-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.45s ease-out, transform 0.4s var(--ease-out-quart);
}

.post-row-cover img.loaded {
  opacity: 1;
}

.post-row:hover .post-row-cover img.loaded {
  transform: scale(1.03);
}

@media (max-width: 640px) {
  .post-row {
    flex-direction: column-reverse;
    gap: 12px;
    padding: 18px 0;
  }

  .post-row-cover {
    width: 100%;
    height: 150px;
    align-self: stretch;
  }
}
</style>
