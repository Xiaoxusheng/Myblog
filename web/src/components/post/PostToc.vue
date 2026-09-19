<template>
  <nav v-if="headings.length" class="post-toc" aria-label="文章目录">
    <p class="toc-title">目录</p>
    <div ref="bodyEl" class="toc-body">
      <span
        class="toc-rail"
        aria-hidden="true"
        :style="{ top: `${railTop}px`, height: `${railHeight}px` }"
      ></span>
      <ul class="toc-list">
        <li v-for="item in headings" :key="item.id">
          <a
            :href="`#${item.id}`"
            class="toc-link"
            :class="[`level-${item.level}`, { active: activeId === item.id }]"
            @click.prevent="goTo(item.id)"
          >
            {{ item.text }}
          </a>
        </li>
      </ul>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { TocItem } from '@/utils/markdown'

const props = defineProps<{ headings: TocItem[] }>()

const activeId = ref('')
const bodyEl = ref<HTMLElement | null>(null)
/** accent 进度段：从首项顶部到当前 active 项底部，随滚动生长（docs/07 §7.1） */
const railTop = ref(0)
const railHeight = ref(0)
let ticking = false

function updateRail(): void {
  const list = bodyEl.value
  if (!list) return
  const links = list.querySelectorAll<HTMLAnchorElement>('.toc-link')
  if (!links.length) return
  const listRect = list.getBoundingClientRect()
  const first = links[0].getBoundingClientRect()
  if (!activeId.value) {
    railHeight.value = 0
    return
  }
  const active = list.querySelector<HTMLElement>('.toc-link.active')
  if (!active) {
    railHeight.value = 0
    return
  }
  const activeRect = active.getBoundingClientRect()
  railTop.value = first.top - listRect.top
  railHeight.value = activeRect.bottom - first.top
}

function updateActive(): void {
  const offset = window.scrollY + 96
  let current = ''
  for (const item of props.headings) {
    const el = document.getElementById(item.id)
    if (!el) continue
    if (el.getBoundingClientRect().top + window.scrollY <= offset) {
      current = item.id
    } else {
      break
    }
  }
  activeId.value = current
  updateRail()
}

function onScroll(): void {
  if (ticking) return
  ticking = true
  window.requestAnimationFrame(() => {
    ticking = false
    updateActive()
  })
}

function goTo(id: string): void {
  const el = document.getElementById(id)
  if (!el) return
  const top = el.getBoundingClientRect().top + window.scrollY - 76
  window.scrollTo({ top, behavior: 'smooth' })
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll, { passive: true })
  updateActive()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
})
</script>

<style scoped>
/* 文字栏 TOC：无卡片；连续灰轨 + accent 进度段表达「读到哪里」（docs/07 §7.1） */
.post-toc {
  position: sticky;
  top: calc(var(--header-height) + 24px);
  max-height: calc(100vh - var(--header-height) - 48px);
  overflow-y: auto;
}

.toc-title {
  padding-bottom: 10px;
  margin-bottom: 6px;
  border-bottom: 1px solid var(--border);
  font-family: var(--font-mono);
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.14em;
  color: var(--text-3);
}

.toc-body {
  position: relative;
  margin-top: 6px;
}

/* 连续灰轨：上下各收进 2px，与标题 hairline 呼应 */
.toc-body::before {
  content: '';
  position: absolute;
  top: 2px;
  bottom: 2px;
  left: 0;
  width: 2px;
  border-radius: 1px;
  background: var(--border);
}

/* accent 进度段：跟随 active 项生长 */
.toc-rail {
  position: absolute;
  left: 0;
  width: 2px;
  border-radius: 1px;
  background: var(--brand);
  transition: top 0.25s var(--ease-out-quart), height 0.25s var(--ease-out-quart);
}

.toc-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.toc-link {
  display: block;
  padding: 5px 0 5px 12px;
  color: var(--text-3);
  font-size: 13px;
  line-height: 1.55;
  transition: color var(--transition);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.toc-link:hover {
  color: var(--text-1);
}

.toc-link.active {
  color: var(--brand);
  font-weight: 500;
}

.toc-link.level-3 {
  padding-left: 24px;
}

.toc-link.level-4 {
  padding-left: 36px;
}
</style>
