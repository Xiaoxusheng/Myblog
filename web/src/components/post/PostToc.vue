<template>
  <nav v-if="headings.length" class="post-toc" aria-label="文章目录">
    <p class="toc-title">目录</p>
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
  </nav>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { TocItem } from '@/utils/markdown'

const props = defineProps<{ headings: TocItem[] }>()

const activeId = ref('')
let ticking = false

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
  updateActive()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>

<style scoped>
/* 文字栏 TOC：无卡片，active 走左竖线 + accent */
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

.toc-list {
  list-style: none;
  margin: 6px 0 0;
  padding: 0;
}

.toc-link {
  display: block;
  padding: 5px 0 5px 12px;
  border-left: 2px solid transparent;
  color: var(--text-3);
  font-size: 13px;
  line-height: 1.55;
  transition: color var(--transition), border-color var(--transition);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.toc-link:hover {
  color: var(--text-1);
}

.toc-link.active {
  color: var(--brand);
  border-left-color: var(--brand);
  font-weight: 500;
}

.toc-link.level-3 {
  padding-left: 24px;
}

.toc-link.level-4 {
  padding-left: 36px;
}
</style>
