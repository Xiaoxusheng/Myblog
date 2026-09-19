<script setup lang="ts">
import { computed } from 'vue'
import BaseTrendChart from './BaseTrendChart.vue'
import type { AnalyticsTrendPoint } from '@/types/api'

/**
 * PV/UV 双系列趋势折线图(访问分析页与单篇文章分析抽屉共用;基础组件薄壳,对外 props 契约不变)
 */
const props = withDefaults(
  defineProps<{
    data: AnalyticsTrendPoint[]
    /** 图表高度(px),抽屉内可用小值 */
    height?: number
  }>(),
  { height: 320 },
)

const series = computed(() => [
  { name: '浏览 PV', data: props.data.map((point) => point.pv) },
  { name: '访客 UV', data: props.data.map((point) => point.uv) }
])
const labels = computed(() => props.data.map((point) => point.date.slice(5)))
</script>

<template>
  <BaseTrendChart :series="series" :labels="labels" :height="height" />
</template>
