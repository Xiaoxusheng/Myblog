<script setup lang="ts">
import { LoadingOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { useMediaQuery } from '@/composables/useMediaQuery'
import { formatUpdatedAt, RANGE_OPTIONS } from '@/utils/analytics'
import type { AnalyticsRange } from '@/types/api'

/**
 * 访问分析操作区：时间范围 + 刷新 + 最后更新时间。
 * 只负责交互与状态展示；请求去重（中止在途请求）由页面统一处理。
 */
defineProps<{
  modelValue: AnalyticsRange
  /** 刷新/切换进行中：刷新按钮转圈，时间戳切换为「同步中」 */
  refreshing: boolean
  /** 最近一次成功加载的时间 */
  updatedAt: Date | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: AnalyticsRange]; refresh: [] }>()

const isMobile = useMediaQuery('(max-width: 768px)')

/** Segmented 的 change 载荷是宽泛联合类型，这里收窄回业务范围 */
function onRangeChange(value: unknown): void {
  const next = value as AnalyticsRange
  emit('update:modelValue', next)
}
</script>

<template>
  <div class="analytics-toolbar">
    <span class="analytics-toolbar__updated" role="status" aria-live="polite">
      <template v-if="refreshing">
        <LoadingOutlined spin class="analytics-toolbar__updated-icon" />
        同步中…
      </template>
      <template v-else-if="updatedAt">更新于 {{ formatUpdatedAt(updatedAt) }}</template>
      <template v-else>尚未加载</template>
    </span>

    <a-button :loading="refreshing" @click="emit('refresh')">
      <template #icon><ReloadOutlined /></template>
      刷新
    </a-button>

    <a-segmented
      class="analytics-toolbar__range"
      :value="modelValue"
      :options="RANGE_OPTIONS"
      :block="isMobile"
      @change="onRangeChange"
    />
  </div>
</template>

<style scoped>
.analytics-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
}

.analytics-toolbar__updated {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--admin-muted);
  white-space: nowrap;
}

.analytics-toolbar__updated-icon {
  font-size: 11px;
}

.analytics-toolbar__range {
  /* 时间范围是主动作，让它比右侧按钮略强一档 */
  margin-left: 2px;
}

@media (max-width: 768px) {
  .analytics-toolbar {
    width: 100%;
  }

  /* 移动端：时间范围独占一行且铺满，其余控件同行排列 */
  .analytics-toolbar__range {
    order: -1;
    width: 100%;
    margin-left: 0;
  }
}
</style>
