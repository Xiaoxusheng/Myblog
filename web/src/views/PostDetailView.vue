<template>
  <div class="container post-detail" :class="{ 'with-toc': toc.length > 0 }">
    <article class="post-main">
      <!-- 加载骨架 -->
      <div v-if="loading" class="post-skeleton" aria-hidden="true">
        <span class="skeleton sk-chip"></span>
        <span class="skeleton sk-title"></span>
        <span class="skeleton sk-meta"></span>
        <div class="sk-body">
          <span v-for="i in 8" :key="i" class="skeleton sk-line" :style="{ width: lineWidth(i) }"></span>
        </div>
      </div>

      <ErrorState v-else-if="error" :message="error" @retry="load" />

      <EmptyState
        v-else-if="notFound"
        title="文章不存在或已下线"
        description="它可能已被删除，或者链接有误。"
      >
        <RouterLink class="btn" to="/">返回首页</RouterLink>
      </EmptyState>

      <template v-else-if="post">
        <header class="post-header">
          <div class="post-cats">
            <RouterLink
              v-if="post.category"
              :to="`/category/${post.category.slug}`"
              class="chip cat"
            >
              {{ post.category.name }}
            </RouterLink>
          </div>
          <h1 class="post-title">{{ post.title }}</h1>
          <div class="post-meta">
            <span>发布于 {{ formatDate(post.publishedAt || post.createdAt) }}</span>
            <template v-if="post.updatedAt && post.updatedAt !== post.createdAt">
              <span class="dot">·</span>
              <span>更新于 {{ formatDate(post.updatedAt) }}</span>
            </template>
            <span class="dot">·</span>
            <span>阅读 {{ formatNumber(post.viewCount) }}</span>
          </div>
        </header>

        <div v-if="post.cover" class="post-cover">
          <img :src="post.cover" :alt="post.title" />
        </div>

        <div class="post-content card">
          <!-- 内容由后台管理员通过 Markdown 维护，markdown-it 渲染时不放行内嵌 HTML -->
          <div class="markdown-body" v-html="html"></div>
        </div>

        <div class="post-footer">
          <div v-if="post.tags.length" class="post-tags">
            <RouterLink v-for="tag in post.tags" :key="tag.id" :to="`/tag/${tag.slug}`" class="chip">
              # {{ tag.name }}
            </RouterLink>
          </div>
          <button class="like-btn" :class="{ liked }" :aria-pressed="liked" @click="onLike">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden="true">
              <path
                d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"
              />
            </svg>
            <span>{{ formatNumber(likeCount) }}</span>
            <span class="like-text">{{ liked ? '已赞' : '点赞' }}</span>
          </button>
        </div>

        <nav v-if="prev || next" class="post-nav" aria-label="上下篇">
          <RouterLink v-if="prev" :to="`/post/${prev.slug}`" class="nav-card card">
            <span class="nav-label">上一篇</span>
            <span class="nav-title">{{ prev.title }}</span>
          </RouterLink>
          <span v-else class="nav-card card placeholder">
            <span class="nav-label">上一篇</span>
            <span class="nav-title">没有更多了</span>
          </span>
          <RouterLink v-if="next" :to="`/post/${next.slug}`" class="nav-card card next">
            <span class="nav-label">下一篇</span>
            <span class="nav-title">{{ next.title }}</span>
          </RouterLink>
          <span v-else class="nav-card card placeholder">
            <span class="nav-label">下一篇</span>
            <span class="nav-title">没有更多了</span>
          </span>
        </nav>

        <section v-if="related.length" class="related">
          <h2 class="related-heading">相关文章</h2>
          <div class="related-grid">
            <RouterLink
              v-for="item in related"
              :key="item.id"
              :to="`/post/${item.slug}`"
              class="related-card card"
            >
              <p class="related-title">{{ item.title }}</p>
              <p class="related-meta">
                {{ item.category?.name || '未分类' }} ·
                {{ formatDate(item.publishedAt || item.createdAt) }}
              </p>
            </RouterLink>
          </div>
        </section>

        <CommentSection :key="slug" :slug="slug" :enabled="site.settings.commentEnabled" />
      </template>
    </article>

    <aside v-if="toc.length" class="post-toc-aside">
      <PostToc :headings="toc" />
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { fetchPostDetail, likePost } from '@/api/post'
import { ApiError } from '@/api/http'
import { useSiteStore } from '@/stores/site'
import { renderMarkdown, type TocItem } from '@/utils/markdown'
import { formatDate, formatNumber } from '@/utils/format'
import { isPostLiked, markPostLiked } from '@/utils/storage'
import { applyDocumentTitle } from '@/utils/title'
import type { PostDetail, PostNav, PostSummary } from '@/types'
import PostToc from '@/components/post/PostToc.vue'
import CommentSection from '@/components/post/CommentSection.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const site = useSiteStore()
const toast = useToast()

const slug = computed(() => String(route.params.slug))

const post = ref<PostDetail | null>(null)
const prev = ref<PostNav | null>(null)
const next = ref<PostNav | null>(null)
const related = ref<PostSummary[]>([])
const html = ref('')
const toc = ref<TocItem[]>([])
const loading = ref(true)
const error = ref('')
const notFound = ref(false)
const liked = ref(false)
const likeCount = ref(0)

