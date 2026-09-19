<script setup lang="ts">
/**
 * 表格批量操作条(docs/09 §6.2):已选计数 + 动作插槽 + 取消选择
 * 评论/文章等带 rowSelection 的表格共用
 */
defineProps<{ count: number }>()

const emit = defineEmits<{ clear: [] }>()
</script>

<template>
  <div class="batch-bar">
    <a-space wrap>
      <span class="batch-bar__count">已选 {{ count }} 项</span>
      <slot />
      <a-button size="small" type="text" @click="emit('clear')">取消选择</a-button>
    </a-space>
  </div>
</template>

<style scoped>
.batch-bar {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  margin-bottom: 12px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-sm);
  background: var(--admin-surface-2);
}

.batch-bar__count {
  margin-right: 4px;
  font-size: 13px;
  color: var(--admin-muted);
}
</style>
