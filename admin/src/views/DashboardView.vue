<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import {
  ArrowRightOutlined,
  AuditOutlined,
  ClockCircleOutlined,
  CommentOutlined,
  EyeOutlined,
  FileTextOutlined,
  FormOutlined,
  LikeOutlined,
  PictureOutlined,
  ReloadOutlined,
  SettingOutlined,
} from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import StatCard from '@/components/StatCard.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import BaseTrendChart from '@/components/BaseTrendChart.vue'
import { getStats } from '@/api/stats'
import { getAnalytics } from '@/api/analytics'
import { COMMENT_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import { computeDelta } from '@/utils/analytics'
import type { AnalyticsRange, AnalyticsTrendPoint, Stats } from '@/types/api'

/**
 * 仪表盘（v2 设计稿 01 页）。
 *
 * 结构：Hero 标题 + 范围筛选 → 4 张「大数字 + 环比 + 迷你趋势」概览卡 →
 * 访问趋势（主视觉）+ 来源分布 → 最近评论 + 计划发布 → 快捷入口。
 *
 * 数据口径：全部字段来自 /admin/stats（契约 #15）与 /admin/analytics（契约 #67）。
 * 环比基线取 analytics.prevTotals（紧邻前一窗口、等长、按已过时长对齐），
 * 缺失时 computeDelta 返回 undefined，UI 隐藏对比区而不是编数字。
 */

const router = useRouter()

const loading = ref(true)
const error = ref(false)
const stats = ref<Stats | null>(null)

/** 概览卡的时间范围（与设计稿「近 7 天 / 近 30 天 / 近 90 天」筛选一致） */
const range = ref<AnalyticsRange>('7d')
const RANGE_OPTIONS: { label: string; value: AnalyticsRange }[] = [
  { label: '近 7 天', value: '7d' },
  { label: '近 30 天', value: '30d' },
  { label: '近 90 天', value: '90d' },
]

async function load() {
  loading.value = true
  error.value = false
  try {
    stats.value = await getStats()
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
  void loadAnalytics()
}

/** 访问数据（趋势 + 环比 + 来源）：失败只降级对应面板，不影响主视图 */
const trend = ref<AnalyticsTrendPoint[]>([])
const totals = ref<{ pv: number; uv: number } | null>(null)
const prevTotals = ref<{ pv: number; uv: number } | null>(null)
const sources = ref<{ source: string; pv: number }[]>([])
const analyticsLoading = ref(false)
/** 请求竞态保护：快速切范围时丢弃过期响应 */
let analyticsSeq = 0

async function loadAnalytics() {
  const seq = ++analyticsSeq
  analyticsLoading.value = true
  try {
    const result = await getAnalytics(range.value)
    if (seq !== analyticsSeq) return
    trend.value = result.trend
    totals.value = result.totals
    prevTotals.value = result.prevTotals ?? null
    sources.value = result.sources ?? []
  } catch {
    if (seq !== analyticsSeq) return
    trend.value = []
    totals.value = null
    prevTotals.value = null
    sources.value = []
  } finally {
    if (seq === analyticsSeq) analyticsLoading.value = false
  }
}

function onRangeChange(value: string | number) {
  range.value = value as AnalyticsRange
  void loadAnalytics()
}

onMounted(load)

/** 迷你趋势线数据：从日分桶序列抽出 PV（与主图同源，不另造数据） */
const pvSeries = computed(() => trend.value.map((point) => point.pv))
const uvSeries = computed(() => trend.value.map((point) => point.uv))

/** 环比基线文案：与筛选范围绑定，让「和什么比」始终明确 */
const compareLabel = computed(() => {
  const map: Record<AnalyticsRange, string> = {
    today: '较昨日同期',
    '7d': '较前 7 天',
    '30d': '较前 30 天',
    '90d': '较前 90 天',
  }
  return map[range.value]
})

interface OverviewCard {
  key: string
  title: string
  value: number | string
  icon: typeof FileTextOutlined
  hint?: string
  warning?: boolean
  spark?: number[]
  sparkLabel?: string
  delta?: ReturnType<typeof computeDelta>
}

/**
 * 四张概览卡（v2 设计稿：内容总量 / 总浏览量 / 待审核 / 总点赞）。
 * 优先级按「需要你动手的程度」排：待审核排第三但用 warning 色抢占视线。
 */
const overviewCards = computed<OverviewCard[]>(() => {
  const s = stats.value
  if (!s) return []

  const pvDelta = computeDelta(totals.value?.pv ?? 0, prevTotals.value?.pv ?? undefined)
  const uvDelta = computeDelta(totals.value?.uv ?? 0, prevTotals.value?.uv ?? undefined)

  return [
    {
      key: 'postCount',
      title: '文章总数',
      value: s.postCount,
      icon: FileTextOutlined,
      hint: `已发布 ${s.postCount - s.draftCount - s.scheduledCount} · 草稿 ${s.draftCount} · 计划 ${s.scheduledCount}`,
    },
    {
      key: 'viewCount',
      title: '总浏览量',
      value: s.viewCount,
      icon: EyeOutlined,
      // 概览卡的趋势线只画 PV，避免 PV/UV 双线在 44px 高度里区分不清
      spark: pvSeries.value,
      sparkLabel: '近段时间 PV 走势',
      delta: pvDelta,
      hint:
        uvDelta || totals.value
          ? `独立访客 ${totals.value?.uv ?? 0}`
          : `今日 ${s.todayPv} · 昨日 ${s.yesterdayPv}`,
    },
    {
      key: 'pendingCommentCount',
      title: '待审核评论',
      value: s.pendingCommentCount,
      icon: AuditOutlined,
      hint: s.pendingCommentCount > 0 ? '需要你处理' : `评论共 ${s.commentCount} 条`,
      warning: s.pendingCommentCount > 0,
    },
    {
      key: 'likeCount',
      title: '总点赞数',
      value: s.likeCount,
      icon: LikeOutlined,
      spark: uvSeries.value,
      sparkLabel: '近段时间 UV 走势',
      hint: `历史累计`,
    },
  ]
})

const quickActions = [
  { key: 'post', title: '新建文章', desc: '撰写并发布内容', icon: FormOutlined, to: '/posts/edit' },
  { key: 'comment', title: '审核评论', desc: '处理待审核评论', icon: CommentOutlined, to: '/comments?status=0' },
  { key: 'media', title: '媒体库', desc: '管理图片素材', icon: PictureOutlined, to: '/media' },
  { key: 'settings', title: '系统设置', desc: '站点参数配置', icon: SettingOutlined, to: '/settings' },
] as const

function goAction(to: string) {
  void router.push(to)
}

function goComments() {
  void router.push('/comments')
}

/** 无头像时取昵称首字 */
function initial(nickname: string): string {
  return nickname.trim().charAt(0).toUpperCase()
}

const scheduledPosts = computed(() => stats.value?.scheduledPosts ?? [])

function formatPlanTime(value: string): string {
  const target = dayjs(value)
  return target.isValid() ? target.format('MM-DD HH:mm') : '-'
}

/** 相对描述：约 N 分钟/小时/天后；已到点但尚未上线的显示「即将发布」 */
function relativePlanText(value: string): string {
  const target = dayjs(value)
  if (!target.isValid()) return '时间待定'
  const minutes = target.diff(dayjs(), 'minute')
  if (minutes <= 0) return '即将发布'
  if (minutes < 60) return `约 ${minutes} 分钟后`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `约 ${hours} 小时后`
  return `约 ${Math.floor(hours / 24)} 天后`
}

function goEditPost(id: number) {
  void router.push(`/posts/edit/${id}`)
}

/** 来源分布：把 source 码转成中文标签（后端只返回英文枚举） */
const SOURCE_LABEL: Record<string, string> = {
  direct: '直接访问',
  search: '搜索引擎',
  github: 'GitHub',
  social: '社交分享',
  other: '其他',
}

/** 来源分布的占比（分母为各项之和，不除以 totals.pv——二者口径可能不同） */
const sourceList = computed(() => {
  const total = sources.value.reduce((sum, item) => sum + item.pv, 0)
  if (total === 0) return []
  return sources.value
    .map((item) => ({
      label: SOURCE_LABEL[item.source] ?? item.source,
      pv: item.pv,
      percent: Math.round((item.pv / total) * 100),
    }))
    .sort((a, b) => b.pv - a.pv)
})

/** 主趋势图数据：转成 BaseTrendChart 期望的 labels/series 形状 */
const trendLabels = computed(() => trend.value.map((point) => point.date.slice(5)))
const trendSeries = computed(() => [
  { name: 'PV', data: pvSeries.value },
  { name: 'UV', data: uvSeries.value },
])
</script>

<template>
  <div class="page">
    <PageHeader title="仪表盘" description="站点内容与互动数据总览">
      <template #actions>
        <a-segmented
          :value="range"
          :options="RANGE_OPTIONS"
          size="small"
          @change="onRangeChange"
        />
        <a-button :loading="loading || analyticsLoading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>
    </PageHeader>

    <!-- 加载骨架 -->
    <template v-if="loading">
      <div class="overview-grid">
        <div v-for="index in 4" :key="index" class="card-skeleton">
          <a-skeleton active :title="false" :paragraph="{ rows: 2 }" />
        </div>
      </div>
    </template>

    <!-- 错误态：说明发生了什么 + 重试入口 -->
    <a-card v-else-if="error">
      <a-result
        status="warning"
        title="数据加载失败"
        sub-title="无法连接后端服务，请确认服务已启动后重试"
      >
        <template #extra>
          <a-button type="primary" @click="load">重新加载</a-button>
        </template>
      </a-result>
    </a-card>

    <template v-else-if="stats">
      <!-- 概览卡：v2 设计稿档位（≥1200 四列，768~1200 两列，<768 一列） -->
      <div class="overview-grid">
        <StatCard
          v-for="card in overviewCards"
          :key="card.key"
          size="large"
          :title="card.title"
          :value="card.value"
          :icon="card.icon"
          :hint="card.hint"
          :warning="card.warning"
          :spark="card.spark"
          :spark-label="card.sparkLabel"
          :delta="card.delta"
          :compare-label="compareLabel"
        />
      </div>

      <!-- 访问趋势 + 来源分布：主视觉在左、分布面板在右 -->
      <div class="dash-row dash-row--main">
        <SectionPanel
          title="访问趋势"
          description="PV / UV · 按日统计"
          :loading="analyticsLoading && trend.length === 0"
          :empty="!analyticsLoading && trend.length === 0"
          empty-title="暂无访问数据"
          :skeleton-rows="5"
        >
          <template #actions>
            <div class="legend">
              <span class="legend__item">
                <i class="legend__dot legend__dot--pv"></i>
                PV
                <b class="tabular-nums">{{ totals?.pv ?? 0 }}</b>
              </span>
              <span class="legend__item">
                <i class="legend__dot legend__dot--uv"></i>
                UV
                <b class="tabular-nums">{{ totals?.uv ?? 0 }}</b>
              </span>
            </div>
          </template>
          <BaseTrendChart
            variant="rich"
            :labels="trendLabels"
            :series="trendSeries"
            :height="248"
            :axis-titles="trend.map((point) => point.date)"
          />
        </SectionPanel>

        <SectionPanel
          title="来源分布"
          description="读者从哪里来到这里"
          :empty="sourceList.length === 0"
          empty-title="暂无来源数据"
        >
          <div v-if="sourceList.length > 0" class="source-panel">
            <div class="source-panel__total">
              <div class="source-panel__figure tabular-nums">{{ totals?.uv ?? 0 }}</div>
              <div class="source-panel__caption">独立访客</div>
            </div>
            <ul class="source-list">
              <li v-for="item in sourceList" :key="item.label" class="source-item">
                <span class="source-item__label">{{ item.label }}</span>
                <span class="source-item__bar" aria-hidden="true">
                  <i :style="{ width: `${item.percent}%` }"></i>
                </span>
                <span class="source-item__percent tabular-nums">{{ item.percent }}%</span>
              </li>
            </ul>
          </div>
        </SectionPanel>
      </div>

      <!-- 最近评论 + 计划发布 -->
      <div class="dash-row dash-row--duo">
        <SectionPanel
          title="最近评论"
          :description="`待审核 ${stats.pendingCommentCount} 条`"
          :empty="stats.recentComments.length === 0"
          empty-title="暂无评论"
        >
          <template #actions>
            <a-button type="link" size="small" @click="goComments">
              全部评论
              <template #icon><ArrowRightOutlined /></template>
            </a-button>
          </template>
          <div class="comment-list">
            <div
              v-for="item in stats.recentComments"
              :key="item.id"
              class="comment-row"
              :class="{ 'comment-row--pending': item.status === 0 }"
              role="button"
              tabindex="0"
              @click="goComments"
              @keydown.enter="goComments"
            >
              <a-avatar :size="32" class="comment-row__avatar">
                {{ initial(item.nickname) }}
              </a-avatar>
              <div class="comment-row__main">
                <div class="comment-row__head">
                  <span class="comment-row__name">{{ item.nickname }}</span>
                  <a-tag :color="COMMENT_STATUS_MAP[item.status as 0 | 1 | 2].color">
                    {{ COMMENT_STATUS_MAP[item.status as 0 | 1 | 2].text }}
                  </a-tag>
                  <span class="comment-row__time">{{ formatTime(item.createdAt) }}</span>
                </div>
                <div class="comment-row__content clamp-2">{{ item.content }}</div>
                <div class="comment-row__post">《{{ item.postTitle }}》</div>
              </div>
            </div>
          </div>
        </SectionPanel>

        <SectionPanel
          title="计划发布"
          :description="`未来 7 天内 ${scheduledPosts.length} 篇`"
          :empty="scheduledPosts.length === 0"
          empty-title="暂无计划发布的文章"
        >
          <div class="scheduled-list">
            <div
              v-for="item in scheduledPosts"
              :key="item.id"
              class="scheduled-row"
              role="button"
              tabindex="0"
              @click="goEditPost(item.id)"
              @keydown.enter="goEditPost(item.id)"
            >
              <ClockCircleOutlined class="scheduled-row__icon" />
              <div class="scheduled-row__main">
                <div class="scheduled-row__title">{{ item.title }}</div>
                <div class="scheduled-row__time tabular-nums">
                  {{ formatPlanTime(item.publishAt) }} · {{ relativePlanText(item.publishAt) }}
                </div>
              </div>
            </div>
          </div>
        </SectionPanel>
      </div>

      <!-- 快捷入口 -->
      <SectionPanel title="快捷入口" description="常用入口" class="dash-quick">
        <div class="quick-grid">
          <button
            v-for="action in quickActions"
            :key="action.key"
            type="button"
            class="quick-action"
            @click="goAction(action.to)"
          >
            <component :is="action.icon" class="quick-action__icon" />
            <span class="quick-action__title">{{ action.title }}</span>
          </button>
        </div>
      </SectionPanel>
    </template>
  </div>
</template>

<style scoped>
/* ---------- 概览卡栅格：四列 → 两列 → 一列 ---------- */
.overview-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

@media (max-width: 1200px) {
  .overview-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .overview-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

.card-skeleton {
  padding: 18px 20px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-sm);
}

/* ---------- 面板行：主行 2fr/1fr，双列行等宽 ---------- */
.dash-row {
  display: grid;
  gap: 16px;
  margin-top: 16px;
}

.dash-row--main {
  grid-template-columns: minmax(0, 2.1fr) minmax(0, 1fr);
}

.dash-row--duo {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

@media (max-width: 1100px) {
  .dash-row--main,
  .dash-row--duo {
    grid-template-columns: minmax(0, 1fr);
  }
}

.panel-empty {
  padding: 40px 0;
  text-align: center;
  font-size: 13px;
  color: var(--admin-muted);
}

/* ---------- 趋势图图例 ---------- */
.legend {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  color: var(--admin-muted);
}

.legend__item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.legend__item b {
  color: var(--admin-text);
  font-weight: 600;
}

.legend__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.legend__dot--pv {
  background: var(--admin-brand);
}

.legend__dot--uv {
  background: var(--admin-muted);
}

/* ---------- 来源分布 ---------- */
.source-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 环形图位：用同心圆示意总 UV，外圈按占比用 conic-gradient 上色（纯 CSS，无新依赖） */
.source-panel__total {
  position: relative;
  width: 148px;
  height: 148px;
  margin: 4px auto 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background:
    conic-gradient(
      from -90deg,
      var(--admin-brand) 0deg calc(var(--ring-share, 0%) * 3.6deg),
      var(--admin-surface-2) calc(var(--ring-share, 0%) * 3.6deg) 360deg
    );
}

/* 内圈挖空成环，圆心放真实数值 */
.source-panel__total::before {
  content: '';
  position: absolute;
  inset: 14px;
  border-radius: 50%;
  background: var(--admin-surface);
}

.source-panel__figure,
.source-panel__caption {
  position: relative;
}

.source-panel__figure {
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.4px;
  color: var(--admin-text);
}

.source-panel__caption {
  font-size: 12px;
  color: var(--admin-muted);
}

.source-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.source-item {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) 40px;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.source-item__label {
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-item__bar {
  height: 6px;
  border-radius: 3px;
  background: var(--admin-surface-2);
  overflow: hidden;
}

.source-item__bar i {
  display: block;
  height: 100%;
  border-radius: 3px;
  background: var(--admin-brand);
  transition: width var(--admin-dur) var(--admin-ease);
}

.source-item__percent {
  text-align: right;
  color: var(--admin-muted);
}

/* ---------- 最近评论 ---------- */
.comment-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.comment-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border-radius: var(--admin-radius-sm);
  cursor: pointer;
  transition: background-color var(--admin-dur) var(--admin-ease);
}

.comment-row:hover {
  background: var(--admin-surface-2);
}

/* 待审核行：左侧 2px warning 竖线 + 浅 warning 底 */
.comment-row--pending {
  box-shadow: inset 2px 0 0 var(--admin-warning);
  background: var(--admin-warning-bg);
}

.comment-row--pending:hover {
  background: color-mix(in srgb, var(--admin-warning) 12%, var(--admin-warning-bg));
}

.comment-row__avatar {
  flex: none;
  background: var(--admin-brand-bg);
  color: var(--admin-brand);
  font-weight: 600;
}

.comment-row__main {
  flex: 1;
  min-width: 0;
}

.comment-row__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.comment-row__name {
  font-weight: 600;
  color: var(--admin-text);
}

.comment-row__time {
  margin-left: auto;
  font-size: 12px;
  color: var(--admin-muted);
  white-space: nowrap;
}

.comment-row__content {
  margin-top: 4px;
  font-size: 13px;
  line-height: 20px;
  color: var(--admin-text);
  white-space: pre-wrap;
  word-break: break-word;
}

.comment-row__post {
  margin-top: 4px;
  font-size: 12px;
  color: var(--admin-muted);
}

/* ---------- 计划发布 ---------- */
.scheduled-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.scheduled-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: var(--admin-radius-sm);
  cursor: pointer;
  transition: background-color var(--admin-dur) var(--admin-ease);
}

.scheduled-row:hover {
  background: var(--admin-surface-2);
}

.scheduled-row__icon {
  flex: none;
  font-size: 15px;
  color: var(--admin-brand);
}

.scheduled-row__main {
  flex: 1;
  min-width: 0;
}

.scheduled-row__title {
  font-size: 14px;
  font-weight: 500;
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-row__time {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

/* ---------- 快捷入口 ---------- */
.quick-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

@media (max-width: 900px) {
  .quick-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.quick-action {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  font-family: inherit;
  font-size: 14px;
  color: var(--admin-text);
  text-align: left;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  cursor: pointer;
  transition:
    border-color var(--admin-dur) var(--admin-ease),
    background-color var(--admin-dur) var(--admin-ease),
    transform var(--admin-dur) var(--admin-ease);
}

.quick-action:hover {
  border-color: color-mix(in srgb, var(--admin-brand) 40%, var(--admin-border));
  background: var(--admin-brand-bg);
  transform: translateY(-1px);
}

.quick-action:focus-visible {
  outline: 2px solid var(--admin-brand);
  outline-offset: 1px;
}

@media (prefers-reduced-motion: reduce) {
  .quick-action,
  .quick-action:hover {
    transition: none;
    transform: none;
  }
}

.quick-action__icon {
  flex: none;
  font-size: 16px;
  color: var(--admin-brand);
}

.quick-action__title {
  font-weight: 500;
}

@media (max-width: 768px) {
  .comment-row__time {
    margin-left: 0;
  }
}
</style>
