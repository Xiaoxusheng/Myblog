<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { adminTheme } from '@/theme'
import { chartDarkPalette, chartPalette } from '@/theme/tokens'
import { formatNumber } from '@/utils/format'

/**
 * 双系列趋势折线图基础组件(docs/09 §4.2)
 * 发布/评论趋势与 PV/UV 趋势共用;色板/轴线/文字全部来自 theme/tokens,与 AntD 主题同源。
 *
 * `variant`:
 * - `plain`(默认)——历史基础样式,既有页面(仪表盘/文章分析抽屉)渲染结果不变
 * - `rich`——访问分析增强样式:面积渐变、弱化网格、图例选中态、峰值标记、格式化悬浮详情
 * 新增能力全部为可选参数,不传即保持原有输出。
 */
const props = withDefaults(
  defineProps<{
    labels: string[]
    series: Array<{ name: string; data: number[] }>
    /** 图表高度(px),抽屉内可用小值 */
    height?: number
    /** 呈现风格;默认 plain 以保证既有调用方零变化 */
    variant?: 'plain' | 'rich'
    /** 数值单位后缀,用于悬浮详情与峰值标签(如「次」) */
    valueSuffix?: string
    /**
     * 与 labels 一一对应的完整标题(如 YYYY-MM-DD),
     * 仅用于悬浮详情与峰值提示;缺省时回退到 labels
     */
    axisTitles?: string[]
  }>(),
  { height: 320, variant: 'plain', valueSuffix: '' },
)

const el = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null

function palette() {
  return adminTheme.value === 'dark' ? chartDarkPalette : chartPalette
}

/** 首条序列(访问分析中即 PV)的峰值点,用于"最高访问日期"提示 */
function findPeak(): { index: number; value: number } | null {
  const data = props.series[0]?.data ?? []
  if (data.length < 2) return null
  let index = 0
  for (let i = 1; i < data.length; i += 1) {
    if (data[i] > data[index]) index = i
  }
  return data[index] > 0 ? { index, value: data[index] } : null
}

/** 峰值标签朝向:贴边时向内偏，避免被绘图区裁切 */
function peakLabelPosition(index: number, total: number): 'top' | 'left' | 'right' {
  const ratio = total > 1 ? index / (total - 1) : 0.5
  if (ratio <= 0.18) return 'right'
  if (ratio >= 0.82) return 'left'
  return 'top'
}

