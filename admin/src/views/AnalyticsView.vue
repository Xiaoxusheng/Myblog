<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { TableColumnsType } from 'ant-design-vue'
import { EyeOutlined, TeamOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import SectionPanel from '@/components/SectionPanel.vue'
import StatCard from '@/components/StatCard.vue'
import DistributionBars from '@/components/DistributionBars.vue'
import TrendPanel from '@/components/TrendPanel.vue'
import TopPostsPanel from '@/components/TopPostsPanel.vue'
import AnalyticsToolbar from '@/components/AnalyticsToolbar.vue'
import { getAnalytics, getSearchStats } from '@/api/analytics'
import type { SearchStatItem } from '@/api/analytics'
import { ApiError, isRequestCanceled } from '@/api/http'
import { computeDelta, RANGE_META } from '@/utils/analytics'
import { formatCount, formatNumber } from '@/utils/format'
import type { AnalyticsData, AnalyticsDistItem, AnalyticsRange } from '@/types/api'

/**
 * 访问分析页。
 *
 * 分层：页面只负责「取数 + 请求编排 + 组装」，视觉与状态呈现交给
 * StatCard / TrendPanel / TopPostsPanel / SectionPanel，
 * 图表渲染交给 PvTrendChart → BaseTrendChart。
 *
 * 数据口径：全部来自 /admin/analytics 真实响应，缺失即降级隐藏，不做任何占位数值。
 */

const router = useRouter()

// ---------------------------------------------------------------- 状态
const range = ref<AnalyticsRange>('7d')
const data = ref<AnalyticsData | null>(null)
const updatedAt = ref<Date | null>(null)

/** 请求在途：已有内容时保留展示，只做轻量同步提示，避免整页闪回骨架 */
const fetching = ref(false)
/** 失败且没有任何可展示数据 → 页级错误态 */
const fatalError = ref(false)
/** 已有数据但本次同步失败 → 行内提示 + 保留上一次内容 */
const staleError = ref(false)

type ErrorKind = 'offline' | 'forbidden' | 'param' | 'network'
const errorKind = ref<ErrorKind>('network')

const ERROR_COPY: Record<ErrorKind, { title: string; description: string }> = {
  offline: {
    title: '网络已断开',
    description: '当前无法与服务端通信，页面保留最近一次成功同步的数据。',
  },
  forbidden: {
    title: '没有访问分析权限',
    description: '当前账号无权查看站点访问数据，请联系管理员开通后再试。',
  },
  param: {
    title: '时间范围不受支持',
    description: '服务端不接受该时间范围，请切换到其他范围后重试。',
  },
  network: {
    title: '数据加载失败',
    description: '无法获取访问分析数据，请确认后端服务已启动后重试。',
  },
}

const firstLoading = computed(() => fetching.value && !data.value)
const rangeMeta = computed(() => RANGE_META[range.value])
const compareLabel = computed(() => rangeMeta.value.compareLabel)

/** 断网状态：浏览器事件驱动，恢复联网后自动补一次同步 */
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)

// ---------------------------------------------------------------- 请求编排
let mainController: AbortController | null = null
let searchController: AbortController | null = null

function classifyError(error: unknown): ErrorKind {
  if (!online.value) return 'offline'
  if (error instanceof ApiError) {
    if (error.code === 10003) return 'forbidden'
    if (error.code === 10001) return 'param'
  }
  return 'network'
}

/**
 * 加载主数据。同一时刻只保留一个在途请求：发起前中止上一个，
 * 快速连点时间范围不会产生重复请求，也不会因旧响应后到而覆盖新数据。
 */
