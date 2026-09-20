<script setup lang="ts">
import { computed } from 'vue'
import { ArrowDownOutlined, ArrowUpOutlined, MinusOutlined } from '@ant-design/icons-vue'
import SectionPanel from './SectionPanel.vue'
import ChartSkeleton from './ChartSkeleton.vue'
import PvTrendChart from './PvTrendChart.vue'
import { useMediaQuery } from '@/composables/useMediaQuery'
import { formatCount, formatNumber, formatPercent } from '@/utils/format'
import { summarizeTrend, displayDate, type Delta } from '@/utils/analytics'
import type { AnalyticsTrendPoint } from '@/types/api'

/**
 * 访问趋势面板：PV/UV 双线 + 数据总览 + 峰值提示 + 环比。
 * 总览只呈现「图表才能回答」的信息（峰值、日均、有效天数），
 * 与页面顶部的 PV/UV 卡不重复计数。
 */
const props = withDefaults(
  defineProps<{
    trend: AnalyticsTrendPoint[]
    loading?: boolean
    /** 与上一周期的变化（PV 口径），缺失时不展示——不编造 */
    delta?: Delta
    /** 环比基线文案，如「较前 7 天」 */
    compareLabel?: string
  }>(),
  { loading: false },
)

const isMobile = useMediaQuery('(max-width: 768px)')
const chartHeight = computed(() => (isMobile.value ? 240 : 300))

const summary = computed(() => summarizeTrend(props.trend))

/** 只有一天数据（今日范围）时，均值/有效天数没有意义，改为陈述当日累计 */
const isSingleDay = computed(() => props.trend.length <= 1)

const overview = computed(() => {
  const { peak, averagePv } = summary.value
  const activeDays = props.trend.filter((point) => point.pv > 0).length
  return [
    {
      key: 'peak',
      label: '峰值',
      value: peak ? formatCount(peak.pv) : '0',
      exact: peak ? formatNumber(peak.pv) : undefined,
      suffix: peak ? displayDate(peak.date) : '暂无峰值',
    },
    {
      key: 'average',
      label: '日均浏览',
      value: formatCount(averagePv),
      exact: formatNumber(averagePv),
      suffix: 'PV / 天',
    },
    {
      key: 'active',
      label: '有效天数',
      value: `${activeDays}/${props.trend.length}`,
      exact: undefined,
      suffix: '有天访问',
    },
  ]
})

/** 环比徽标：方向用箭头 + 语义色双重表达，不单靠颜色 */
const ARROW = { up: ArrowUpOutlined, down: ArrowDownOutlined, flat: MinusOutlined } as const
</script>

<template>
  <SectionPanel
    title="访问趋势"
    description="按天分桶的浏览量（PV）与独立访客（UV），点击图例可单独查看某条线"
    :loading="loading"
    :empty="!loading && !summary.hasData"
    empty-title="暂无访问趋势"
    empty-hint="所选时间范围内还没有访问记录，站点产生流量后这里会按天展示走势"
  >
    <template #actions>
      <span
        v-if="delta"
        class="trend-panel__delta"
        :class="`trend-panel__delta--${delta.direction}`"
        :title="compareLabel"
      >
        <component :is="ARROW[delta.direction]" class="trend-panel__delta-icon" />
        <span class="tabular-nums">{{ delta.ratio === null ? '新增' : formatPercent(delta.ratio) }}</span>
        <span v-if="compareLabel" class="trend-panel__delta-suffix">{{ compareLabel }}</span>
      </span>
    </template>

    <template #skeleton>
      <div class="trend-panel__skeleton">
        <div class="trend-panel__overview-skeleton">
          <span v-for="item in 3" :key="item"></span>
        </div>
        <ChartSkeleton :height="chartHeight" />
      </div>
    </template>

    <!-- 总览：单日范围下均值无意义，改为陈述当日累计 -->
    <div v-if="isSingleDay" class="trend-panel__single">
      <span>
        今日累计浏览 <b class="tabular-nums">{{ formatNumber(summary.totalPv) }}</b> 次
      </span>
      <span class="trend-panel__single-sep" aria-hidden="true">·</span>
      <span>
        独立访客 <b class="tabular-nums">{{ formatNumber(summary.totalUv) }}</b> 人
      </span>
    </div>

    <dl v-else class="trend-panel__overview">
      <div v-for="item in overview" :key="item.key" class="trend-panel__overview-item">
        <dt class="trend-panel__overview-label">{{ item.label }}</dt>
        <dd class="trend-panel__overview-value">
          <span class="tabular-nums" :title="item.exact">{{ item.value }}</span>
          <small class="trend-panel__overview-suffix">{{ item.suffix }}</small>
        </dd>
      </div>
    </dl>

    <PvTrendChart :data="trend" variant="rich" value-suffix="次" :height="chartHeight" />
  </SectionPanel>
</template>

<style scoped>
/* ---------- 环比徽标 ---------- */
.trend-panel__delta {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  border-radius: 999px;
  white-space: nowrap;
}

.trend-panel__delta-icon {
  font-size: 10px;
}

.trend-panel__delta-suffix {
  font-weight: 400;
}

.trend-panel__delta--up {
  color: var(--admin-brand);
  background: var(--admin-brand-bg);
}

.trend-panel__delta--down {
  color: var(--admin-warning-text);
  background: var(--admin-warning-bg);
}

.trend-panel__delta--flat {
  color: var(--admin-muted);
  background: var(--admin-surface-2);
}

/* ---------- 总览条：轻量分隔，不做成第二组卡片 ---------- */
.trend-panel__overview {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 28px;
  margin: 0 0 4px;
  padding: 0 0 12px;
  border-bottom: 1px solid var(--admin-border);
}

.trend-panel__overview-item {
  min-width: 0;
}

.trend-panel__overview-label {
  margin: 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

.trend-panel__overview-value {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  line-height: 22px;
  letter-spacing: -0.2px;
  color: var(--admin-text);
}

.trend-panel__overview-suffix {
  font-size: 12px;
  font-weight: 400;
  color: var(--admin-muted);
}

.trend-panel__single {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 0 12px;
  margin-bottom: 4px;
  font-size: 13px;
  color: var(--admin-muted);
  border-bottom: 1px solid var(--admin-border);
}

.trend-panel__single b {
  font-weight: 600;
  color: var(--admin-text);
}

.trend-panel__single-sep {
  color: var(--admin-border);
}

/* ---------- 骨架 ---------- */
.trend-panel__skeleton {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.trend-panel__overview-skeleton {
  display: flex;
  gap: 28px;
}

.trend-panel__overview-skeleton span {
  display: block;
  width: 72px;
  height: 34px;
  border-radius: var(--admin-radius-sm);
  background: var(--admin-surface-2);
}

@media (max-width: 768px) {
  .trend-panel__overview {
    gap: 8px 20px;
  }

  .trend-panel__overview-value {
    font-size: 15px;
  }
}
</style>
