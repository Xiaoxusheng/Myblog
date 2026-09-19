<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import type { TrendPoint } from '@/types/api'

/**
 * 近 7 天发布/评论趋势折线图
 */
const props = defineProps<{
  data: TrendPoint[]
}>()

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
      data: ['发布文章', '新增评论'],
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
        name: '发布文章',
        type: 'line',
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.06 },
        data: props.data.map((point) => point.posts),
      },
      {
        name: '新增评论',
        type: 'line',
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2 },
        areaStyle: { opacity: 0.06 },
        data: props.data.map((point) => point.comments),
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
  <div ref="el" style="width: 100%; height: 320px"></div>
</template>