async function loadMain(): Promise<void> {
  mainController?.abort()
  const controller = new AbortController()
  mainController = controller

  fetching.value = true
  fatalError.value = false
  staleError.value = false

  try {
    const result = await getAnalytics(range.value, controller.signal)
    if (controller.signal.aborted) return
    data.value = result
    updatedAt.value = new Date()
  } catch (error) {
    // 主动中止属于正常控制流，静默丢弃
    if (isRequestCanceled(error) || controller.signal.aborted) return
    errorKind.value = classifyError(error)
    if (data.value) staleError.value = true
    else fatalError.value = true
  } finally {
    // 已被更新的请求接管时不再改动 loading，避免状态被旧请求提前收尾
    if (mainController === controller) {
      mainController = null
      fetching.value = false
    }
  }
}

// ---------------------------------------------------------------- 搜索统计（次级面板）
const searchRows = ref<SearchStatItem[]>([])
const searchLoading = ref(false)
const searchError = ref(false)

/** 该范围是否支持搜索统计（后端最小粒度 7d） */
const searchSupported = computed(() => rangeMeta.value.supportsSearch)

async function loadSearch(): Promise<void> {
  // 先中止上一轮，再决定是否需要发起：切到「今日」时同样要丢弃在途请求
  searchController?.abort()
  searchController = null

  if (!searchSupported.value) {
    searchRows.value = []
    searchError.value = false
    searchLoading.value = false
    return
  }

  const controller = new AbortController()
  searchController = controller
  searchLoading.value = true
  searchError.value = false

  try {
    const result = await getSearchStats(
      { range: range.value as '7d' | '30d' | '90d' },
      controller.signal,
    )
    if (controller.signal.aborted) return
    searchRows.value = result.list
  } catch (error) {
    // 接口标记 silent，不弹全局 toast；此处仅局部降级，避免未捕获的 rejection
    if (isRequestCanceled(error) || controller.signal.aborted) return
    searchRows.value = []
    searchError.value = true
  } finally {
    if (searchController === controller) {
      searchController = null
      searchLoading.value = false
    }
  }
}

function reloadAll(): void {
  void loadMain()
  void loadSearch()
}

/** 时间范围变化即重新取数；loadMain/loadSearch 内部会中止在途请求，天然去重 */
watch(range, () => reloadAll())

// ---------------------------------------------------------------- 联网状态
function onOnline(): void {
  online.value = true
  // 断网期间失败的同步，恢复后自动补上，用户无需手动刷新
  if (fatalError.value || staleError.value) reloadAll()
}

function onOffline(): void {
  online.value = false
}

onMounted(() => {
  window.addEventListener('online', onOnline)
  window.addEventListener('offline', onOffline)
  reloadAll()
})

onBeforeUnmount(() => {
  window.removeEventListener('online', onOnline)
  window.removeEventListener('offline', onOffline)
  mainController?.abort()
  searchController?.abort()
  mainController = null
  searchController = null
})

// ---------------------------------------------------------------- 派生数据
const trend = computed(() => data.value?.trend ?? [])
const topPosts = computed(() => data.value?.topPosts ?? [])

/** 范围内是否完全没有数据点——用于区分「真实为 0」与「没有数据」 */
const noDataPoints = computed(() => !!data.value && trend.value.length === 0)

const pvValue = computed(() => data.value?.totals.pv ?? 0)
const uvValue = computed(() => data.value?.totals.uv ?? 0)

const pvDelta = computed(() => computeDelta(pvValue.value, data.value?.prevTotals?.pv))
const uvDelta = computed(() => computeDelta(uvValue.value, data.value?.prevTotals?.uv))

const pvSpark = computed(() => trend.value.map((point) => point.pv))
const uvSpark = computed(() => trend.value.map((point) => point.uv))

const pvHint = computed(() => {
  if (!data.value || noDataPoints.value) return '所选范围暂无访问数据'
  if (uvValue.value <= 0) return `总计 ${formatNumber(pvValue.value)} 次浏览`
  return `人均 ${(pvValue.value / uvValue.value).toFixed(1)} 次 / 人`
})

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

