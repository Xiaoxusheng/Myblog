<template>
  <Teleport to="body">
    <Transition name="palette">
      <div v-if="isOpen" class="palette-overlay" @click.self="close">
        <div class="palette" role="dialog" aria-modal="true" aria-label="站内搜索">
          <div class="palette-input-row">
            <svg class="palette-icon" viewBox="0 0 24 24" width="16" height="16" fill="currentColor" aria-hidden="true">
              <path d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 1 0-.7.7l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0A4.5 4.5 0 1 1 14 9.5 4.5 4.5 0 0 1 9.5 14z" />
            </svg>
            <input
              ref="inputEl"
              v-model="keyword"
              type="text"
              class="palette-input"
              placeholder="搜索文章…"
              aria-label="搜索文章"
              aria-controls="palette-listbox"
              :aria-activedescendant="activeId"
              @keydown.down.prevent="move(1)"
              @keydown.up.prevent="move(-1)"
              @keydown.enter.prevent="goActive"
            />
            <span class="kbd">Esc</span>
          </div>

          <div id="palette-listbox" class="palette-body" role="listbox" aria-label="搜索结果">
            <!-- Loading：两行骨架 -->
            <div v-if="loading" class="palette-skel" aria-hidden="true">
              <div v-for="i in 2" :key="i" class="palette-skel-row">
                <span class="skeleton" style="height: 14px; width: 58%"></span>
                <span class="skeleton" style="height: 11px; width: 30%"></span>
              </div>
            </div>

            <!-- 有结果 / 快捷导航 -->
            <template v-else-if="rows.length">
              <p class="palette-group">{{ groupLabel }}</p>
              <button
                v-for="(row, i) in rows"
                :id="`palette-opt-${i}`"
                :key="row.to"
                type="button"
                class="palette-row"
                role="option"
                :aria-selected="i === activeIndex"
                :class="{ active: i === activeIndex }"
                @click="go(row)"
                @mousemove="activeIndex = i"
              >
                <span v-if="row.num" class="palette-row-num">{{ row.num }}</span>
                <span class="palette-row-title">{{ row.title }}</span>
                <span v-if="row.meta" class="palette-row-meta">{{ row.meta }}</span>
              </button>
            </template>

            <!-- 关键词无结果 / 出错：引导去搜索页 -->
            <div v-else class="palette-empty">
              <p>{{ loadError ? '搜索暂时不可用，请稍后再试。' : `没有找到与「${keyword}」相关的文章。` }}</p>
              <RouterLink v-if="keyword && !loadError" :to="`/search?keyword=${encodeURIComponent(keyword)}`" @click="close">
                在搜索页查看「{{ keyword }}」→
              </RouterLink>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { fetchPosts } from '@/api/post'
import type { PostSummary } from '@/types'
import { formatDate } from '@/utils/format'
import { useCommandPalette } from '@/composables/useCommandPalette'

interface PaletteRow {
  title: string
  to: string
  num?: string
  meta?: string
}

const QUICK_NAV: PaletteRow[] = [
  { num: '01', title: '首页', to: '/' },
  { num: '02', title: '归档', to: '/archives' },
  { num: '03', title: '分类', to: '/categories' },
  { num: '04', title: '标签', to: '/tags' },
  { num: '05', title: '专题', to: '/series' },
  { num: '06', title: '友链', to: '/links' }
]

const router = useRouter()
const { isOpen, open, close } = useCommandPalette()

const keyword = ref('')
const loading = ref(false)
const loadError = ref(false)
const posts = ref<PostSummary[]>([])
const activeIndex = ref(0)
const inputEl = ref<HTMLInputElement | null>(null)

const rows = computed<PaletteRow[]>(() => {
  if (!keyword.value.trim()) return QUICK_NAV
  return posts.value.map((p) => ({
    title: p.title,
    to: `/post/${p.slug}`,
    meta: [formatDate(p.publishedAt || p.createdAt), p.category?.name].filter(Boolean).join(' · ')
  }))
})

const groupLabel = computed(() =>
  keyword.value.trim() ? `文章 · ${rows.value.length} 条结果` : '快捷导航'
)

const activeId = computed(() =>
  rows.value.length ? `palette-opt-${activeIndex.value}` : undefined
)

let debounceTimer: ReturnType<typeof setTimeout> | null = null
let abortCtrl: AbortController | null = null

