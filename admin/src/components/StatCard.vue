<script setup lang="ts">
import type { Component } from 'vue'

/**
 * 统计卡片：图标块 + 标题 + 大数字 + 辅助信息（数据一律来自 /admin/stats 真实字段）
 */
defineProps<{
  title: string
  value: number | string
  icon: Component
  /** 辅助信息（真实字段派生，如"待审核 2 条"） */
  hint?: string
  /** 告警态（如待审 > 0）：数字与图标用 warning 色 */
  warning?: boolean
}>()
</script>

<template>
  <div class="stat-card">
    <div class="stat-card__icon" :class="{ 'stat-card__icon--warning': warning }">
      <component :is="icon" />
    </div>
    <div class="stat-card__meta">
      <div class="stat-card__title">{{ title }}</div>
      <div class="stat-card__value tabular-nums" :class="{ 'stat-card__value--warning': warning }">
        {{ value }}
      </div>
      <div v-if="hint" class="stat-card__hint">{{ hint }}</div>
    </div>
  </div>
</template>

<style scoped>
.stat-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-sm);
}

.stat-card__icon {
  flex: none;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: var(--admin-brand);
  background: var(--admin-brand-bg);
  border-radius: var(--admin-radius-md);
}

.stat-card__icon--warning {
  color: var(--admin-warning);
  background: var(--admin-warning-bg);
}

.stat-card__meta {
  min-width: 0;
}

.stat-card__title {
  font-size: 13px;
  line-height: 20px;
  color: var(--admin-muted);
}

.stat-card__value {
  margin-top: 2px;
  font-size: 24px;
  font-weight: 600;
  line-height: 30px;
  color: var(--admin-text);
}

.stat-card__value--warning {
  color: var(--admin-warning-text);
}

.stat-card__hint {
  margin-top: 2px;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}
</style>
