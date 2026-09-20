<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import {
  AuditOutlined,
  ClockCircleOutlined,
  CommentOutlined,
  EditOutlined,
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
import TrendChart from '@/components/TrendChart.vue'
import PvTrendChart from '@/components/PvTrendChart.vue'
import { getStats } from '@/api/stats'
import { getAnalytics } from '@/api/analytics'
import { COMMENT_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { AnalyticsTrendPoint, Stats } from '@/types/api'

const router = useRouter()

const loading = ref(true)
const error = ref(false)
const stats = ref<Stats | null>(null)

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
  // 访问趋势（近 7 天 PV/UV）与内容统计并行加载；失败/空数据面板显示空态，不影响主视图
  void loadVisitTrend()
}

/** 近 7 天访问趋势：来自 /admin/analytics?range=7d 的 trend（契约 #67） */
const visitTrend = ref<AnalyticsTrendPoint[]>([])
const visitTrendLoading = ref(false)

async function loadVisitTrend() {
  visitTrendLoading.value = true
  try {
    const result = await getAnalytics('7d')
    visitTrend.value = result.trend
  } catch {
    visitTrend.value = []
  } finally {
    visitTrendLoading.value = false
  }
}

onMounted(load)

interface StatItem {
  key: string
  title: string
  value: number
  icon: typeof FileTextOutlined
  hint?: string
  warning?: boolean
}

/** 全部字段来自 /admin/stats 契约，辅助信息为真实字段派生，不虚构趋势 */
const cards = computed<StatItem[]>(() => {
  const s = stats.value
  if (!s) return []
  return [
    {
      key: 'postCount',
      title: '文章总数',
      value: s.postCount,
      icon: FileTextOutlined,
      hint: `含草稿 ${s.draftCount} 篇`,
    },
    {
      key: 'draftCount',
      title: '草稿',
      value: s.draftCount,
      icon: EditOutlined,
      // 草稿是文章总数的子集，"文章共 X 篇"与首卡重复；无增量信息时不显示 hint
    },
    {
      key: 'scheduledCount',
      title: '计划发布',
      value: s.scheduledCount,
      icon: ClockCircleOutlined,
      // 展示最近一条计划时间（真实字段派生），比凑数的评论数更有信息量
      hint:
        (s.scheduledPosts?.length ?? 0) > 0
          ? `最近 ${formatPlanTime(s.scheduledPosts[0].publishAt)}`
          : undefined,
    },
    {
      key: 'pendingCommentCount',
      title: '待审核评论',
      value: s.pendingCommentCount,
      icon: AuditOutlined,
      hint: `评论共 ${s.commentCount} 条`,
      warning: s.pendingCommentCount > 0,
    },
    {
      key: 'viewCount',
      title: '总浏览量',
      value: s.viewCount,
      icon: EyeOutlined,
      hint: `今日 ${s.todayPv} · 昨日 ${s.yesterdayPv}`,
    },
    {
      key: 'likeCount',
      title: '总点赞数',
      value: s.likeCount,
      icon: LikeOutlined,
      // 原先凑数显示浏览量；无点赞趋势字段，不虚构 hint
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

const trend = computed(() => stats.value?.trend ?? [])

/** 计划发布中的文章（来自 /admin/stats 契约字段） */
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
</script>

<template>
  <div class="page">
    <PageHeader title="仪表盘" description="站点内容与互动数据总览">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>
    </PageHeader>

    <!-- 加载骨架 -->
    <template v-if="loading">
      <a-row :gutter="[16, 16]">
        <a-col v-for="index in 6" :key="index" :xs="12" :sm="8" :xl="4">
          <div class="stat-card-skeleton"><a-skeleton active :title="false" :paragraph="{ rows: 2 }" /></div>
        </a-col>
      </a-row>
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
      <!-- 统计卡片：≥1200 一行 6 张，<768 两列 -->
      <a-row :gutter="[16, 16]">
        <a-col v-for="card in cards" :key="card.key" :xs="12" :sm="8" :xl="4">
          <StatCard
            :title="card.title"
            :value="card.value"
            :icon="card.icon"
            :hint="card.hint"
            :warning="card.warning"
          />
        </a-col>
      </a-row>

      <!-- 快捷操作 -->
      <a-card title="快捷操作" class="dashboard-section">
        <a-row :gutter="[16, 16]">
          <a-col v-for="action in quickActions" :key="action.key" :xs="12" :md="6">
            <div
              class="quick-action"
              role="button"
              tabindex="0"
              @click="goAction(action.to)"
              @keydown.enter="goAction(action.to)"
            >
              <component :is="action.icon" class="quick-action__icon" />
              <div class="quick-action__meta">
                <div class="quick-action__title">{{ action.title }}</div>
                <div class="quick-action__desc">{{ action.desc }}</div>
              </div>
            </div>
          </a-col>
        </a-row>
      </a-card>

      <!-- 趋势图 -->
      <a-card title="近 7 天趋势" class="dashboard-section">
        <TrendChart v-if="trend.length > 0" :data="trend" />
        <a-empty v-else description="暂无趋势数据" class="dashboard-empty" />
      </a-card>

      <!-- 访问趋势：近 7 天 PV/UV（来自 /admin/analytics），失败或空数据显示空态 -->
      <a-card title="访问趋势（近 7 天）" class="dashboard-section">
        <div v-if="visitTrendLoading" class="dashboard-spin">
          <a-spin />
        </div>
        <PvTrendChart v-else-if="visitTrend.length > 0" :data="visitTrend" />
        <a-empty v-else description="暂无访问趋势数据" class="dashboard-empty" />
      </a-card>

      <!-- 计划发布 + 最近评论：桌面并排，移动端上下堆叠 -->
      <a-row :gutter="[16, 16]" class="dashboard-section">
        <a-col :xs="24" :lg="12">
          <a-card title="计划发布" class="dashboard-duo">
            <a-empty
              v-if="scheduledPosts.length === 0"
              description="暂无计划发布的文章"
              class="dashboard-empty"
            />
            <div v-else class="scheduled-posts">
              <div
                v-for="item in scheduledPosts"
                :key="item.id"
                class="scheduled-post"
                role="button"
                tabindex="0"
                @click="goEditPost(item.id)"
                @keydown.enter="goEditPost(item.id)"
              >
                <ClockCircleOutlined class="scheduled-post__icon" />
                <div class="scheduled-post__main">
                  <div class="scheduled-post__title">{{ item.title }}</div>
                  <div class="scheduled-post__time tabular-nums">
                    {{ formatPlanTime(item.publishAt) }} · {{ relativePlanText(item.publishAt) }}
                  </div>
                </div>
              </div>
            </div>
          </a-card>
        </a-col>
        <a-col :xs="24" :lg="12">
          <a-card title="最近评论" class="dashboard-duo">
            <a-empty v-if="stats.recentComments.length === 0" description="暂无评论" class="dashboard-empty" />
            <div v-else class="recent-comments">
              <div
                v-for="item in stats.recentComments"
                :key="item.id"
                class="recent-comment"
                :class="{ 'recent-comment--pending': item.status === 0 }"
                @click="goComments"
              >
                <a-avatar :size="32" class="recent-comment__avatar">{{ initial(item.nickname) }}</a-avatar>
                <div class="recent-comment__main">
                  <div class="recent-comment__head">
                    <span class="recent-comment__name">{{ item.nickname }}</span>
                    <a-tag :color="COMMENT_STATUS_MAP[item.status as 0 | 1 | 2].color">
                      {{ COMMENT_STATUS_MAP[item.status as 0 | 1 | 2].text }}
                    </a-tag>
                    <span class="recent-comment__time">{{ formatTime(item.createdAt) }}</span>
                  </div>
                  <div class="recent-comment__content clamp-2">{{ item.content }}</div>
                  <div class="recent-comment__post">《{{ item.postTitle }}》</div>
                </div>
              </div>
            </div>
          </a-card>
        </a-col>
      </a-row>
    </template>
  </div>
</template>

<style scoped>
.dashboard-section {
  margin-top: 16px;
}

/* 并排面板：等高卡片 */
.dashboard-duo {
  height: 100%;
}

/* 计划发布面板 */
.scheduled-posts {
  display: flex;
  flex-direction: column;
}

.scheduled-post {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: var(--admin-radius-sm);
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.scheduled-post:hover {
  background: var(--admin-surface-2);
}

.scheduled-post__icon {
  flex: none;
  font-size: 16px;
  color: var(--admin-brand);
}

.scheduled-post__main {
  flex: 1;
  min-width: 0;
}

.scheduled-post__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-post__time {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

.dashboard-empty {
  padding: 24px 0;
}

.dashboard-spin {
  display: flex;
  justify-content: center;
  padding: 48px 0;
}

.stat-card-skeleton {
  padding: 16px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-sm);
}

/* 快捷操作：一排四格轻量入口 */
.quick-action {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  cursor: pointer;
  transition:
    border-color var(--admin-dur) var(--admin-ease),
    background-color var(--admin-dur) var(--admin-ease),
    transform var(--admin-dur) var(--admin-ease);
}

.quick-action:hover {
  border-color: color-mix(in srgb, var(--admin-brand) 35%, var(--admin-border));
  background: var(--admin-brand-bg);
  transform: translateY(-1px);
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
  font-size: 18px;
  color: var(--admin-brand);
  transition: transform var(--admin-dur) var(--admin-ease);
}

/* 悬停时图标轻微放大，给入口一点"活"的感觉 */
.quick-action:hover .quick-action__icon {
  transform: scale(1.12);
}

.quick-action__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--admin-text);
}

.quick-action__desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

/* 最近评论列表 */
.recent-comments {
  display: flex;
  flex-direction: column;
}

.recent-comment {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border-radius: var(--admin-radius-sm);
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.recent-comment + .recent-comment {
  margin-top: 4px;
}

.recent-comment:hover {
  background: var(--admin-surface-2);
}

/* 待审核行：左侧 2px warning 竖线 + 浅 warning 底 */
.recent-comment--pending {
  box-shadow: inset 2px 0 0 var(--admin-warning);
  background: var(--admin-warning-bg);
}

.recent-comment--pending:hover {
  /* token 混色深化，替代写死的浅橙（暗色模式下不再是刺眼亮块） */
  background: color-mix(in srgb, var(--admin-warning) 10%, var(--admin-warning-bg));
}

.recent-comment__avatar {
  flex: none;
  background: var(--admin-brand-bg);
  color: var(--admin-brand);
  font-weight: 600;
}

.recent-comment__main {
  flex: 1;
  min-width: 0;
}

.recent-comment__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.recent-comment__name {
  font-weight: 600;
  color: var(--admin-text);
}

.recent-comment__time {
  margin-left: auto;
  font-size: 12px;
  color: var(--admin-muted);
}

.recent-comment__content {
  margin-top: 4px;
  font-size: 13px;
  line-height: 20px;
  color: var(--admin-text);
  white-space: pre-wrap;
  word-break: break-word;
}

.recent-comment__post {
  margin-top: 4px;
  font-size: 12px;
  color: var(--admin-muted);
}

@media (max-width: 768px) {
  /* 手机上时间换行展示 */
  .recent-comment__time {
    margin-left: 0;
  }
}
</style>
