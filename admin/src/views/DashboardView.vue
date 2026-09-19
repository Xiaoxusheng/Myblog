<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  AuditOutlined,
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
import { getStats } from '@/api/stats'
import { COMMENT_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { Stats } from '@/types/api'

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
      hint: `文章共 ${s.postCount} 篇`,
    },
    {
      key: 'commentCount',
      title: '评论总数',
      value: s.commentCount,
      icon: CommentOutlined,
      hint: `待审核 ${s.pendingCommentCount} 条`,
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
      hint: `获赞 ${s.likeCount}`,
    },
    {
      key: 'likeCount',
      title: '总点赞数',
      value: s.likeCount,
      icon: LikeOutlined,
      hint: `浏览 ${s.viewCount}`,
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

      <!-- 最近评论 -->
      <a-card title="最近评论" class="dashboard-section">
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
    </template>
  </div>
</template>

<style scoped>
.dashboard-section {
  margin-top: 16px;
}

.dashboard-empty {
  padding: 24px 0;
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
  transition: border-color 0.2s ease, background-color 0.2s ease;
}

.quick-action:hover {
  border-color: var(--admin-brand);
  background: #fafcff;
}

.quick-action__icon {
  flex: none;
  font-size: 18px;
  color: var(--admin-brand);
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
  background: #fff4dd;
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
