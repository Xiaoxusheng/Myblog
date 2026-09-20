<script setup lang="ts">
import { computed } from 'vue'
import { formatCount, formatNumber } from '@/utils/format'
import type { AnalyticsDistItem } from '@/types/api'

/**
 * 分布横条列表：名称 + 占比横条 + PV 数（纯 CSS，无 ECharts）
 * 来源/设备/浏览器/操作系统分布共用
 */
const props = defineProps<{
  items: AnalyticsDistItem[]
}>()

const maxPv = computed(() => props.items.reduce((max, item) => Math.max(max, item.pv), 0))
const totalPv = computed(() => props.items.reduce((sum, item) => sum + item.pv, 0))

function barWidth(pv: number): string {
  if (maxPv.value <= 0 || pv <= 0) return '0%'
  return `${Math.max((pv / maxPv.value) * 100, 2)}%`
}

/** 悬浮给出精确值与占比——横条只表达相对量级，精确读数在这里补齐 */
function rowTitle(name: string, pv: number): string {
  if (totalPv.value <= 0) return `${name}：0`
  const share = ((pv / totalPv.value) * 100).toFixed(1)
  return `${name}：${formatNumber(pv)} 次（占 ${share}%）`
}
</script>

<template>
  <div class="dist">
    <div v-for="item in items" :key="item.name" class="dist__row" :title="rowTitle(item.name, item.pv)">
      <span class="dist__name">{{ item.name }}</span>
      <span class="dist__track">
        <span class="dist__fill" :style="{ width: barWidth(item.pv) }"></span>
      </span>
      <span class="dist__pv tabular-nums">{{ formatCount(item.pv) }}</span>
    </div>
  </div>
</template>

<style scoped>
.dist {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dist__row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.dist__name {
  flex: none;
  width: 76px;
  font-size: 13px;
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dist__track {
  flex: 1;
  min-width: 0;
  height: 8px;
  border-radius: 4px;
  background: var(--admin-surface-2);
  overflow: hidden;
}

/* 品牌蓝低饱和底：与图表主色一致 */
.dist__fill {
  display: block;
  height: 100%;
  min-width: 2px;
  border-radius: 4px;
  background: rgba(64, 150, 255, 0.55);
  transition: width 0.3s ease;
}

.dist__pv {
  flex: none;
  width: 48px;
  text-align: right;
  font-size: 13px;
  color: var(--admin-muted);
}

@media (max-width: 768px) {
  .dist__name {
    width: 64px;
  }

  .dist__pv {
    width: 40px;
  }
}
</style>
