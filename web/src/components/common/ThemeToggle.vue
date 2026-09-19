<template>
  <button class="icon-btn" :title="title" :aria-label="title" @click="theme.cycle()">
    <Transition name="theme-icon" mode="out-in">
      <!-- 浅色：太阳 -->
      <svg
        v-if="theme.mode === 'light'"
        key="light"
        viewBox="0 0 24 24"
        width="18"
        height="18"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        aria-hidden="true"
      >
        <circle cx="12" cy="12" r="4" />
        <path
          d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"
        />
      </svg>
      <!-- 深色：月亮 -->
      <svg
        v-else-if="theme.mode === 'dark'"
        key="dark"
        viewBox="0 0 24 24"
        width="18"
        height="18"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
      </svg>
      <!-- 跟随系统：半填充圆 -->
      <svg
        v-else
        key="auto"
        viewBox="0 0 24 24"
        width="18"
        height="18"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        aria-hidden="true"
      >
        <circle cx="12" cy="12" r="9" />
        <path d="M12 3a9 9 0 0 1 0 18z" fill="currentColor" stroke="none" />
      </svg>
    </Transition>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'

const theme = useThemeStore()

const title = computed(() => {
  const name =
    theme.mode === 'auto' ? '跟随系统' : theme.mode === 'light' ? '浅色模式' : '深色模式'
    return `当前：${name}，点击切换`
  })
</script>

<style scoped>
.theme-icon-enter-active,
.theme-icon-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.theme-icon-enter-from {
  opacity: 0;
  transform: rotate(-45deg) scale(0.6);
}

.theme-icon-leave-to {
  opacity: 0;
  transform: rotate(45deg) scale(0.6);
}
</style>
