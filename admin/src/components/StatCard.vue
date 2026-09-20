<script setup lang="ts">
import type { Component } from 'vue'
import { ArrowDownOutlined, ArrowUpOutlined, MinusOutlined } from '@ant-design/icons-vue'
import SparkLine from './SparkLine.vue'
import { formatPercent } from '@/utils/format'
import type { Delta } from '@/utils/analytics'

/**
 * 统计卡片：图标块 + 标题 + 大数字 + 辅助信息（数据一律来自真实接口字段）。
 *
 * 扩展项（delta / spark / loading / empty）全部可选且默认关闭：
 * 不传时渲染结果与历史版本完全一致，仪表盘/草稿工作区/文章分析抽屉零影响。
 */
withDefaults(
  defineProps<{
    title: string
    value: number | string
    icon: Component
    /** 辅助信息（真实字段派生，如"待审核 2 条"） */
    hint?: string
    /** 告警态（如待审 > 0）：数字与图标用 warning 色 */
    warning?: boolean
    /** 数值悬浮提示（紧凑数字场景给出精确值） */
    valueTitle?: string
    /** 数值字号：默认档保持与既有页面一致，large 用于宽版数据概览卡 */
    size?: 'default' | 'large'
    /** 加载态：渲染骨架，避免数字从 0 跳到真实值 */
    loading?: boolean
    /** 空数据态：数值显示占位符并说明，不显示 0 冒充真实读数 */
    empty?: boolean
    /** 环比结果；未提供时不渲染对比区（缺失即隐藏，不编造） */
    delta?: Delta
    /** 环比基线文案，如「较前 7 天」——让"和什么比"始终明确 */
    compareLabel?: string
    /** 迷你趋势线时序值（与主图同源的真实数据） */
    spark?: number[]
    /** 趋势线无障碍描述 */
    sparkLabel?: string
  }>(),
  { size: 'default' },
)

/** 环比文案：基期为 0 时百分比无意义，如实显示"新增" */
function deltaText(delta: Delta): string {
  return delta.ratio === null ? '新增' : formatPercent(delta.ratio)
}

/** 变化方向 → 语义类（升=品牌蓝重点、降=警示、持平=中性；方向另有箭头标识，不单靠颜色） */
function deltaClass(delta: Delta): string {
  return `stat-card__delta--${delta.direction}`
}

const ARROW = { up: ArrowUpOutlined, down: ArrowDownOutlined, flat: MinusOutlined } as const
</script>

<template>
  <div
    class="stat-card"
    :class="[
      `stat-card--${size}`,
      {
        'stat-card--warning': warning,
        'stat-card--loading': loading,
        'stat-card--empty': empty,
      },
    ]"
  >
    <div class="stat-card__head">
      <div class="stat-card__icon" :class="{ 'stat-card__icon--warning': warning }">
        <component :is="icon" />
      </div>

      <div class="stat-card__meta">
        <div class="stat-card__title">{{ title }}</div>

        <div v-if="loading" class="stat-card__skeleton" aria-hidden="true">
          <span class="stat-card__skeleton-value"></span>
          <span class="stat-card__skeleton-hint"></span>
        </div>

        <template v-else>
          <div class="stat-card__figure">
            <span
              class="stat-card__value tabular-nums"
              :class="{ 'stat-card__value--warning': warning }"
              :title="valueTitle"
            >
              {{ empty ? '—' : value }}
            </span>

            <span
              v-if="delta && !empty"
              class="stat-card__delta"
              :class="deltaClass(delta)"
              :title="compareLabel ? `${compareLabel}：${deltaText(delta)}` : deltaText(delta)"
            >
              <component :is="ARROW[delta.direction]" class="stat-card__delta-icon" />
              <span class="tabular-nums">{{ deltaText(delta) }}</span>
              <span v-if="compareLabel" class="stat-card__delta-suffix">{{ compareLabel }}</span>
            </span>
          </div>

          <div v-if="hint" class="stat-card__hint">{{ hint }}</div>
        </template>
      </div>
    </div>

    <!-- 迷你趋势：有真实时序数据才渲染（单点由 SparkLine 降级为基准线 + 点） -->
    <div v-if="!loading && !empty && spark && spark.length > 0" class="stat-card__spark">
      <SparkLine
        :values="spark"
        :direction="delta?.direction ?? 'flat'"
        :height="size === 'large' ? 44 : 34"
        :aria-label="sparkLabel"
      />
    </div>
  </div>