function render() {
  if (!chart) return
  const p = palette()
  const rich = props.variant === 'rich'
  const titles = props.axisTitles ?? props.labels
  const single = props.labels.length === 1
  const peak = rich ? findPeak() : null

  const series = props.series.map((item, seriesIndex) => ({
    name: item.name,
    type: 'line' as const,
    smooth: true,
    // 点少时显示节点，便于读数；点多时隐藏以免线变毛躁
    showSymbol: rich && (props.labels.length <= 16 || single),
    symbolSize: single ? 9 : 5,
    lineStyle: { width: rich ? 2 : 2 },
    ...(rich
      ? {
          itemStyle: { borderWidth: 2, borderColor: p.markerRing },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              {
                offset: 0,
                color: seriesIndex === 0 ? p.areaPrimary[0] : p.areaSecondary[0],
              },
              {
                offset: 1,
                color: seriesIndex === 0 ? p.areaPrimary[1] : p.areaSecondary[1],
              },
            ]),
          },
        }
      : { areaStyle: { opacity: 0.06 } }),
    data: item.data,
    // 仅在首条序列上标记峰值
    ...(peak && seriesIndex === 0
      ? {
          markPoint: {
            silent: true,
            symbol: 'circle',
            symbolSize: 7,
            itemStyle: { color: p.primary, borderColor: p.markerRing, borderWidth: 2 },
            label: {
              show: true,
              position: peakLabelPosition(peak.index, props.labels.length),
              distance: 9,
              formatter: `峰值 ${formatNumber(peak.value)}${props.valueSuffix}`,
              color: p.legendText,
              fontSize: 11,
              fontWeight: 500 as const,
            },
            data: [{ coord: [peak.index, peak.value] }],
          },
        }
      : {}),
  }))

  chart.setOption(
    {
      color: [p.primary, p.secondary],
      tooltip: {
        trigger: 'axis',
        axisPointer: rich
          ? { type: 'line', lineStyle: { color: p.axisPointer, width: 1 } }
          : undefined,
        ...(rich
          ? {
              backgroundColor: p.tooltipBg,
              borderColor: p.tooltipBorder,
              borderWidth: 1,
              padding: [8, 12],
              extraCssText:
                'border-radius:8px;box-shadow:0 2px 4px rgba(16,24,40,0.04),0 12px 28px rgba(16,24,40,0.1);',
            }
          : {}),
        textStyle: { color: rich ? p.tooltipText : undefined, fontSize: 12 },
        formatter: rich
          ? (params: unknown) => {
              const list = (Array.isArray(params) ? params : [params]) as {
                dataIndex: number
                seriesName: string
                value: number
                color: string
              }[]
              if (list.length === 0) return ''
              const title = titles[list[0].dataIndex] ?? props.labels[list[0].dataIndex] ?? ''
              const rows = list
                .map(
                  (item) =>
                    `<div style="display:flex;align-items:center;gap:8px;line-height:20px">
                       <span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${item.color}"></span>
                       <span style="flex:1;margin-right:16px">${item.seriesName}</span>
                       <strong style="font-variant-numeric:tabular-nums">${formatNumber(item.value)}${props.valueSuffix}</strong>
                     </div>`,
                )
                .join('')
              return `<div style="font-weight:600;margin-bottom:4px">${title}</div>${rows}`
            }
          : undefined,
      },
      legend: {
        data: props.series.map((item) => item.name),
        top: 0,
        right: 8,
        icon: 'roundRect',
        itemWidth: 8,
        itemHeight: 8,
        itemGap: 16,
        // 选中/未选中拉开差距，让图例的筛选状态一眼可辨
        inactiveColor: p.legendMuted,
        textStyle: { color: p.legendText, fontSize: 12 },
      },
      grid: rich
        ? { left: 4, right: 12, top: rich && peak ? 46 : 40, bottom: 0, containLabel: true }
        : { left: 16, right: 24, top: 40, bottom: 8, containLabel: true },
      xAxis: {
        type: 'category',
        // 只有一个数据点时居中放置，避免单点贴在左边缘、右侧大片留白
        boundaryGap: rich ? single : false,
        axisLine: { lineStyle: { color: p.axisLine } },
        axisTick: { show: false },
        axisLabel: {
          color: p.axisLabel,
          hideOverlap: true,
        },
        data: props.labels,
      },
      yAxis: {
        type: 'value',
        minInterval: 1,
        // 空数据时固定 0~1，避免出现"0 到 0"的退化刻度
        ...(single
          ? { min: 0, max: Math.max((props.series[0]?.data[0] ?? 0) * 1.35, 1) }
          : {}),
        ...(rich ? { splitNumber: 4 } : {}),
        axisLine: { show: !rich },
        axisTick: { show: false },
        axisLabel: { color: p.axisLabel },
        splitLine: {
          lineStyle: { color: p.splitLine, ...(rich ? { type: 'dashed' as const } : {}) },
        },
      },
      series,
    },
    // 不合并：范围切换/主题切换时整块重设，避免上一态的 markPoint 残留
    { notMerge: true },
  )
}

onMounted(() => {
  if (!el.value) return
  chart = echarts.init(el.value)
  render()
  // 容器尺寸监听：覆盖窗口缩放、侧栏折叠、响应式断点变化（卸载时必须断开）
  if (typeof ResizeObserver !== 'undefined') {
    observer = new ResizeObserver(() => chart?.resize())
    observer.observe(el.value)
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
  chart?.dispose()
  chart = null
})

watch(
  () => [props.labels, props.series, props.variant, props.valueSuffix, props.axisTitles],
  () => render(),
  { deep: true },
)

// 主题切换时重设图表色板
watch(adminTheme, () => render())
</script>

<template>
  <div
    ref="el"
    class="base-trend"
    :style="{ height: `${height}px` }"
    role="img"
    :aria-label="`${series.map((item) => item.name).join('与')}趋势图`"
  ></div>
</template>

<style scoped>
.base-trend {
  width: 100%;
}
</style>
