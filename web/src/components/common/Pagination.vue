<template>
  <nav v-if="totalPages > 1" class="pagination" aria-label="分页">
    <button class="page-btn" :disabled="page <= 1" @click="go(page - 1)">上一页</button>
    <template v-for="(item, index) in pages" :key="`${item}-${index}`">
      <span v-if="item === '…'" class="page-ellipsis">…</span>
      <button
        v-else
        class="page-btn page-num"
        :class="{ current: item === page }"
        :aria-current="item === page ? 'page' : undefined"
        @click="go(item)"
      >
        {{ item }}
      </button>
    </template>
    <button class="page-btn" :disabled="page >= totalPages" @click="go(page + 1)">下一页</button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ page: number; totalPages: number }>()

const emit = defineEmits<{ change: [page: number] }>()

const pages = computed<(number | '…')[]>(() => {
  const total = props.totalPages
  const cur = props.page
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  const result: (number | '…')[] = [1]
  const start = Math.max(2, cur - 1)
  const end = Math.min(total - 1, cur + 1)
  if (start > 2) result.push('…')
  for (let i = start; i <= end; i++) result.push(i)
  if (end < total - 1) result.push('…')
  result.push(total)
  return result
})

function go(target: number): void {
  const next = Math.min(Math.max(target, 1), props.totalPages)
  if (next === props.page) return
  emit('change', next)
}
</script>

<style scoped>
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 32px;
}

.page-btn {
  min-width: 34px;
  height: 34px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-2);
  font-size: 13.5px;
  line-height: 1;
  transition: border-color var(--transition), color var(--transition),
    background var(--transition);
}

.page-btn:hover:not(:disabled):not(.current) {
  border-color: var(--brand);
  color: var(--brand);
}

.page-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.page-btn.current {
  background: var(--brand);
  border-color: var(--brand);
  color: #fff;
  font-weight: 500;
}

.page-ellipsis {
  padding: 0 2px;
  color: var(--text-3);
}
</style>