/** 四个分布面板：统一在此声明，避免模板里重复四段几乎相同的结构 */
const distributionPanels = computed(() => [
  {
    key: 'sources',
    title: '来源分布',
    hint: '访问入口归类',
    items: sourceItems.value,
    emptyHint: '所选范围内还没有可归类的来源记录',
  },
  {
    key: 'devices',
    title: '设备分布',
    hint: '按访问终端归类',
    items: deviceItems.value,
    emptyHint: '所选范围内还没有设备信息记录',
  },
  {
    key: 'browsers',
    title: '浏览器',
    hint: '按 UA 解析结果归类',
    items: browserItems.value,
    emptyHint: '所选范围内还没有浏览器信息记录',
  },
  {
    key: 'oses',
    title: '操作系统',
    hint: '按 UA 解析结果归类',
    items: osItems.value,
    emptyHint: '所选范围内还没有系统信息记录',
  },
])

// ---------------------------------------------------------------- 交互
const searchColumns: TableColumnsType = [
  { title: '关键词', dataIndex: 'keyword', key: 'keyword', ellipsis: true },
  { title: '搜索次数', dataIndex: 'count', key: 'count', width: 110, align: 'right' },
  { title: '无结果', dataIndex: 'noResultCount', key: 'noResultCount', width: 100, align: 'right' },
]

function goEditPost(postId: number): void {
  void router.push(`/posts/edit/${postId}`)
}

function goPostList(): void {
  void router.push('/posts')
}
</script>

