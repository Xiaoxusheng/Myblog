<template>
  <div class="reading-progress" aria-hidden="true">
    <div class="reading-progress-bar" :style="{ transform: `scaleX(${progress})` }"></div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { saveReadingPosition } from '@/utils/storage'

/** 传入 slug 时同时持久化阅读位置（比例 + 完成状态），供下次访问恢复 */
const props = defineProps<{ slug?: string }>()

const progress = ref(0)
let ticking = false
let lastSavedAt = 0
let lastSavedProgress = -1
let lastDone = false

// 换文章时重置节流基线，避免上一篇的进度影响新文章的首次保存
watch(
  () => props.slug,
  () => {
    lastSavedAt = 0
    lastSavedProgress = -1
    lastDone = false
  }
)

/** 完成判定：滚动到接近文末（95%）即视为读完 */
const DONE_RATIO = 0.95
/** 持久化节流：至多 1.2s 一次，且进度变化不足 2% 不写，避免高频滚动下的同步 IO */
const SAVE_INTERVAL_MS = 1200
const SAVE_MIN_DELTA = 0.02

function persist(force: boolean): void {
  const slug = props.slug
  if (!slug) return
  const now = Date.now()
  // 完成判定：比例达到 95%，或视口已到达文档底部附近
  //（文章详情页底部还有相关文章/评论区/页脚，仅按比例判定可能永远到不了 95%）
  const doc = document.documentElement
  const atBottom = window.innerHeight + window.scrollY >= doc.scrollHeight - 80
  const done = atBottom || progress.value >= DONE_RATIO
  // 完成态首次达成时放行节流，确保 d=true 及时落盘（瞬间跳底后静止的场景）
  const doneJustReached = done && !lastDone
  lastDone = done
  if (!force && !doneJustReached && now - lastSavedAt < SAVE_INTERVAL_MS) return
  if (!force && !doneJustReached && Math.abs(progress.value - lastSavedProgress) < SAVE_MIN_DELTA) return
  lastSavedAt = now
  lastSavedProgress = progress.value
  saveReadingPosition(slug, progress.value, done)
}

function update(): void {
  const doc = document.documentElement
  const scrollable = doc.scrollHeight - window.innerHeight
  progress.value = scrollable > 0 ? Math.min(1, window.scrollY / scrollable) : 0
  persist(false)
}

function onScroll(): void {
  if (ticking) return
  ticking = true
  window.requestAnimationFrame(() => {
    ticking = false
    update()
  })
}

function onPageHide(): void {
  persist(true)
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll, { passive: true })
  window.addEventListener('pagehide', onPageHide)
  update()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
  window.removeEventListener('pagehide', onPageHide)
  persist(true)
})
</script>

<style scoped>
.reading-progress {
  position: fixed;
  top: var(--header-height);
  left: 0;
  right: 0;
  z-index: 99;
  height: 2px;
  pointer-events: none;
  background: transparent;
}

.reading-progress-bar {
  height: 100%;
  transform-origin: 0 50%;
  transform: scaleX(0);
  background: var(--brand);
  transition: transform 0.08s linear;
}
</style>