</template>

<style scoped>
.stat-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  padding: 16px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-sm);
  /* 悬停轻微浮起：纯 transform/阴影过渡，不引起重排 */
  transition:
    transform var(--admin-dur) var(--admin-ease),
    box-shadow var(--admin-dur) var(--admin-ease),
    border-color var(--admin-dur) var(--admin-ease);
}

.stat-card--large {
  padding: 18px 20px;
}

.stat-card:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--admin-brand) 28%, var(--admin-border));
  box-shadow: var(--admin-shadow-md);
}

/* 空态卡无数据可交互，不做浮起反馈 */
.stat-card--empty:hover {
  transform: none;
  border-color: var(--admin-border);
  box-shadow: var(--admin-shadow-sm);
}

@media (prefers-reduced-motion: reduce) {
  .stat-card,
  .stat-card:hover {
    transition: none;
    transform: none;
  }
}

.stat-card__head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.stat-card__icon {
  flex: none;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: var(--admin-brand);
  background: var(--admin-brand-bg);
  border-radius: var(--admin-radius-md);
}

.stat-card__icon--warning {
  color: var(--admin-warning);
  background: var(--admin-warning-bg);
}

.stat-card__meta {
  flex: 1;
  min-width: 0;
}

.stat-card__title {
  font-size: 13px;
  line-height: 20px;
  color: var(--admin-muted);
}

.stat-card__figure {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 4px 10px;
  margin-top: 2px;
}

.stat-card__value {
  font-size: 26px;
  font-weight: 700;
  line-height: 32px;
  letter-spacing: -0.3px;
  color: var(--admin-text);
}

.stat-card--large .stat-card__value {
  font-size: 30px;
  line-height: 38px;
}

.stat-card__value--warning {
  color: var(--admin-warning-text);
}

.stat-card--empty .stat-card__value {
  color: var(--admin-muted);
  font-weight: 600;
}

/* ---------- 环比 ---------- */
.stat-card__delta {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  white-space: nowrap;
}

.stat-card__delta-icon {
  font-size: 10px;
}

.stat-card__delta-suffix {
  margin-left: 2px;
  font-weight: 400;
  color: var(--admin-muted);
}

.stat-card__delta--up {
  color: var(--admin-brand);
}

.stat-card__delta--down {
  color: var(--admin-warning-text);
}

.stat-card__delta--flat {
  color: var(--admin-muted);
}

/* ---------- 辅助信息 ---------- */
.stat-card__hint {
  margin-top: 2px;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

/* ---------- 迷你趋势 ---------- */
.stat-card__spark {
  margin-top: auto;
  /* 负外边距让趋势线通栏铺满，视觉上成为卡片的"底纹"而非独立元素 */
  margin-inline: -4px;
  margin-bottom: -2px;
}

/* ---------- 加载骨架 ---------- */
.stat-card__skeleton {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 6px;
}

.stat-card__skeleton-value,
.stat-card__skeleton-hint {
  display: block;
  border-radius: var(--admin-radius-sm);
  background: linear-gradient(
    90deg,
    var(--admin-surface-2) 25%,
    color-mix(in srgb, var(--admin-border) 55%, var(--admin-surface-2)) 37%,
    var(--admin-surface-2) 63%
  );
  background-size: 400% 100%;
  animation: stat-card-shimmer 1.4s ease infinite;
}

.stat-card__skeleton-value {
  width: 58%;
  height: 26px;
}

.stat-card__skeleton-hint {
  width: 38%;
  height: 12px;
}

@keyframes stat-card-shimmer {
  from {
    background-position: 100% 50%;
  }
  to {
    background-position: 0 50%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .stat-card__skeleton-value,
  .stat-card__skeleton-hint {
    animation: none;
  }
}
</style>
