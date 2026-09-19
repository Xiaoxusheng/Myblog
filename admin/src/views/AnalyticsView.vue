<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { TableColumnsType } from 'ant-design-vue'
import { EyeOutlined, ReloadOutlined, TeamOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { getSearchStats } from '@/api/analytics'
import StatCard from '@/components/StatCard.vue'
import PvTrendChart from '@/components/PvTrendChart.vue'
import DistributionBars from '@/components/DistributionBars.vue'
import { getAnalytics } from '@/api/analytics'
import type { AnalyticsData, AnalyticsDistItem, AnalyticsRange } from '@/types/api'

const router = useRouter()

const rangeOptions: { label: string; value: AnalyticsRange }[] = [
  { label: '今日', value: 'today' },
  { label: '近 7 天', value: '7d' },
  { label: '近 30 天', value: '30d' },
  { label: '近 90 天', value: '90d' },
]

const range = ref<AnalyticsRange>('7d')
const loading = ref(true)
const error = ref(false)
const data = ref<AnalyticsData | null>(null)

async function load() {
  loading.value = true
  error.value = false
  try {
    data.value = await getAnalytics(range.value)
  } catch {
    // 拦截器已 toast 具体原因；无数据时展示重试面板
    error.value = true
  } finally {
    loading.value = false
  }
}

function onRangeChange() {
  void load()
  void loadSearchStats()
}

/** 来源中文映射（契约取值 direct/search/github/social/other，未知值原样显示） */
const SOURCE_MAP: Record<string, string> = {
  direct: '直接访问',
  search: '搜索引擎',
  github: 'GitHub',
  social: '社交平台',
  other: '其他',
}

/** 设备中文映射（契约取值 desktop/mobile/tablet） */
const DEVICE_MAP: Record<string, string> = {
  desktop: '桌面',
  mobile: '移动',
  tablet: '平板',
}

const trend = computed(() => data.value?.trend ?? [])

const sourceItems = computed<AnalyticsDistItem[]>(() =>
  (data.value?.sources ?? []).map((item) => ({
    name: SOURCE_MAP[item.source] ?? item.source,
    pv: item.pv,
  })),
)

const deviceItems = computed<AnalyticsDistItem[]>(() =>
  (data.value?.devices ?? []).map((item) => ({
    name: DEVICE_MAP[item.device] ?? item.device,
    pv: item.pv,
  })),
)

const browserItems = computed<AnalyticsDistItem[]>(() =>
  (data.value?.browsers ?? []).map((item) => ({ name: item.browser, pv: item.pv })),
)

const osItems = computed<AnalyticsDistItem[]>(() =>
  (data.value?.oses ?? []).map((item) => ({ name: item.os, pv: item.pv })),
)

// ---------- 搜索统计（契约 #85，range 联动） ----------
const searchStats = ref<{ keyword: string; count: number; noResultCount: number }[]>([])
const searchLoading = ref(false)

async function loadSearchStats() {
  if (range.value === 'today') {
    searchStats.value = [] // 后端搜索统计最小粒度为 7d
    return
  }
  searchLoading.value = true
  try {
    const result = await getSearchStats({ range: range.value as '7d' | '30d' | '90d' })
    searchStats.value = result.list
  } finally {
    searchLoading.value = false
  }
}

const searchColumns: TableColumnsType = [
  { title: '关键词', dataIndex: 'keyword', key: 'keyword', ellipsis: true },
  { title: '搜索次数', dataIndex: 'count', key: 'count', width: 120 },
  { title: '无结果次数', dataIndex: 'noResultCount', key: 'noResultCount', width: 120 },
]

const topColumns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: 'PV', dataIndex: 'pv', key: 'pv', width: 90 },
  { title: 'UV', dataIndex: 'uv', key: 'uv', width: 90 },
  { title: '点赞', dataIndex: 'likeCount', key: 'likeCount', width: 80 },
  { title: '评论', dataIndex: 'commentCount', key: 'commentCount', width: 80 },
]

function goEditPost(postId: number) {
  void router.push(`/posts/edit/${postId}`)
}

onMounted(() => {
  void load()
  void loadSearchStats()
})
</script>

