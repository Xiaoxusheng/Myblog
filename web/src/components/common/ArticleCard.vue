<template>
  <article class="post-card">
    <RouterLink
      :to="`/post/${post.slug}`"
      class="post-card-cover"
      :class="`tint-${tintName}`"
      tabindex="-1"
      aria-hidden="true"
    >
      <img v-if="post.cover" ref="coverEl" :src="post.cover" :alt="post.title" loading="lazy" @load="onImgLoad" />
    </RouterLink>

    <div class="post-card-main">
      <div class="post-card-flags">
        <span v-if="post.isTop" class="pin">置顶</span>
        <span class="post-card-date">{{ displayDate }}</span>
      </div>

      <h2 class="post-card-title">
        <!-- eslint-disable-next-line vue/no-v-html：内容已 HTML 转义，仅注入 <mark> -->
        <span v-html="titleHtml"></span>
      </h2>

      <p class="post-card-summary" :class="{ placeholder: !post.summary }">
        <!-- eslint-disable-next-line vue/no-v-html：同上 -->
        <span v-html="summaryHtml"></span>
      </p>

      <div class="post-card-meta">
        <span>{{ post.category?.name || '未分类' }}</span>
        <span class="meta-dot">·</span>
        <span>阅读 {{ formatNumber(post.viewCount) }}</span>
        <span class="meta-dot">·</span>
        <span>{{ formatNumber(post.likeCount) }} 赞</span>
        <span v-if="readingMinutes" class="meta-dot">·</span>
        <span v-if="readingMinutes">约 {{ readingMinutes }} 分钟</span>
      </div>

      <RouterLink :to="`/post/${post.slug}`" class="post-card-more">
        继续阅读 →
      </RouterLink>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { PostSummary } from '@/types'
import { formatDate, formatNumber } from '@/utils/format'
import { highlightText } from '@/utils/highlight'

const props = withDefaults(
  defineProps<{ post: PostSummary; keyword?: string; tintIndex?: number }>(),
  { keyword: '', tintIndex: 0 }
)

const TINTS = ['sky', 'mint', 'peach', 'lavender'] as const

/** 无封面时的色块：四色按索引轮转（p02 卡片左色块） */
const tintName = computed(() => TINTS[props.tintIndex % TINTS.length])

const titleHtml = computed(() => highlightText(props.post.title, props.keyword))
const summaryHtml = computed(() => highlightText(props.post.summary || '暂无摘要', props.keyword))

/** p02 卡片日期带「更新」后缀 */
const displayDate = computed(() => {
  const d = formatDate(props.post.publishedAt || props.post.createdAt)
  return d ? `${d} 更新` : ''
})

/** 摘要长度估算阅读时长（CJK 400 字/分 + 其余 200 词/分），与详情页口径一致 */
const readingMinutes = computed(() => {
  const text = `${props.post.title ?? ''}${props.post.summary ?? ''}`
  if (!text.trim()) return 0
  const cjk = (text.match(/[\u4e00-\u9fa5]/g) ?? []).length
  const words = text.replace(/[\u4e00-\u9fa5]/g, ' ').trim().split(/\s+/).filter(Boolean).length
  return Math.max(1, Math.round(cjk / 400 + words / 200))
})

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
/* 文章卡片：12px 圆角 + 发丝描边，左色块右文字（p02） */
.post-card {
  display: flex;
  gap: 22px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg);
  transition: border-color var(--transition), box-shadow var(--transition);
}

.post-card:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow-sm);
}

/* 封面区：有图显示图，无图显示 tint 色块；固定在左侧，占卡片主体（p02 实测约 1:1） */
.post-card-cover {
  flex-shrink: 0;
  align-self: stretch;
  width: 46%;
  max-width: 420px;
  min-height: 128px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--surface);
}

.post-card-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.45s ease-out, transform 0.4s var(--ease-out-quart);
}

.post-card-cover img.loaded {
  opacity: 1;
}

.post-card:hover .post-card-cover img.loaded {
  transform: scale(1.04);
}

.tint-sky {
  background: var(--tint-sky);
}

.tint-mint {
  background: var(--tint-mint);
}

.tint-peach {
  background: var(--tint-peach);
}

.tint-lavender {
  background: var(--tint-lavender);
}

.post-card-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

/* 顶行：置顶徽标 + mono 日期 */
.post-card-flags {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 22px;
}

.post-card-flags:empty {
  display: none;
}

.pin {
  padding: 1px 8px;
  border-radius: var(--radius-xs);
  background: #f7e2a0;
  color: #6b4f08;
  font-size: var(--fs-xs);
  font-weight: 500;
  line-height: 1.7;
}

html.dark .pin {
  background: #4a3c12;
  color: #f0d98a;
}

.post-card-date {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.post-card-title {
  margin-top: 6px;
  font-size: var(--fs-xl);
  font-weight: 650;
  line-height: 1.32;
  letter-spacing: -0.015em;
}

.post-card-title span {
  color: var(--text-1);
  transition: color var(--transition);
}

.post-card:hover .post-card-title span {
  color: var(--brand);
}

.post-card-summary {
  margin-top: 8px;
  color: var(--text-2);
  font-size: var(--fs-sm);
  line-height: 1.7;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.post-card-summary.placeholder {
  color: var(--text-3);
}

/* meta 行：纯文字（p02，不用分类徽章） */
.post-card-meta {
  margin-top: 9px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: var(--fs-xs);
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.meta-dot {
  color: var(--border-strong);
}

.post-card-more {
  margin-top: 9px;
  align-self: flex-start;
  font-size: var(--fs-sm);
  color: var(--brand);
  transition: color var(--transition);
}

.post-card:hover .post-card-more {
  color: var(--brand-hover);
}

@media (max-width: 767px) {
  .post-card {
    gap: 14px;
    padding: 14px;
  }

  /* 移动端退回小缩略图，文字优先（p15） */
  .post-card-cover {
    width: 104px;
    max-width: none;
    min-height: 92px;
    align-self: flex-start;
  }

  .post-card-title {
    font-size: var(--fs-lg);
    line-height: 1.38;
  }

  .post-card-summary {
    -webkit-line-clamp: 2;
    line-clamp: 2;
  }
}
</style>

