<template>
  <div class="error-state" :class="{ small }">
    <svg
      class="error-art"
      viewBox="0 0 200 120"
      width="132"
      height="80"
      fill="none"
      aria-hidden="true"
    >
      <circle cx="100" cy="60" r="48" stroke="var(--border)" stroke-width="1.5" />
      <path
        d="M100 32l38 62H62z"
        stroke="var(--text-3)"
        stroke-width="1.5"
        stroke-linejoin="round"
      />
      <path
        d="M100 54v20"
        stroke="var(--text-3)"
        stroke-width="2.4"
        stroke-linecap="round"
      />
      <circle cx="100" cy="84" r="1.8" fill="var(--text-3)" />
    </svg>
    <p class="error-title">加载失败</p>
    <p class="error-desc">{{ message }}</p>
    <button v-if="!small" class="btn" @click="emit('retry')">重试</button>
    <button v-else class="retry-link" @click="emit('retry')">重试</button>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ message?: string; small?: boolean }>(), {
  message: '出错了，请稍后重试',
  small: false
})

const emit = defineEmits<{ retry: [] }>()
</script>

<style scoped>
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 56px 20px;
  text-align: center;
}

.error-state.small {
  padding: 20px 8px;
}

.error-title {
  margin-top: 12px;
  font-size: var(--fs-base);
  font-weight: 500;
  color: var(--text-2);
}

.error-desc {
  margin-top: 6px;
  margin-bottom: 16px;
  font-size: var(--fs-sm);
  color: var(--text-3);
}

.error-state.small .error-title {
  display: none;
}

.error-state.small .error-desc {
  margin-bottom: 6px;
}

.retry-link {
  border: none;
  background: transparent;
  color: var(--brand);
  font-size: var(--fs-sm);
  padding: 0;
}

.retry-link:hover {
  color: var(--brand-hover);
}
</style>
