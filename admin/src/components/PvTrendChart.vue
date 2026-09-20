<script setup lang="ts">
import { computed } from 'vue'
import BaseTrendChart from './BaseTrendChart.vue'
import { axisDate, displayDate } from '@/utils/analytics'
import type { AnalyticsTrendPoint } from '@/types/api'

/**
 * PV/UV 双系列趋势折线图（访问分析页与单篇文章分析抽屉共用；
 * 基础组件薄壳，对外 props 契约向后兼容——新增参数均有默认值）
 */
const props = withDefaults(
  defineProps<{
    data: AnalyticsTrendPoint[]
    /** 图表高度(px)，抽屉内可用小值 */
    height?: number
    /** 呈现风格：默认 plain，访问分析页传 rich 启用面积/峰值/格式化悬浮 */
    variant?: 'plain' | 'rich'
    /** 数值单位后缀，如「次」 */
    valueSuffix?: string
  }>(),
  { height: 320, variant: 'plain', valueSuffix: '' },
)

const series = computed(() => [
  { name: '浏览 PV', data: props.data.map((point) => point.pv) },
  { name: '访客 UV', data: props.data.map((point) => point.uv) },
])
/** 坐标轴刻度：短日期，保证 90 天时不拥挤 */
const labels = computed(() => props.data.map((point) => axisDate(point.date)))
/** 悬浮详情用完整日期，读数时不必再猜年份月份 */
const titles = computed(() => props.data.map((point) => displayDate(point.date)))
</script>

<template>
  <BaseTrendChart
    :series="series"
    :labels="labels"
    :axis-titles="titles"
    :height="height"
    :variant="variant"
    :value-suffix="valueSuffix"
  />
</template>
