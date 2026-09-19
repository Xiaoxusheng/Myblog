<template>
  <div class="reading-progress" aria-hidden="true">
    <div class="reading-progress-bar" :style="{ transform: `scaleX(${progress})` }"></div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const progress = ref(0)
let ticking = false

function update(): void {
  const doc = document.documentElement
  const scrollable = doc.scrollHeight - window.innerHeight
  progress.value = scrollable > 0 ? Math.min(1, window.scrollY / scrollable) : 0
}

function onScroll(): void {
  if (ticking) return
  ticking = true
  window.requestAnimationFrame(() => {
    ticking = false
    update()
  })
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll, { passive: true })
  update()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
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
  background: linear-gradient(90deg, var(--brand), var(--brand-hover));
  transition: transform 0.08s linear;
}
</style>
