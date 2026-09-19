<template>
  <div class="container post-detail" :class="{ 'with-toc': toc.length > 0 }">
    <ReadingProgress />
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
            <span class="dot">·</span>
            <span>约 {{ readingMinutes }} 分钟</span>
            <template v-if="series">
              <span class="dot">·</span>
              <span>
                专题：<RouterLink :to="`/series/${series.slug}`" class="series-link">
                  {{ series.name }}
                </RouterLink>
                <span class="dot">·</span>第 {{ series.index }} / {{ series.total }} 篇
              </span>
            </template>
            <button v-if="toc.length" class="toc-toggle" type="button" @click="tocOpen = true">
              目录
              <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
                <path d="M9 6l6 6-6 6" />
              </svg>
            </button>
          </div>
        </header>

        <div v-if="post.cover" class="post-cover">
          <img ref="coverEl" :src="post.cover" :alt="post.title" @load="onImgLoad" />
        </div>

        <div v-reveal class="post-content card">
          <!-- 内容由后台管理员通过 Markdown 维护，markdown-it 渲染时不放行内嵌 HTML -->
          <div class="markdown-body" v-html="html" @click="onContentClick"></div>
        </div>

        <div class="post-footer">
          <div v-if="post.tags.length" class="post-tags">
            <RouterLink v-for="tag in post.tags" :key="tag.id" :to="`/tag/${tag.slug}`" class="chip">
              # {{ tag.name }}
            </RouterLink>
          </div>
          <button
            class="like-btn"
            :class="{ liked, bursting }"
            :aria-pressed="liked"
            @click="onLike"
          >
            <span class="like-heart">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden="true">
                <path
                  d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"
                />
              </svg>
              <span v-if="bursting" class="like-burst" aria-hidden="true">
                <i v-for="i in 6" :key="i" class="burst-dot" :style="{ '--angle': `${(i - 1) * 60}deg` }"></i>
              </span>
            </span>
            <span>{{ formatNumber(likeCount) }}</span>
            <span class="like-text">{{ liked ? '已赞' : '点赞' }}</span>
          </button>
        </div>

        <nav v-if="prev || next" v-reveal class="post-nav" aria-label="上下篇">
          <RouterLink v-if="prev" :to="`/post/${prev.slug}`" class="nav-card card">
            <span class="nav-label"><span class="nav-arrow">←</span>上一篇</span>
            <span class="nav-title">{{ prev.title }}</span>
          </RouterLink>
          <span v-else class="nav-card card placeholder">
            <span class="nav-label"><span class="nav-arrow">←</span>上一篇</span>
            <span class="nav-title">没有更多了</span>
          </span>
          <RouterLink v-if="next" :to="`/post/${next.slug}`" class="nav-card card next">
            <span class="nav-label">下一篇<span class="nav-arrow">→</span></span>
            <span class="nav-title">{{ next.title }}</span>
          </RouterLink>
          <span v-else class="nav-card card placeholder">
            <span class="nav-label">下一篇<span class="nav-arrow">→</span></span>
            <span class="nav-title">没有更多了</span>
          </span>
        </nav>

        <!-- 本专题上下篇：与全局上下篇并列，仅在文章属于专题时展示 -->
        <div v-if="series && (seriesPrev || seriesNext)" v-reveal class="series-nav">
          <p class="series-nav-label">本专题</p>
          <nav class="post-nav" aria-label="本专题上下篇">
            <RouterLink v-if="seriesPrev" :to="`/post/${seriesPrev.slug}`" class="nav-card card">
              <span class="nav-label"><span class="nav-arrow">←</span>本专题上一篇</span>
              <span class="nav-title">{{ seriesPrev.title }}</span>
            </RouterLink>
            <span v-else class="nav-card card placeholder">
              <span class="nav-label"><span class="nav-arrow">←</span>本专题上一篇</span>
              <span class="nav-title">已是本专题第一篇</span>
            </span>
            <RouterLink v-if="seriesNext" :to="`/post/${seriesNext.slug}`" class="nav-card card next">
              <span class="nav-label">本专题下一篇<span class="nav-arrow">→</span></span>
              <span class="nav-title">{{ seriesNext.title }}</span>
            </RouterLink>
            <span v-else class="nav-card card placeholder">
              <span class="nav-label">本专题下一篇<span class="nav-arrow">→</span></span>
              <span class="nav-title">已是本专题最后一篇</span>
            </span>
          </nav>
        </div>

        <section v-if="related.length" v-reveal="{ delay: 80 }" class="related">
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

    <Lightbox :src="preview?.src ?? ''" :alt="preview?.alt" @close="closePreview" />

    <!-- 移动端目录抽屉 -->
    <Teleport to="body">
      <Transition name="toc-drawer">
        <div v-if="tocOpen" class="toc-mask" @click="tocOpen = false">
          <nav class="toc-panel" role="dialog" aria-modal="true" aria-label="文章目录" @click.stop>
            <p class="toc-panel-title">目录</p>
            <ul class="toc-panel-list">
              <li v-for="item in toc" :key="item.id">
                <a
                  :href="`#${item.id}`"
                  class="toc-panel-link"
                  :class="[`level-${item.level}`, { active: activeTocId === item.id }]"
                  @click.prevent="goToTocItem(item.id)"
                >
                  {{ item.text }}
                </a>
              </li>
            </ul>
          </nav>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { fetchPostDetail, likePost } from '@/api/post'