<template>
  <div class="page">
    <PageHeader title="访问分析" description="站点浏览量、访客与来源/设备分布">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-segmented v-model:value="range" :options="rangeOptions" @change="onRangeChange" />
      </template>
    </PageHeader>

    <!-- 首次加载骨架 -->
    <template v-if="loading && !data">
      <a-row :gutter="[16, 16]">
        <a-col :xs="12" :xl="6">
          <div class="stat-card-skeleton"><a-skeleton active :title="false" :paragraph="{ rows: 2 }" /></div>
        </a-col>
        <a-col :xs="12" :xl="6">
          <div class="stat-card-skeleton"><a-skeleton active :title="false" :paragraph="{ rows: 2 }" /></div>
        </a-col>
      </a-row>
      <div class="stat-card-skeleton analytics-chart-skeleton">
        <a-skeleton active :title="false" :paragraph="{ rows: 6 }" />
      </div>
    </template>

    <!-- 错误态：说明发生了什么 + 重试入口 -->
    <a-card v-else-if="error && !data">
      <a-result
        status="warning"
        title="数据加载失败"
        sub-title="无法获取访问分析数据，请确认服务已启动后重试"
      >
        <template #extra>
          <a-button type="primary" @click="load">重新加载</a-button>
        </template>
      </a-result>
    </a-card>

    <template v-else-if="data">
      <a-spin :spinning="loading" tip="加载中...">
        <!-- PV / UV 概览 -->
        <a-row :gutter="[16, 16]">
          <a-col :xs="12" :xl="6">
            <StatCard title="浏览量 PV" :value="data.totals.pv" :icon="EyeOutlined" />
          </a-col>
          <a-col :xs="12" :xl="6">
            <StatCard title="访客数 UV" :value="data.totals.uv" :icon="TeamOutlined" />
          </a-col>
        </a-row>

        <!-- PV/UV 趋势 -->
        <a-card title="访问趋势" class="analytics-section">
          <PvTrendChart v-if="trend.length > 0" :data="trend" />
          <a-empty v-else description="暂无趋势数据" class="analytics-empty" />
        </a-card>

        <!-- 热门文章 -->
        <a-card title="热门文章" class="analytics-section">
          <a-table
            :columns="topColumns"
            :data-source="data.topPosts"
            :loading="loading"
            :pagination="false"
            row-key="postId"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'title'">
                <a class="top-post-title" @click="goEditPost(record.postId)">{{ record.title }}</a>
              </template>
              <template v-else-if="column.key === 'pv'">
                <span class="tabular-nums">{{ record.pv }}</span>
              </template>
              <template v-else-if="column.key === 'uv'">
                <span class="tabular-nums">{{ record.uv }}</span>
              </template>
              <template v-else-if="column.key === 'likeCount'">
                <span class="tabular-nums">{{ record.likeCount }}</span>
              </template>
              <template v-else-if="column.key === 'commentCount'">
                <span class="tabular-nums">{{ record.commentCount }}</span>
              </template>
            </template>
          </a-table>
        </a-card>

        <!-- 搜索统计 -->
        <a-card title="搜索统计" class="analytics-section">
          <a-spin :spinning="searchLoading">
            <a-table
              v-if="searchStats.length > 0"
              :columns="searchColumns"
              :data-source="searchStats"
              :pagination="false"
              row-key="keyword"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'count'">
                  <span class="tabular-nums">{{ record.count }}</span>
                </template>
                <template v-else-if="column.key === 'noResultCount'">
                  <span class="tabular-nums" :class="{ 'search-no-result': record.noResultCount > 0 }">
                    {{ record.noResultCount }}
                  </span>
                </template>
              </template>
            </a-table>
            <a-empty v-else description="暂无搜索记录" class="analytics-empty" />
          </a-spin>
        </a-card>

        <!-- 来源 / 设备 / 浏览器 / 操作系统 分布 -->
        <a-row :gutter="[16, 16]" class="analytics-section">
          <a-col :xs="24" :sm="12" :xl="6">
            <a-card title="来源分布" class="analytics-dist">
              <DistributionBars v-if="sourceItems.length > 0" :items="sourceItems" />
              <a-empty v-else description="暂无数据" class="analytics-empty" />
            </a-card>
          </a-col>
          <a-col :xs="24" :sm="12" :xl="6">
            <a-card title="设备分布" class="analytics-dist">
              <DistributionBars v-if="deviceItems.length > 0" :items="deviceItems" />
              <a-empty v-else description="暂无数据" class="analytics-empty" />
            </a-card>
          </a-col>
          <a-col :xs="24" :sm="12" :xl="6">
            <a-card title="浏览器" class="analytics-dist">
              <DistributionBars v-if="browserItems.length > 0" :items="browserItems" />
              <a-empty v-else description="暂无数据" class="analytics-empty" />
            </a-card>
          </a-col>
          <a-col :xs="24" :sm="12" :xl="6">
            <a-card title="操作系统" class="analytics-dist">
              <DistributionBars v-if="osItems.length > 0" :items="osItems" />
              <a-empty v-else description="暂无数据" class="analytics-empty" />
            </a-card>
          </a-col>
        </a-row>
      </a-spin>
    </template>
  </div>
</template>

<style scoped>
.analytics-section {
  margin-top: 16px;
}

.analytics-empty {
  padding: 24px 0;
}

.analytics-dist {
  height: 100%;
}

.top-post-title {
  color: var(--admin-text);
}

.top-post-title:hover {
  color: var(--admin-brand);
}

.stat-card-skeleton {
  padding: 16px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-sm);
}

.analytics-chart-skeleton {
  margin-top: 16px;
}

.search-no-result {
  color: var(--admin-warning, #d46b08);
  font-weight: 500;
}
</style>
