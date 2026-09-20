<script setup lang="ts">
import { computed, useId } from 'vue'

/**
 * 迷你趋势线（Sparkline）：纯 SVG，无 ECharts 实例与监听器开销。
 * 只表达「走向」，精确数值由同卡片的数字承担，故不画坐标轴、不响应悬浮。
 */
const props = withDefaults(
  defineProps<{
    /** 真实时序值（与主图同源，按时间升序） */
    values: number[]
    /** 依赖语义色的走向，用于取色（升/降/持平） */
    direction?: 'up' | 'down' | 'flat'
    /** 趋势线高度（px），宽度自适应容器 */
    height?: number
    /** 是否填充面积 */
    area?: boolean
    /** 无障碍描述；纯装饰时可留空 */
    ariaLabel?: string
  }>(),
  { direction: 'flat', height: 40, area: true, ariaLabel: '' },
)

/** 视口坐标系（宽度固定 100，高度按比例换算为等比纵向坐标，避免非等比拉伸） */
const VIEW_W = 100

const state = computed(() => {
  const values = props.values.filter((value) => Number.isFinite(value))
  if (values.length < 2) return { path: '', areaPath: '', single: values.length === 1 ? values[0] : null }

  const min = Math.min(...values)
  const max = Math.max(...values)
  const span = max - min || 1
  const pad = 3 // 上下留白，防止极值贴边被裁切
  const usable = VIEW_W - pad * 2

  const points = values.map((value, index) => ({
    x: pad + (index / (values.length - 1)) * usable,
    y: pad + (1 - (value - min) / span) * (100 - pad * 2),
  }))

  const path = smoothPath(points)
  const areaPath = `${path} L ${points[points.length - 1].x} 100 L ${points[0].x} 100 Z`
  return { path, areaPath, single: null }
})

/**
 * Catmull-Rom 转三次贝塞尔：让迷你趋势线保持自然柔和，
 * 比折线更贴近「趋势」语义，也不需要引入额外依赖。
 */
function smoothPath(points: { x: number; y: number }[]): string {
  if (points.length < 2) return ''
  let d = `M ${points[0].x} ${points[0].y}`
  for (let i = 0; i < points.length - 1; i += 1) {
    const p0 = points[i - 1] ?? points[i]
    const p1 = points[i]
    const p2 = points[i + 1]
    const p3 = points[i + 2] ?? p2
    const c1x = p1.x + (p2.x - p0.x) / 6
    const c1y = p1.y + (p2.y - p0.y) / 6
    const c2x = p2.x - (p3.x - p1.x) / 6
    const c2y = p2.y - (p3.y - p1.y) / 6
    d += ` C ${c1x.toFixed(2)} ${c1y.toFixed(2)}, ${c2x.toFixed(2)} ${c2y.toFixed(2)}, ${p2.x.toFixed(2)} ${p2.y.toFixed(2)}`
  }
  return d
}

/** 每个实例独立的渐变 id，避免多卡片互相引用到同一渐变 */
const gradientId = `spark-grad-${useId().replace(/[^a-zA-Z0-9-]/g, '')}`
</script>

<template>
  <svg
    class="sparkline"
    :class="`sparkline--${direction}`"
    :style="{ height: `${height}px` }"
    :viewBox="`0 0 ${VIEW_W} 100`"
    preserveAspectRatio="none"
    role="img"
    :aria-label="ariaLabel || undefined"
    :aria-hidden="ariaLabel ? undefined : 'true'"
    focusable="false"
  >
    <defs>
      <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" class="sparkline__stop-from" />
        <stop offset="100%" class="sparkline__stop-to" />
      </linearGradient>
    </defs>

    <!-- 单点：只画一条基准线 + 一个点，避免用一条平线假装趋势 -->
    <template v-if="state.single !== null">
      <line class="sparkline__baseline" x1="2" y1="50" x2="98" y2="50" />
      <line class="sparkline__dot" x1="49.6" y1="50" x2="50.4" y2="50" />
    </template>

    <template v-else-if="state.path">
      <path v-if="area" :d="state.areaPath" :fill="`url(#${gradientId})`" stroke="none" />
      <path
        class="sparkline__line"
        :d="state.path"
        fill="none"
        stroke-linecap="round"
        stroke-linejoin="round"
        vector-effect="non-scaling-stroke"
      />
    </template>
  </svg>
</template>

<style scoped>
.sparkline {
  display: block;
  width: 100%;
  overflow: visible;
}

/* 走向语义色：涨=品牌蓝、跌=警示、持平=中性——仅此三档，不引入额外彩色 */
.sparkline--up {
  color: var(--admin-brand);
}

.sparkline--down {
  color: var(--admin-warning);
}

.sparkline--flat {
  color: var(--admin-muted);
}

.sparkline__line {
  stroke: currentColor;
  stroke-width: 1.6;
  opacity: 0.9;
}

.sparkline__stop-from {
  stop-color: currentColor;
  stop-opacity: 0.16;
}

.sparkline__stop-to {
  stop-color: currentColor;
  stop-opacity: 0;
}

.sparkline__baseline {
  stroke: var(--admin-border);
  stroke-width: 1;
  stroke-dasharray: 3 3;
  vector-effect: non-scaling-stroke;
}

/* 单点圆点：用极短线段 + round 线帽绘制，
   这样在 preserveAspectRatio="none" 的非等比缩放下仍是正圆而非扁椭圆 */
.sparkline__dot {
  stroke: currentColor;
  stroke-width: 3.5;
  stroke-linecap: round;
  vector-effect: non-scaling-stroke;
}

@media (prefers-reduced-motion: reduce) {
  .sparkline {
    transition: none;
  }
}
</style>
