<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { TableColumnsType } from 'ant-design-vue'
import {
  CommentOutlined,
  EyeOutlined,
  LikeOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
} from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import StatCard from '@/components/StatCard.vue'
import PvTrendChart from '@/components/PvTrendChart.vue'
import DistributionBars from '@/components/DistributionBars.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import BatchBar from '@/components/BatchBar.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { deletePost, getPosts, updatePostStatus } from '@/api/posts'
import { getPostAnalytics } from '@/api/analytics'
import { getCategories } from '@/api/taxonomy'
import { POST_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type {
  AdminPostItem,
  AnalyticsDistItem,
  Category,
  PostAnalyticsData,
  PostAnalyticsRange,
  PostStatus,
} from '@/types/api'

const route = useRoute()
const router = useRouter()
const { message, modal } = useFeedback()

const categories = ref<Category[]>([])

// 筛选条件：keyword 需回车/点击搜索才生效
const keywordInput = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const keyword = ref(keywordInput.value)
const status = ref<PostStatus | ''>(
  route.query.status !== undefined && route.query.status !== '' ? (Number(route.query.status) as PostStatus) : '',
)
const categoryId = ref<number | ''>(
  route.query.categoryId !== undefined && route.query.categoryId !== '' ? Number(route.query.categoryId) : '',
)

const {
  loading,
  error,
  list,
  page,
  pageSize,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<AdminPostItem>(
  ({ page, pageSize }) =>
    getPosts({
      keyword: keyword.value || undefined,
      status: status.value === '' ? undefined : status.value,
      categoryId: categoryId.value === '' ? undefined : categoryId.value,
      page,
      pageSize,
    }),
  {
    initialPage: Number(route.query.page) || 1,
  },
)

const columns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', key: 'title', width: 240, ellipsis: true },
  { title: '分类', dataIndex: ['category', 'name'], key: 'category', width: 110 },
  { title: '标签', key: 'tags', width: 160 },
  { title: '状态', key: 'status', width: 120 },
  { title: '浏览/评论', key: 'stats', width: 90 },
  {
    title: '发布时间',
    key: 'publishedAt',
    width: 170,
    // 服务端分页下仅对当前页排序(接口无排序参数),列头有 tooltip 说明
    sorter: (a: AdminPostItem, b: AdminPostItem) =>
      new Date(a.publishedAt || a.publishAt || a.createdAt).getTime() -
      new Date(b.publishedAt || b.publishAt || b.createdAt).getTime(),
  },
  { title: '操作', key: 'action', width: 170, fixed: 'right' },
]

// 筛选/翻页变化同步到 URL,便于分享与后退(docs/09 §5.1)
watch([page, pageSize], () => {
  void router.replace({
    query: {
      page: page.value > 1 ? String(page.value) : undefined,
      keyword: keyword.value || undefined,
      status: status.value !== '' ? String(status.value) : undefined,
      categoryId: categoryId.value !== '' ? String(categoryId.value) : undefined,
    },
  })
})

async function loadCategories() {
  try {
    const result = await getCategories({ page: 1, pageSize: 100 })
    categories.value = result.list
  } catch {
    // 分类筛选项为辅助数据：失败时保留空列表，不影响列表主流程
  }
}

function onSearch() {
  keyword.value = keywordInput.value.trim()
  page.value = 1
  void load()
}

function onFilterChange() {
  page.value = 1
  void load()
}

async function togglePublish(record: AdminPostItem) {
  const next: PostStatus = record.status === 1 ? 2 : 1
  await updatePostStatus(record.id, next)
  message.success(next === 1 ? '已发布' : '已下架')
  await load()
}

async function onDelete(record: AdminPostItem) {
  await deletePost(record.id)
  message.success('删除成功')
  await reloadAfterDelete()
}

/* ---------------- 批量操作(循环单条接口,零后端变更 docs/09 §6.2) ---------------- */

const selectedRowKeys = ref<number[]>([])
const batchRunning = ref(false)

const rowSelection = computed(() => ({
  selectedRowKeys: selectedRowKeys.value,
  onChange: (keys: (number | string)[]) => {
    selectedRowKeys.value = keys.map(Number)
  },
}))

async function batchSetStatus(next: PostStatus) {
  batchRunning.value = true
  try {
    const results = await Promise.allSettled(
      selectedRowKeys.value.map((id) => updatePostStatus(id, next)),
    )
    const failed = results.filter((r) => r.status === 'rejected').length
    if (failed > 0) {
      message.warning(`${next === 1 ? '发布' : '下架'}完成，${failed} 篇失败`)
    } else {
      message.success(`已${next === 1 ? '发布' : '下架'} ${selectedRowKeys.value.length} 篇`)
    }
    selectedRowKeys.value = []
    await load()
  } finally {
    batchRunning.value = false
  }
}

function batchDelete() {
  const count = selectedRowKeys.value.length
  modal.confirm({
    title: '批量删除',
    content: `将删除选中的 ${count} 篇文章，其标签关联与评论将一并删除，删除后不可恢复，确定吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      batchRunning.value = true
      try {
        const results = await Promise.allSettled(
          selectedRowKeys.value.map((id) => deletePost(id)),
        )
        const failed = results.filter((r) => r.status === 'rejected').length
        if (failed > 0) message.warning(`删除完成，${failed} 篇失败`)
        else message.success(`已删除 ${count} 篇`)
        selectedRowKeys.value = []
        await load()
      } finally {
        batchRunning.value = false
      }
    },
  })
}

/** 行内「更多」:分析/前台查看直达,删除走 Modal 二次确认(docs/09 §6.1) */
function onRowAction(action: string, record: AdminPostItem) {
  if (action === 'analytics') openAnalytics(record)
  else if (action === 'view') void router.push(`/post/${record.slug}`)
  else if (action === 'delete') {
    modal.confirm({
      title: '删除文章',
      content: '删除后不可恢复，该文章的标签关联与评论将一并删除，确定删除吗？',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => onDelete(record),
    })
  }
}

/** a-menu click：直接双参数调用（模板 @click 内联柯里化表达式会被 Vue 丢弃内层函数，菜单曾整体失效） */
const onMenuClick = (record: AdminPostItem, info: { key: string | number }) =>
  onRowAction(String(info.key), record)

function goCreate() {
  void router.push('/posts/edit')
}

function goEdit(record: AdminPostItem) {
  void router.push(`/posts/edit/${record.id}`)
}

/* ---------------- 单篇文章分析抽屉（契约 #68） ---------------- */

const ANALYTICS_RANGE_OPTIONS: { label: string; value: PostAnalyticsRange }[] = [
  { label: '近 7 天', value: '7d' },
  { label: '近 30 天', value: '30d' },
  { label: '近 90 天', value: '90d' },
]

const analyticsOpen = ref(false)
const analyticsLoading = ref(false)
const analyticsError = ref(false)
const postAnalytics = ref<PostAnalyticsData | null>(null)
const analyticsRange = ref<PostAnalyticsRange>('7d')
const analyticsPost = ref<{ id: number; title: string } | null>(null)

/** 来源/设备中文映射，与访问分析页一致 */
const SOURCE_MAP: Record<string, string> = {
  direct: '直接访问',
  search: '搜索引擎',
  github: 'GitHub',
  social: '社交平台',
  other: '其他',
}

const DEVICE_MAP: Record<string, string> = {
  desktop: '桌面',
  mobile: '移动',
  tablet: '平板',
}

const analyticsTrend = computed(() => postAnalytics.value?.trend ?? [])

const analyticsSourceItems = computed<AnalyticsDistItem[]>(() =>
  (postAnalytics.value?.sources ?? []).map((item) => ({
    name: SOURCE_MAP[item.source] ?? item.source,
    pv: item.pv,
  })),
)

const analyticsDeviceItems = computed<AnalyticsDistItem[]>(() =>
  (postAnalytics.value?.devices ?? []).map((item) => ({
    name: DEVICE_MAP[item.device] ?? item.device,
    pv: item.pv,
  })),
)

async function loadPostAnalytics() {
  if (!analyticsPost.value) return
  analyticsLoading.value = true
  analyticsError.value = false
  try {
    postAnalytics.value = await getPostAnalytics(analyticsPost.value.id, analyticsRange.value)
  } catch {
    // 拦截器已 toast 具体原因
    analyticsError.value = true
  } finally {
    analyticsLoading.value = false
  }
}

function openAnalytics(record: AdminPostItem) {
  analyticsPost.value = { id: record.id, title: record.title }
  analyticsRange.value = '7d'
  postAnalytics.value = null
  analyticsOpen.value = true
  void loadPostAnalytics()
}

function onAnalyticsRangeChange() {
  void loadPostAnalytics()
}

onMounted(() => {
  void load()
  void loadCategories()
})
</script>

<template>
  <div class="page">
    <PageHeader title="文章管理" description="管理博客文章：草稿、发布、隐藏与置顶">
      <template #actions>
        <a-button type="primary" @click="goCreate">
          <template #icon><PlusOutlined /></template>
          新建文章
        </a-button>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <div class="table-toolbar">
        <div class="table-toolbar__filters">
          <a-input
            v-model:value="keywordInput"
            placeholder="搜索标题 / 摘要 / 内容"
            style="width: 220px"
            allow-clear
            @press-enter="onSearch"
          />
          <a-select
            v-model:value="status"
            placeholder="状态"
            style="width: 120px"
            allow-clear
            :options="[
              { label: '草稿', value: 0 },
              { label: '已发布', value: 1 },
              { label: '隐藏', value: 2 },
              { label: '定时发布', value: 3 },
            ]"
            @change="onFilterChange"
          />
          <a-select
            v-model:value="categoryId"
            placeholder="分类"
            style="width: 140px"
            allow-clear
            :options="categories.map((item) => ({ label: item.name, value: item.id }))"
            @change="onFilterChange"
          />
          <a-button @click="onSearch">
            <template #icon><SearchOutlined /></template>
            搜索
          </a-button>
        </div>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </div>

      <LoadError v-if="error" @retry="load" />

      <a-table
        v-else
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1100 }"
        :row-selection="rowSelection"
        row-key="id"
        @change="onTableChange"
      >
        <template #emptyText>
          <TableEmpty text="暂无文章，点击右上角「新建文章」开始写作" />
        </template>
        <template #headerCell="{ column }">
          <template v-if="column.key === 'publishedAt'">
            <a-tooltip title="服务端分页下仅对当前页排序">
              <span>发布时间</span>
            </a-tooltip>
          </template>
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'title'">
            <a class="post-title" @click="goEdit(record)">{{ record.title }}</a>
            <div class="cell-secondary">/ {{ record.slug }}</div>
          </template>

          <template v-else-if="column.key === 'category'">
            {{ record.category ? record.category.name : '未分类' }}
          </template>

          <template v-else-if="column.key === 'tags'">
            <a-space :size="4" wrap>
              <a-tag v-for="tag in record.tags.slice(0, 3)" :key="tag.id">{{ tag.name }}</a-tag>
              <a-tag v-if="record.tags.length > 3">+{{ record.tags.length - 3 }}</a-tag>
              <span v-if="record.tags.length === 0" class="cell-secondary">-</span>
            </a-space>
          </template>

          <template v-else-if="column.key === 'status'">
            <a-tag :color="POST_STATUS_MAP[record.status as PostStatus].color">
              {{ POST_STATUS_MAP[record.status as PostStatus].text }}
            </a-tag>
            <a-tag v-if="record.isTop" color="orange">置顶</a-tag>
          </template>

          <template v-else-if="column.key === 'stats'">
            {{ record.viewCount }} / {{ record.commentCount }}
          </template>

          <template v-else-if="column.key === 'publishedAt'">
            <div v-if="record.status === 3" class="cell-scheduled">
              <span class="cell-scheduled__badge">计划</span>
              <span class="tabular-nums">{{ formatTime(record.publishedAt || record.publishAt) }}</span>
            </div>
            <template v-else>{{ formatTime(record.publishedAt || record.createdAt) }}</template>
          </template>

          <template v-else-if="column.key === 'action'">
            <!-- 操作 >3 收敛:编辑/发布切换平铺,删除进「更多」(docs/09 §6.1) -->
            <a-space :size="0">
              <a-button type="link" size="small" @click="goEdit(record)">编辑</a-button>
              <a-button type="link" size="small" @click="togglePublish(record)">
                {{ record.status === 1 ? '下架' : '发布' }}
              </a-button>
              <a-dropdown :trigger="['click']">
                <a-button type="link" size="small" aria-label="更多操作">
                  <template #icon><MoreOutlined /></template>
                </a-button>
                <template #overlay>
                  <a-menu @click="onMenuClick(record, $event)">
                    <a-menu-item key="analytics">分析</a-menu-item>
                    <a-menu-item key="view" :disabled="record.status !== 1">前台查看</a-menu-item>
                    <a-menu-divider />
                    <a-menu-item key="delete" danger>删除</a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </a-space>
          </template>
        </template>
      </a-table>

      <!-- 批量条：挂在列表尾部（v2 设计稿位置） -->
      <BatchBar
        v-if="selectedRowKeys.length > 0"
        :count="selectedRowKeys.length"
        unit="篇"
        @clear="selectedRowKeys = []"
      >
        <a-button size="small" type="primary" :loading="batchRunning" @click="batchSetStatus(1)">
          批量发布
        </a-button>
        <a-button size="small" :loading="batchRunning" @click="batchSetStatus(2)">批量下架</a-button>
        <a-button size="small" danger :loading="batchRunning" @click="batchDelete">批量删除</a-button>
      </BatchBar>
    </a-card>

    <!-- 单篇文章分析抽屉 -->
    <a-drawer
      v-model:open="analyticsOpen"
      title="文章分析"
      width="min(760px, 100vw)"
      destroy-on-close
    >
      <template #extra>
        <a-segmented
          v-model:value="analyticsRange"
          :options="ANALYTICS_RANGE_OPTIONS"
          :disabled="analyticsLoading"
          @change="onAnalyticsRangeChange"
        />
      </template>

      <div class="post-analytics__subject">
        <span class="post-analytics__subject-label">当前文章</span>
        <span class="post-analytics__subject-title" :title="analyticsPost?.title">
          {{ analyticsPost?.title || '-' }}
        </span>
      </div>

      <!-- 加载骨架 -->
      <div v-if="analyticsLoading && !postAnalytics" class="post-analytics__skeleton">
        <a-skeleton active :title="false" :paragraph="{ rows: 2 }" />
        <a-skeleton active :title="false" :paragraph="{ rows: 6 }" />
      </div>

      <!-- 错误态 -->
      <a-empty v-else-if="analyticsError" description="分析数据加载失败">
        <a-button type="primary" @click="loadPostAnalytics">重试</a-button>
      </a-empty>

      <template v-else-if="postAnalytics">
        <!-- PV / UV / 点赞 / 评论 数字块 -->
        <a-spin :spinning="analyticsLoading">
          <a-row :gutter="[12, 12]">
            <a-col :xs="12" :sm="6">
              <StatCard title="浏览 PV" :value="postAnalytics.totals.pv" :icon="EyeOutlined" />
            </a-col>
            <a-col :xs="12" :sm="6">
              <StatCard title="访客 UV" :value="postAnalytics.totals.uv" :icon="TeamOutlined" />
            </a-col>
            <a-col :xs="12" :sm="6">
              <StatCard title="点赞" :value="postAnalytics.totals.likeCount" :icon="LikeOutlined" />
            </a-col>
            <a-col :xs="12" :sm="6">
              <StatCard title="评论" :value="postAnalytics.totals.commentCount" :icon="CommentOutlined" />
            </a-col>
          </a-row>

          <a-card title="访问趋势" class="post-analytics__section" :body-style="{ padding: '8px 12px 0' }">
            <PvTrendChart v-if="analyticsTrend.length > 0" :data="analyticsTrend" :height="260" />
            <a-empty v-else description="暂无趋势数据" class="post-analytics__empty" />
          </a-card>

          <a-row :gutter="[12, 12]" class="post-analytics__section">
            <a-col :xs="24" :sm="12">
              <a-card title="来源分布" class="post-analytics__dist">
                <DistributionBars v-if="analyticsSourceItems.length > 0" :items="analyticsSourceItems" />
                <a-empty v-else description="暂无数据" class="post-analytics__empty" />
              </a-card>
            </a-col>
            <a-col :xs="24" :sm="12">
              <a-card title="设备分布" class="post-analytics__dist">
                <DistributionBars v-if="analyticsDeviceItems.length > 0" :items="analyticsDeviceItems" />
                <a-empty v-else description="暂无数据" class="post-analytics__empty" />
              </a-card>
            </a-col>
          </a-row>
        </a-spin>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.post-title {
  color: var(--admin-text);
}

.post-title:hover {
  color: var(--admin-brand);
}

/* 定时发布：计划时间标注 */
.cell-scheduled {
  display: flex;
  align-items: center;
  gap: 6px;
}

.cell-scheduled__badge {
  flex: none;
  padding: 0 6px;
  font-size: 12px;
  line-height: 20px;
  color: var(--admin-brand);
  background: var(--admin-brand-bg);
  border-radius: var(--admin-radius-sm);
}

/* 文章分析抽屉 */
.post-analytics__subject {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  min-width: 0;
}

.post-analytics__subject-label {
  flex: none;
  padding: 0 6px;
  font-size: 12px;
  line-height: 20px;
  color: var(--admin-brand);
  background: var(--admin-brand-bg);
  border-radius: var(--admin-radius-sm);
}

.post-analytics__subject-title {
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-analytics__skeleton {
  padding: 4px 0;
}

.post-analytics__section {
  margin-top: 16px;
}

.post-analytics__dist {
  height: 100%;
}

.post-analytics__empty {
  padding: 16px 0;
}
</style>