function search(kw: string): void {
  abortCtrl?.abort()
  abortCtrl = null
  if (!kw) {
    loading.value = false
    loadError.value = false
    posts.value = []
    return
  }
  loading.value = true
  loadError.value = false
  const ctrl = new AbortController()
  abortCtrl = ctrl
  fetchPosts({ page: 1, pageSize: 8, keyword: kw }, ctrl.signal)
    .then((data) => {
      if (ctrl.signal.aborted) return
      posts.value = data.list ?? []
    })
    .catch((err: unknown) => {
      if (ctrl.signal.aborted) return
      posts.value = []
      loadError.value = true
      console.warn('[palette] search failed', err)
    })
    .finally(() => {
      if (ctrl.signal.aborted) return
      loading.value = false
    })
}

watch(keyword, (val) => {
  activeIndex.value = 0
  if (debounceTimer) clearTimeout(debounceTimer)
  const kw = val.trim()
  if (!kw) {
    search('')
    return
  }
  loading.value = true
  debounceTimer = setTimeout(() => search(kw), 250)
})

watch(isOpen, async (visible) => {
  // 滚动锁：面板打开期间锁定页面，防止背景滚动
  document.documentElement.style.overflow = visible ? 'hidden' : ''
  if (visible) {
    keyword.value = ''
    activeIndex.value = 0
    posts.value = []
    loadError.value = false
    loading.value = false
    await nextTick()
    inputEl.value?.focus()
  } else if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
})

function move(delta: number): void {
  if (!rows.value.length) return
  const len = rows.value.length
  activeIndex.value = (activeIndex.value + delta + len) % len
}

function goActive(): void {
  const row = rows.value[activeIndex.value]
  if (row) go(row)
}

function go(row: PaletteRow): void {
  close()
  void router.push(row.to)
}

function onGlobalKey(e: KeyboardEvent): void {
  if ((e.metaKey || e.ctrlKey) && e.code === 'KeyK') {
    e.preventDefault()
    if (isOpen.value) close()
    else open()
  } else if (e.key === 'Escape' && isOpen.value) {
    close()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKey)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKey)
  document.documentElement.style.overflow = ''
  if (debounceTimer) clearTimeout(debounceTimer)
  abortCtrl?.abort()
})
</script>

<style scoped>
.palette-overlay {
  position: fixed;
  inset: 0;
  z-index: 1050;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 14vh 20px 20px;
  background: rgba(20, 20, 19, 0.32);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

html.dark .palette-overlay {
  background: rgba(0, 0, 0, 0.5);
}

.palette {
  width: 100%;
  max-width: 560px;
  max-height: 62vh;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

.palette-input-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  height: 52px;
  padding: 0 16px;
  border-bottom: 1px solid var(--border);
}

.palette-icon {
  flex-shrink: 0;
  color: var(--text-3);
}

.palette-input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  font-size: 15px;
  color: var(--text-1);
}

.palette-input::placeholder {
  color: var(--text-3);
}

.palette-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.palette-group {
  padding: 8px 10px 6px;
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
}

.palette-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 48px;
  padding: 8px 10px;
  border: none;
  border-left: 2px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  text-align: left;
}

/* 选中态：与 TOC active 同语言（surface-2 底 + accent 竖线） */
.palette-row.active {
  background: var(--surface-2);
  border-left-color: var(--brand);
}

.palette-row-num {
  flex-shrink: 0;
  width: 22px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.palette-row-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  color: var(--text-1);
}

.palette-row-meta {
  flex-shrink: 0;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.palette-skel {
  padding: 8px 10px;
}

.palette-skel-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 0;
}

.palette-skel-row + .palette-skel-row {
  border-top: 1px solid var(--border);
}

.palette-empty {
  padding: 28px 12px;
  text-align: center;
  font-size: 13.5px;
  color: var(--text-2);
}

.palette-empty a {
  display: inline-block;
  margin-top: 8px;
  font-size: 13px;
}

/* ---------- 进出场 ---------- */
.palette-enter-active {
  transition: opacity 0.18s ease-out;
}

.palette-enter-active .palette {
  transition: opacity 0.18s ease-out, transform 0.18s var(--ease-out-quart);
}

.palette-leave-active {
  transition: opacity 0.12s ease-out;
}

.palette-enter-from,
.palette-leave-to {
  opacity: 0;
}

.palette-enter-from .palette {
  opacity: 0;
  transform: translateY(6px);
}

/* ---------- 移动端：全屏化 ---------- */
@media (max-width: 767px) {
  .palette-overlay {
    padding: 0;
  }

  .palette {
    max-width: none;
    max-height: none;
    height: 100%;
    border: none;
    border-radius: 0;
  }
}
</style>