<template>
  <div class="page analytics-page">
    <PageHeader title="访问分析" description="站点浏览量、独立访客与来源 / 设备分布">
      <template #actions>
        <AnalyticsToolbar
          v-model="range"
          :refreshing="fetching"
          :updated-at="updatedAt"
          @refresh="reloadAll"
        />
      </template>
    </PageHeader>

    <!-- 断网：先说明链路状态，再谈数据 -->
    <a-alert
      v-if="!online"
      class="analytics-page__alert"
      type="warning"
      show-icon
      message="网络已断开"
      :description="ERROR_COPY.offline.description"
    />

    <!-- 同步失败但仍有历史内容：保留内容 + 行内提示，不用空白页惩罚用户 -->
    <a-alert
      v-else-if="staleError"
      class="analytics-page__alert"
      type="warning"
      show-icon
      :message="ERROR_COPY[errorKind].title"
    >
      <template #description>
        <span>{{ ERROR_COPY[errorKind].description }}</span>
        <a-button class="analytics-page__alert-action" type="link" size="small" @click="reloadAll">
          重试
        </a-button>
      </template>
    </a-alert>

    <!-- 页级失败：无任何可展示数据，给出原因与恢复动作 -->
    <SectionPanel
      v-if="fatalError && !data"
      :title="ERROR_COPY[errorKind].title"
      :error="true"
      :error-text="ERROR_COPY[errorKind].description"
      @retry="reloadAll"
    />

    <!-- 内容区：首屏用各区块自带的加载态占位，数据到位不跳版 -->
    <div v-else class="analytics-stack" :aria-busy="fetching">
      <a-row :gutter="[16, 16]">
        <a-col :xs="24" :md="12">
          <StatCard
            title="浏览量 PV"
            size="large"
            :value="formatCount(pvValue)"
            :value-title="formatNumber(pvValue)"
            :icon="EyeOutlined"
            :loading="firstLoading"
            :empty="noDataPoints"
            :delta="pvDelta"
            :compare-label="compareLabel"
            :spark="pvSpark"
            :spark-label="`${rangeMeta.label}浏览量走势`"
            :hint="pvHint"
          />
        </a-col>
        <a-col :xs="24" :md="12">
          <StatCard
            title="访客数 UV"
            size="large"
            :value="formatCount(uvValue)"
            :value-title="formatNumber(uvValue)"
            :icon="TeamOutlined"
            :loading="firstLoading"
            :empty="noDataPoints"
            :delta="uvDelta"
            :compare-label="compareLabel"
            :spark="uvSpark"
            :spark-label="`${rangeMeta.label}访客走势`"
            hint="按访客匿名指纹去重"
          />
        </a-col>
      </a-row>

      <TrendPanel
        :trend="trend"
        :loading="firstLoading"
        :delta="pvDelta"
        :compare-label="compareLabel"
      />

      <TopPostsPanel
        :items="topPosts"
        :loading="firstLoading"
        :has-site-traffic="pvValue > 0"
        @select="goEditPost"
        @browse="goPostList"
      />

      <!-- 站内搜索统计（次级面板，失败独立降级，不影响主数据区） -->
      <SectionPanel
        title="搜索统计"
        :description="
          searchSupported
            ? '站内搜索关键词，按搜索次数降序（最多 20 条）'
            : '后端搜索统计的最小粒度为 7 天'
        "
        flush
        :loading="firstLoading || searchLoading"
        :error="searchError"
        :empty="!searchError && !firstLoading && !searchLoading && searchRows.length === 0"
        :empty-title="searchSupported ? '暂无搜索记录' : '今日范围不提供搜索统计'"
        :empty-hint="
          searchSupported
            ? '所选范围内还没有站内搜索行为'
            : '切换到「近 7 天」及以上范围即可查看关键词排行'
        "
        :skeleton-rows="4"
        @retry="loadSearch"
      >
        <a-table
          class="table-quiet search-stats__table"
          :columns="searchColumns"
          :data-source="searchRows"
          :pagination="false"
          row-key="keyword"
          size="middle"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'keyword'">
              <a-tooltip :title="record.keyword" placement="topLeft">
                <span class="analytics-page__keyword">{{ record.keyword }}</span>
              </a-tooltip>
            </template>
            <template v-else-if="column.key === 'count'">
              <span class="tabular-nums">{{ formatCount(record.count) }}</span>
            </template>
            <template v-else-if="column.key === 'noResultCount'">
              <span
                class="tabular-nums"
                :class="{ 'analytics-page__no-result': record.noResultCount > 0 }"
                :title="record.noResultCount > 0 ? '该关键词没有命中任何内容，可考虑补充相关文章' : undefined"
              >
                {{ formatCount(record.noResultCount) }}
              </span>
            </template>
          </template>
        </a-table>
      </SectionPanel>

      <!-- 来源 / 设备 / 浏览器 / 系统：次级面板，浅底无投影，与主面板形成层次节奏 -->
      <a-row :gutter="[16, 16]">
        <a-col v-for="panel in distributionPanels" :key="panel.key" :xs="24" :sm="12" :xl="6">
          <SectionPanel
            variant="plain"
            :title="panel.title"
            :description="panel.hint"
            :loading="firstLoading"
            :empty="!firstLoading && panel.items.length === 0"
            :empty-title="`暂无${panel.title}数据`"
            :empty-hint="panel.emptyHint"
            :skeleton-rows="3"
          >
            <DistributionBars :items="panel.items" />
          </SectionPanel>
        </a-col>
      </a-row>
    </div>
  </div>
</template>

<style scoped>
/* 宽度与其它页统一由 .page 控制（此前这里单独设 1560px，会造成切页时宽度跳变） */

.analytics-page__alert {
  margin-bottom: var(--admin-space-4);
}

.analytics-page__alert-action {
  padding: 0;
  height: auto;
  margin-inline-start: var(--admin-space-2);
}

/* 模块节奏：统一用 gap 控制区块间距，模块内部不再各自写 margin-top */
.analytics-stack {
  display: flex;
  flex-direction: column;
  gap: var(--admin-space-4);
}

.analytics-page__keyword {
  display: block;
  overflow: hidden;
  font-size: 14px;
  color: var(--admin-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 无结果的关键词是需要运营动作的信号，用警示色点名 */
.analytics-page__no-result {
  font-weight: 600;
  color: var(--admin-warning-text);
}

@media (max-width: 768px) {
  .analytics-stack {
    gap: var(--admin-space-3);
  }
}
</style>
