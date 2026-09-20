<script setup lang="ts">
/**
 * 发布前检查清单（模块五）。
 *
 * 定位是「提醒」而非「拦截」：列出会被读者/搜索引擎直接看到的缺失项，用户可
 * 逐项定位回去补，也可以直接确认发布。不引入虚假评分，每项都可明确验证。
 */
import { computed } from 'vue'

export interface PrePublishCheckItem {
  key: string
  label: string
  detail?: string
  ok: boolean
}

const props = withDefaults(
  defineProps<{
    open: boolean
    checks: PrePublishCheckItem[]
    confirmText?: string
  }>(),
  { confirmText: '确认发布' },
)

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'publish'): void
  (e: 'locate', key: string): void
}>()

const visible = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value),
})

const failed = computed(() => props.checks.filter((c) => !c.ok))
const passed = computed(() => props.checks.filter((c) => c.ok))
</script>

<template>
  <a-modal
    v-model:open="visible"
    title="发布前检查"
    width="520px"
    :ok-text="confirmText"
    cancel-text="返回修改"
    @ok="emit('publish')"
  >
    <p class="precheck__lead">
      <template v-if="failed.length === 0">
        所有检查项均已通过，可以直接发布。
      </template>
      <template v-else>
        有 {{ failed.length }} 项建议在发布前完善（不阻止发布）：
      </template>
    </p>

    <ul class="precheck__list">
      <li v-for="item in failed" :key="item.key" class="precheck__item precheck__item--warn">
        <span class="precheck__icon">⚠</span>
        <div class="precheck__text">
          <span class="precheck__label">{{ item.label }}</span>
          <span v-if="item.detail" class="precheck__detail">{{ item.detail }}</span>
        </div>
        <a-button type="link" size="small" @click="emit('locate', item.key)">定位</a-button>
      </li>
      <li v-for="item in passed" :key="item.key" class="precheck__item">
        <span class="precheck__icon precheck__icon--ok">✓</span>
        <div class="precheck__text">
          <span class="precheck__label">{{ item.label }}</span>
        </div>
      </li>
    </ul>
  </a-modal>
</template>

<style scoped>
.precheck__lead {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--admin-muted);
}

.precheck__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
  max-height: 46vh;
  overflow-y: auto;
}

.precheck__item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 7px 8px;
  border-radius: var(--admin-radius-sm);
  font-size: 13px;
  line-height: 20px;
}

.precheck__item--warn {
  background: var(--admin-warning-bg);
}

.precheck__icon {
  flex: none;
  width: 16px;
  color: var(--admin-warning-text);
  text-align: center;
}

.precheck__icon--ok {
  color: var(--admin-success);
}

.precheck__text {
  flex: 1;
  min-width: 0;
}

.precheck__label {
  color: var(--admin-text);
}

.precheck__detail {
  display: block;
  margin-top: 2px;
  color: var(--admin-muted);
  font-size: 12px;
  word-break: break-all;
}
</style>