import { sendTrack } from '@/api/track'
import { ApiError } from '@/api/http'
import { useSiteStore } from '@/stores/site'
import { renderMarkdown, type TocItem } from '@/utils/markdown'
import { formatDate, formatNumber } from '@/utils/format'
import { isPostLiked, markPostLiked } from '@/utils/storage'
import { applyDocumentTitle } from '@/utils/title'
import { setSeo } from '@/utils/seo'
import { useMarkdownActions } from '@/composables/useMarkdownActions'
import type { PostDetail, PostNav, PostSeriesRef, PostSummary } from '@/types'
import PostToc from '@/components/post/PostToc.vue'
import CommentSection from '@/components/post/CommentSection.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'
import Lightbox from '@/components/common/Lightbox.vue'
import ReadingProgress from '@/components/common/ReadingProgress.vue'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const site = useSiteStore()
const toast = useToast()
const { preview, onContentClick, closePreview } = useMarkdownActions()

const slug = computed(() => String(route.params.slug))

const post = ref<PostDetail | null>(null)
const prev = ref<PostNav | null>(null)
const next = ref<PostNav | null>(null)
const related = ref<PostSummary[]>([])
const series = ref<PostSeriesRef | null>(null)
const seriesPrev = ref<PostNav | null>(null)
const seriesNext = ref<PostNav | null>(null)
const html = ref('')
const toc = ref<TocItem[]>([])
const loading = ref(true)
const error = ref('')
const notFound = ref(false)
const liked = ref(false)
const likeCount = ref(0)
const tocOpen = ref(false)
const activeTocId = ref('')
const bursting = ref(false)
let burstTimer: ReturnType<typeof setTimeout> | null = null

// 骨架行宽节奏，避免每行等宽的呆板感
function lineWidth(index: number): string {
  const widths = ['100%', '92%', '78%', '100%', '86%', '95%', '64%', '40%']
  return widths[(index - 1) % widths.length]
}

