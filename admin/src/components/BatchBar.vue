<script setup lang="ts">
/**
 * 表格批量操作条（docs/09 §6.2 / v2 设计稿第 4 页）。
 *
 * 形态：**底部浮出**（表格下沿），左「已选 N 篇」、右动作组 + 取消选择。
 * 设计稿里批量条压在表格底部而非工具栏下方——因为选中是"对当前列表的动作"，
 * 挂在列表尾部比挤在筛选行更贴近被操作对象。
 *
 * 评论/文章等带 rowSelection 的表格共用。
 */
withDefaults(
  defineProps<{
    count: number
    /** 计数单位：文章用「篇」，评论用「条」 */
    unit?: string
  }>(),
  { unit: '项' },
)

const emit = defineEmits<{ clear: [] }>()
</script>

<template>
  <div class="batch-bar">
    <span class="batch-bar__count">
      已选 <b class="tabular-nums">{{ count }}</b> {{ unit }}
    </span>
    <div class="batch-bar__actions">
      <slot />
      <a-button size="small" type="text" @click="emit('clear')">取消选择</a-button>
    </div>
  </div>
</template>

<style scoped>
.batch-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  padding: 10px 16px;
  margin-top: 12px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
  /* 浮出感：从下方轻微上移进入，不弹跳 */
  animation: batch-bar-in 0.18s var(--admin-ease) both;
}

@keyframes batch-bar-in {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .batch-bar {
    animation: none;
  }
}

.batch-bar__count {
  font-size: 13px;
  color: var(--admin-text);
}

.batch-bar__count b {
  font-weight: 600;
}

.batch-bar__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
