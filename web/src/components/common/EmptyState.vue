<template>
  <div class="empty-state" :class="{ compact: size === 'compact' }">
    <svg
      v-if="size !== 'compact'"
      class="empty-art"
      viewBox="0 0 200 120"
      width="176"
      height="106"
      fill="none"
      aria-hidden="true"
    >
      <circle cx="100" cy="60" r="52" stroke="var(--border)" stroke-width="1.5" />
      <rect
        x="64"
        y="40"
        width="72"
        height="48"
        rx="6"
        stroke="var(--text-3)"
        stroke-width="1.5"
      />
      <path
        d="M64 48l36 24 36-24"
        stroke="var(--text-3)"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <p class="empty-title">{{ title }}</p>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <div v-if="$slots.default" class="empty-action">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string
    description?: string
    /** compact：窄栏/区块内轻量空态，无插画（docs/08 §6.1） */
    size?: 'default' | 'compact'
  }>(),
  {
    title: '暂无内容',
    description: '',
    size: 'default'
  }
)
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 56px 20px;
  text-align: center;
}

.empty-state.compact {
  padding: 26px 16px;
}

.empty-title {
  margin-top: 18px;
  font-size: 15.5px;
  font-weight: 500;
  color: var(--text-2);
}

.compact .empty-title {
  margin-top: 0;
  font-size: 13px;
  font-weight: 400;
  color: var(--text-3);
}

.empty-desc {
  margin-top: 6px;
  font-size: 13.5px;
  color: var(--text-3);
}

.empty-action {
  margin-top: 18px;
}
</style>
