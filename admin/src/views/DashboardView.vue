<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { getStats } from '@/api/stats'
import TrendChart from '@/components/TrendChart.vue'
import { COMMENT_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { Stats } from '@/types/api'

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

interface StatCard {
  key: string
  title: string
  value: number
  warning?: boolean
}

const cards = computed<StatCard[]>(() => {
  const s = stats.value
  if (!s) return []
  return [
    { key: 'postCount', title: '文章总数', value: s.postCount },
    { key: 'draftCount', title: '草稿', value: s.draftCount },
    { key: 'commentCount', title: '评论总数', value: s.commentCount },
    { key: 'pendingCommentCount', title: '待审评论', value: s.pendingCommentCount, warning: s.pendingCommentCount > 0 },
    { key: 'viewCount', title: '总浏览量', value: s.viewCount },
    { key: 'likeCount', title: '总点赞数', value: s.likeCount },
    { key: 'linkCount', title: '友链数量', value: s.linkCount },
  ]
})

const trend = computed(() => stats.value?.trend ?? [])
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div class="page-header__meta">
        <a-typography-title :level="4" style="margin: 0">仪表盘</a-typography-title>
        <a-typography-text type="secondary" class="page-header__desc">
          站点内容与互动数据总览
        </a-typography-text>
      </div>
      <a-space>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </a-space>
    </div>

    <!-- 加载骨架 -->
    <template v-if="loading">
      <a-row :gutter="[16, 16]">
        <a-col v-for="index in 7" :key="index" :xs="12" :sm="8" :md="6">
          <a-card><a-skeleton active :title="false" :paragraph="{ rows: 2 }" /></a-card>
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
      <!-- 统计卡片 -->
      <a-row :gutter="[16, 16]">
        <a-col v-for="card in cards" :key="card.key" :xs="12" :sm="8" :md="6">
          <a-card>
            <a-statistic
              :title="card.title"
              :value="card.value"
              :value-style="card.warning ? { color: '#fa8c16' } : undefined"
            />
          </a-card>
        </a-col>
      </a-row>

      <!-- 趋势图 -->
      <a-card title="近 7 天趋势" class="dashboard-section">
        <TrendChart :data="trend" />
      </a-card>

      <!-- 最近评论 -->
      <a-card title="最近评论" class="dashboard-section">
        <a-list :data-source="stats.recentComments" :loading="false">
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta>
                <template #title>
                  <a-space :size="8" wrap>
                    <span>{{ item.nickname }}</span>
                    <a-tag :color="COMMENT_STATUS_MAP[item.status as 0 | 1 | 2].color">
                      {{ COMMENT_STATUS_MAP[item.status as 0 | 1 | 2].text }}
                    </a-tag>
                  </a-space>
                </template>
                <template #description>
                  <div class="recent-comment__content">{{ item.content }}</div>
                  <a-typography-text type="secondary" style="font-size: 12px">
                    《{{ item.postTitle }}》 · {{ formatTime(item.createdAt) }}
                  </a-typography-text>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
      </a-card>
    </template>
  </div>
</template>

<style scoped>
.dashboard-section {
  margin-top: 16px;
}

.recent-comment__content {
  color: rgba(0, 0, 0, 0.88);
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
