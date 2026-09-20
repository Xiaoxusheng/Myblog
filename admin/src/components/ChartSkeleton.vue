<script setup lang="ts">
/**
 * 图表骨架屏：用低对比度的网格 + 面积轮廓占据图表同等体积，
 * 数据到位时布局不跳动。纯装饰，对读屏隐藏。
 */
withDefaults(defineProps<{ height?: number }>(), { height: 320 })
</script>

<template>
  <div class="chart-skeleton" :style="{ height: `${height}px` }" aria-hidden="true">
    <span
      v-for="line in 4"
      :key="line"
      class="chart-skeleton__gridline"
      :style="{ top: `${(line - 1) * 25}%` }"
    ></span>
    <svg
      class="chart-skeleton__area"
      viewBox="0 0 100 40"
      preserveAspectRatio="none"
      focusable="false"
    >
      <path
        d="M0 31 C 9 27, 15 34, 24 29 S 38 15, 48 21 S 63 11, 74 17 S 89 9, 100 13 L 100 40 L 0 40 Z"
      />
    </svg>
  </div>
</template>

<style scoped>
.chart-skeleton {
  position: relative;
  width: 100%;
  overflow: hidden;
  border-radius: var(--admin-radius-sm);
}

.chart-skeleton__gridline {
  position: absolute;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--admin-border);
  opacity: 0.7;
}

.chart-skeleton__area {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  fill: var(--admin-surface-2);
}

/* 轻微扫光：表明"在加载"而非"加载完成但为空" */
.chart-skeleton::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    transparent 0%,
    color-mix(in srgb, var(--admin-surface) 55%, transparent) 50%,
    transparent 100%
  );
  transform: translateX(-100%);
  animation: chart-skeleton-sweep 1.6s ease-in-out infinite;
}

@keyframes chart-skeleton-sweep {
  to {
    transform: translateX(100%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .chart-skeleton::after {
    animation: none;
    opacity: 0;
  }
}
</style>
