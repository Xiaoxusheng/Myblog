<template>
  <Transition name="backtop">
    <button
      v-if="visible"
      class="backtop"
      title="回到顶部"
      aria-label="回到顶部"
      @click="toTop"
    >
      <svg class="ring" viewBox="0 0 40 40" aria-hidden="true">
        <circle class="ring-track" cx="20" cy="20" r="16" />
        <circle
          class="ring-bar"
          cx="20"
          cy="20"
          r="16"
          :style="{ strokeDashoffset: ringOffset }"
        />
      </svg>
      <svg
        class="arrow"
        viewBox="0 0 24 24"
        width="15"
        height="15"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M12 19V5M5 12l7-7 7 7" />
      </svg>
    </button>
  </Transition>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const SHOW_AT = 480
const visible = ref(false)
const progress = ref(0)
let ticking = false

const ringOffset = computed(() => {
  const circumference = 2 * Math.PI * 16
  return String(circumference * (1 - progress.value))
})

function update(): void {
  const doc = document.documentElement
  const scrollable = doc.scrollHeight - window.innerHeight
  progress.value = scrollable > 0 ? Math.min(1, window.scrollY / scrollable) : 0
  visible.value = window.scrollY > SHOW_AT
}

function onScroll(): void {
  if (ticking) return
  ticking = true
  window.requestAnimationFrame(() => {
    ticking = false
    update()
  })
}

function toTop(): void {
  const reduced =
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  update()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>

<style scoped>
.backtop {
  position: fixed;
  right: 22px;
  bottom: 26px;
  z-index: 90;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: var(--surface);
  color: var(--text-2);
  box-shadow: var(--shadow-md);
  transition: color var(--transition), transform var(--transition),
    box-shadow var(--transition);
}

.backtop:hover {
  color: var(--brand);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.backtop:active {
  transform: translateY(0);
}

.ring {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.ring circle {
  fill: none;
  stroke-width: 2.5;
}

.ring-track {
  stroke: var(--border);
}

.ring-bar {
  stroke: var(--brand);
  stroke-linecap: round;
  stroke-dasharray: 100.53;
  transition: stroke-dashoffset 0.1s linear;
}

.arrow {
  position: relative;
}

.backtop-enter-active,
.backtop-leave-active {
  transition: opacity 0.22s ease-out, transform 0.22s var(--ease-out-quart);
}

.backtop-enter-from,
.backtop-leave-to {
  opacity: 0;
  transform: translateY(10px) scale(0.9);
}
</style>
