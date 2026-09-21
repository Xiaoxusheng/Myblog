<template>
  <div class="seg-tabs" role="tablist">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      type="button"
      role="tab"
      class="seg-tab"
      :class="{ active: modelValue === tab.value }"
      :aria-selected="modelValue === tab.value"
      @click="emit('update:modelValue', tab.value)"
    >
      {{ tab.label }}
      <span v-if="tab.count != null" class="seg-count">{{ tab.count }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  tabs: { value: string; label: string; count?: number }[]
  modelValue: string
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<style scoped>
/* 分段 Tab：胶囊容器 + 选中项白底凸起（p06） */
.seg-tabs {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border-radius: var(--radius-pill);
  background: var(--surface);
}

.seg-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 18px;
  border: none;
  border-radius: var(--radius-pill);
  background: transparent;
  color: var(--text-2);
  font-size: var(--fs-md);
  line-height: 1.2;
  transition: background var(--transition), color var(--transition),
    box-shadow var(--transition);
}

.seg-tab:hover {
  color: var(--text-1);
}

.seg-tab.active {
  background: var(--bg);
  color: var(--text-1);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}

.seg-count {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-weight: 400;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}
</style>