// 骨架行宽节奏，避免每行等宽的呆板感
function lineWidth(index: number): string {
  const widths = ['100%', '92%', '78%', '100%', '86%', '95%', '64%', '40%']
  return widths[(index - 1) % widths.length]
}

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  try {
    const data = await fetchPostDetail(slug.value)
    post.value = data.post
    prev.value = data.prev ?? null
    next.value = data.next ?? null
    related.value = data.related ?? []
    const rendered = renderMarkdown(data.post.content ?? '')
    html.value = rendered.html
    toc.value = rendered.toc
    likeCount.value = data.post.likeCount ?? 0
    liked.value = isPostLiked(data.post.id)
    applyDocumentTitle(data.post.title)
  } catch (e) {
    if (e instanceof ApiError && e.code === 10004) {
      notFound.value = true
    } else {
      error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
    }
    applyDocumentTitle('文章')
  } finally {
    loading.value = false
  }
}

async function onLike(): Promise<void> {
  if (!post.value) return
  if (liked.value) {
    toast.info('已经点过赞啦')
    return
  }
  try {
    likeCount.value = await likePost(slug.value)
    liked.value = true
    markPostLiked(post.value.id)
    toast.success('感谢点赞！')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '点赞失败，请稍后重试')
  }
}

watch(slug, () => void load(), { immediate: true })
</script>

<style scoped>
.post-detail {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 36px;
  align-items: start;
  padding-top: 28px;
  padding-bottom: 56px;
}

.post-detail.with-toc {
  grid-template-columns: minmax(0, 1fr) 240px;
}

.post-toc-aside {
  min-width: 0;
}

@media (max-width: 1199px) {
  .post-detail.with-toc {
    grid-template-columns: minmax(0, 1fr);
  }

  .post-toc-aside {
    display: none;
  }
}

.post-main {
  min-width: 0;
}

.post-header {
  text-align: left;
}

.post-cats {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.chip.cat {
  background: var(--brand-soft);
  color: var(--brand);
}

.post-title {
  font-size: 27px;
  font-weight: 600;
  line-height: 1.4;
  letter-spacing: 0.2px;
}

.post-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 12px;
  font-size: 13px;
  color: var(--text-3);
}

.dot {
  color: var(--border-strong);
}

.post-cover {
  margin-top: 22px;
  border-radius: var(--radius-lg);
  overflow: hidden;
  border: 1px solid var(--border);
  background: var(--surface-2);
}

.post-cover img {
  display: block;
  width: 100%;
  max-height: 430px;
  object-fit: cover;
}

.post-content {
  margin-top: 24px;
  padding: 28px 32px;
}

.post-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 14px;
  margin-top: 26px;
}

.post-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.like-btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 38px;
  padding: 0 18px;
  border: 1px solid var(--border-strong);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-2);
  font-size: 14px;
  font-variant-numeric: tabular-nums;
  transition: color var(--transition), border-color var(--transition),
    background var(--transition), transform var(--transition);
}

.like-btn:hover {
  color: var(--brand);
  border-color: var(--brand);
}

.like-btn:active {
  transform: scale(0.96);
}

.like-btn.liked {
  color: var(--brand);
  border-color: var(--brand-soft-border);
  background: var(--brand-soft);
}

.like-text {
  font-size: 13px;
}

/* 上一篇/下一篇 */
.post-nav {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  margin-top: 30px;
}

.nav-card {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 14px 18px;
  min-width: 0;
}

.nav-card.next {
  text-align: right;
  align-items: flex-end;
}

.nav-label {
  font-size: 12px;
  color: var(--text-3);
}

.nav-title {
  font-size: 14.5px;
  font-weight: 500;
  color: var(--text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color var(--transition);
}

a.nav-card:hover .nav-title {
  color: var(--brand);
}

.nav-card.placeholder .nav-title {
  color: var(--text-3);
  font-weight: 400;
}

/* 相关文章 */
.related {
  margin-top: 36px;
}

.related-heading {
  font-size: 18px;
  font-weight: 600;
}

.related-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-top: 14px;
}

.related-card {
  padding: 14px 16px;
  transition: border-color var(--transition), box-shadow 0.2s ease-out;
}

.related-card:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow-md);
}

.related-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-1);
  line-height: 1.55;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  transition: color var(--transition);
}

.related-card:hover .related-title {
  color: var(--brand);
}

.related-meta {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-3);
}

@media (max-width: 640px) {
  .related-grid {
    grid-template-columns: 1fr;
  }

  .post-nav {
    grid-template-columns: 1fr;
  }

  .nav-card.next {
    text-align: left;
    align-items: flex-start;
  }

  .post-content {
    padding: 20px 18px;
  }

  .post-title {
    font-size: 22px;
  }
}

/* 详情骨架 */
.post-skeleton {
  display: flex;
  flex-direction: column;
}

.sk-chip {
  width: 72px;
  height: 22px;
  border-radius: 6px;
}

.sk-title {
  margin-top: 16px;
  width: 70%;
  height: 30px;
  border-radius: 8px;
}

.sk-meta {
  margin-top: 14px;
  width: 40%;
  height: 13px;
}

.sk-body {
  margin-top: 28px;
  padding: 28px 32px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sk-line {
  display: block;
  height: 13px;
}
</style>