/** 阅读时长（分钟）：中文按 400 字/分钟，其余按 200 词/分钟，至少 1 分钟 */
const readingMinutes = computed<number>(() => {
  const text = (post.value?.content ?? '')
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`[^`]*`/g, ' ')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
  const cjk = (text.match(/[\u4e00-\u9fff]/g) ?? []).length
  const words = text
    .replace(/[\u4e00-\u9fff]/g, ' ')
    .split(/\s+/)
    .filter(Boolean).length
  return Math.max(1, Math.round(cjk / 400 + words / 200))
})

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  try {
    const data = await fetchPostDetail(slug.value)
    post.value = data.post
    // 文章页埋点：加载成功才计数，并带上 postId 供单篇分析归集
    sendTrack({ path: route.path, postId: data.post.id })
    prev.value = data.prev ?? null
    next.value = data.next ?? null
    related.value = data.related ?? []
    series.value = data.series ?? null
    seriesPrev.value = data.seriesPrev ?? null
    seriesNext.value = data.seriesNext ?? null
    const rendered = renderMarkdown(data.post.content ?? '')
    html.value = rendered.html
    toc.value = rendered.toc
    likeCount.value = data.post.likeCount ?? 0
    liked.value = isPostLiked(data.post.id)
    applyDocumentTitle(data.post.title)
    setSeo({
      description: data.post.summary,
      keywords: data.post.tags.map((t) => t.name).join(','),
      ogType: 'article',
      article: {
        headline: data.post.title,
        publishedAt: data.post.publishedAt || data.post.createdAt,
        modifiedAt: data.post.updatedAt,
        author: site.settings.siteName,
        cover: data.post.cover,
      },
    })
  } catch (e) {
    if (e instanceof ApiError && e.code === 10004) {
      notFound.value = true
    } else {
      error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
    }
    applyDocumentTitle('文章')
    setSeo({})
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
    // 心跳 + 粒子迸发只播一次，播完即收
    bursting.value = true
    if (burstTimer) clearTimeout(burstTimer)
    burstTimer = setTimeout(() => {
      bursting.value = false
    }, 700)
    toast.success('感谢点赞！')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : '点赞失败，请稍后重试')
  }
}

// 封面加载完成后淡入
const coverEl = ref<HTMLImageElement | null>(null)

function onImgLoad(): void {
  coverEl.value?.classList.add('loaded')
}

onMounted(() => {
  if (coverEl.value?.complete) coverEl.value.classList.add('loaded')
})

onBeforeUnmount(() => {
  if (burstTimer) clearTimeout(burstTimer)
})

/** 移动端目录抽屉内点击：关闭后平滑滚动到对应小节 */
function goToTocItem(id: string): void {
  tocOpen.value = false
  activeTocId.value = id
  window.setTimeout(() => {
    const el = document.getElementById(id)
    if (!el) return
    const top = el.getBoundingClientRect().top + window.scrollY - 76
    window.scrollTo({ top, behavior: 'smooth' })
  }, 220)
}

// Esc 关闭目录抽屉；路由切换时一并关闭
function onDrawerKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') tocOpen.value = false
}

onMounted(() => window.addEventListener('keydown', onDrawerKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onDrawerKeydown)
  document.body.style.overflow = ''
})

watch(tocOpen, (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
})

watch(slug, () => {
  tocOpen.value = false
  void load()
}, { immediate: true })
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
  font-size: clamp(28px, 4.2vw, 40px);
  font-weight: 650;
  line-height: 1.35;
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

/* meta 区专题链接：略强于周围灰字，hover 走品牌色 */
.series-link {
  color: var(--text-2);
}

.series-link:hover {
  color: var(--brand);
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
  opacity: 0;
  transition: opacity 0.5s ease-out;
}

/* 正文阅读列宽：不占满整屏，保证行长舒适 */
.post-content .markdown-body {
  max-width: var(--reading-width);
}

/* 移动端目录按钮（桌面隐藏，桌面用右侧吸附 TOC） */
.toc-toggle {
  display: none;
  align-items: center;
  gap: 3px;
  margin-left: auto;
  padding: 2px 10px;
  border: 1px solid var(--border-strong);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text-2);
  font-size: 12px;
  line-height: 1.7;
  transition: color var(--transition), border-color var(--transition);
}

.toc-toggle:hover {
  color: var(--brand);
  border-color: var(--brand);
}

.post-cover img.loaded {
  opacity: 1;
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

/* 点赞心跳 + 粒子迸发（一次性反馈） */
.like-heart {
  position: relative;
  display: inline-flex;
}

.like-btn.liked .like-heart svg {
  animation: heart-pop 0.45s var(--ease-out-quart);
}

@keyframes heart-pop {
  0% {
    transform: scale(1);
  }
  35% {
    transform: scale(1.35);
  }
  62% {
    transform: scale(0.9);
  }
  100% {
    transform: scale(1);
  }
}

.like-burst {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.burst-dot {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--brand);
  animation: burst 0.55s var(--ease-out-quart) 0.1s backwards;
}

@keyframes burst {
  0% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 1;
  }
  100% {
    transform: translate(
        calc(-50% + cos(var(--angle)) * 20px),
        calc(-50% + sin(var(--angle)) * 20px)
      )
      scale(0.35);
    opacity: 0;
  }
}

/* 上一篇/下一篇 */
.post-nav {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  margin-top: 30px;
}

/* 本专题导航组：小标题 + 复用全局 nav-card 结构 */
.series-nav {
  margin-top: 30px;
}

.series-nav-label {
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.12em;
  color: var(--text-3);
}

.series-nav .post-nav {
  margin-top: 0;
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

.nav-arrow {
  display: inline-block;
  margin: 0 4px;
  transition: transform var(--transition);
}

a.nav-card:hover .nav-arrow {
  transform: translateX(-2px);
}

a.nav-card.next:hover .nav-arrow {
  transform: translateX(2px);
}

.nav-title {
  font-size: 14.5px;
  font-weight: 500;
  color: var(--text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color var(--transition), transform 0.25s var(--ease-out-quart);
}

a.nav-card:hover .nav-title {
  color: var(--brand);
}

a.nav-card:not(.next):hover .nav-title {
  transform: translateX(-3px);
}

a.nav-card.next:hover .nav-title {
  transform: translateX(3px);
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
  transition: border-color var(--transition), box-shadow 0.2s ease-out,
    transform 0.2s var(--ease-out-quart);
}

.related-card:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
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

@media (max-width: 1199px) {
  .toc-toggle {
    display: inline-flex;
  }
}

/* 移动端目录抽屉 */
.toc-mask {
  position: fixed;
  inset: 0;
  z-index: 900;
  background: rgba(0, 0, 0, 0.4);
}

.toc-panel {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  max-height: 68vh;
  overflow-y: auto;
  padding: 18px 20px 26px;
  border-radius: 16px 16px 0 0;
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.toc-panel-title {
  padding-bottom: 10px;
  margin-bottom: 6px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.12em;
  color: var(--text-3);
}

.toc-panel-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.toc-panel-link {
  display: block;
  padding: 9px 4px;
  color: var(--text-2);
  font-size: 14.5px;
  line-height: 1.6;
}

.toc-panel-link.level-3 {
  padding-left: 20px;
}

.toc-panel-link.level-4 {
  padding-left: 36px;
}

.toc-panel-link:hover,
.toc-panel-link.active {
  color: var(--brand);
}

.toc-drawer-enter-active,
.toc-drawer-leave-active {
  transition: opacity 0.2s ease-out;
}

.toc-drawer-enter-active .toc-panel,
.toc-drawer-leave-active .toc-panel {
  transition: transform 0.22s var(--ease-out-quart);
}

.toc-drawer-enter-from,
.toc-drawer-leave-to {
  opacity: 0;
}

.toc-drawer-enter-from .toc-panel,
.toc-drawer-leave-to .toc-panel {
  transform: translateY(40%);
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
