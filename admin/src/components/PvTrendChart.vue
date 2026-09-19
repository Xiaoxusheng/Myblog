<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { AnalyticsTrendPoint } from '@/types/api'

/**
 * PV/UV 双系列趋势折线图（访问分析页与单篇文章分析抽屉共用）
 * 配色沿用 TrendChart 的降饱和方案：品牌蓝 + 灰
 */
const props = withDefaults(
  defineProps<{
    data: AnalyticsTrendPoint[]
    /** 图表高度（px），抽屉内可用小值 */
    height?: number
  }>(),
  { height: 320 },
)

const el = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

function render() {
  if (!chart) return
  chart.setOption({
    color: ['#4096ff', '#a6adb8'],
    tooltip: {
      trigger: 'axis',
    },
    legend: {
      data: ['浏览 PV', '访客 UV'],
      top: 0,
      right: 8,
      icon: 'roundRect',
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { color: 'rgba(0,0,0,0.65)', fontSize: 12 },
    },
    grid: {
      left: 16,
      right: 24,
      top: 40,
      bottom: 8,
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      axisLine: { lineStyle: { color: '#e6e8eb' } },
      axisTick: { show: false },
      axisLabel: { color: 'rgba(0,0,0,0.45)' },
      data: props.data.map((point) => point.date.slice(5)),
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: { color: 'rgba(0,0,0,0.45)' },
      splitLine: { lineStyle: { color: '#f0f1f3' } },
    },
    series: [
      {
        name: '浏览 PV',
        type: 'line',
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.06 },
        data: props.data.map((point) => point.pv),
      },
      {
        name: '访客 UV',
        type: 'line',
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.06 },
        data: props.data.map((point) => point.uv),
      },
    ],
  })
}

function onResize() {
  chart?.resize()
}

watch(
  () => props.data,
  () => render(),
  { deep: true },
)

onMounted(() => {
  if (el.value) {
    chart = echarts.init(el.value)
    render()
    window.addEventListener('resize', onResize)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  chart?.dispose()
  chart = null
})
</script>

<template>
  <div ref="el" class="pv-trend" :style="{ height: `${height}px` }"></div>
</template>

<style scoped>
.pv-trend {
  width: 100%;
}
</style>
