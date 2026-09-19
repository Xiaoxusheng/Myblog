<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { chartPalette } from '@/theme/tokens'

/**
 * 双系列趋势折线图基础组件(docs/09 §4.2)
 * 发布/评论趋势与 PV/UV 趋势共用;色板/轴线/文字全部来自 theme/tokens,与 AntD 主题同源
 */
const props = withDefaults(
  defineProps<{
    labels: string[]
    series: Array<{ name: string; data: number[] }>
    /** 图表高度(px),抽屉内可用小值 */
    height?: number
  }>(),
  { height: 320 },
)

const el = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

function render() {
  if (!chart) return
  chart.setOption({
    color: [chartPalette.primary, chartPalette.secondary],
    tooltip: {
      trigger: 'axis',
    },
    legend: {
      data: props.series.map((s) => s.name),
      top: 0,
      right: 8,
      icon: 'roundRect',
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { color: chartPalette.legendText, fontSize: 12 },
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
      axisLine: { lineStyle: { color: chartPalette.axisLine } },
      axisTick: { show: false },
      axisLabel: { color: chartPalette.axisLabel },
      data: props.labels,
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: { color: chartPalette.axisLabel },
      splitLine: { lineStyle: { color: chartPalette.splitLine } },
    },
    series: props.series.map((s) => ({
      name: s.name,
      type: 'line',
      smooth: true,
      showSymbol: false,
      lineStyle: { width: 2 },
      areaStyle: { opacity: 0.06 },
      data: s.data,
    })),
  })
}

function onResize() {
  chart?.resize()
}

watch(
  () => [props.labels, props.series],
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
  <div ref="el" class="base-trend" :style="{ height: `${height}px` }"></div>
</template>

<style scoped>
.base-trend {
  width: 100%;
}
</style>
