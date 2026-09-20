<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { TableColumnsType } from 'ant-design-vue'
import {
  AppstoreOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import BatchBar from '@/components/BatchBar.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { batchPages, deletePage, getPages, updatePageStatus } from '@/api/pages'
import { PAGE_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { PageBatchAction, PageListItem, PageListMeta, PageStatus } from '@/types/api'

const route = useRoute()
const router = useRouter()
const { message, modal } = useFeedback()

/* ---------------- 筛选 ---------------- */

const keywordInput = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const keyword = ref(keywordInput.value)
const status = ref<PageStatus | ''>(
  route.query.status !== undefined && route.query.status !== ''
    ? (Number(route.query.status) as PageStatus)
    : '',
)

/** 概览数据来自列表接口的 meta，不额外发请求（性能要求：不每行单独请求） */
const meta = ref<PageListMeta>({
  totalCount: 0,
  publishedCount: 0,
  draftCount: 0,
  scheduledCount: 0,
})

const {
  loading,
  error,
  list,
  total,
  page,
  pageSize,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<PageListItem>(
  async (params) => {
    const result = await getPages({
      keyword: keyword.value || undefined,
      status: status.value === '' ? undefined : status.value,
      page: params.page,
      pageSize: params.pageSize,
    })
    meta.value = result.meta
    return result
  },
  { initialPage: Number(route.query.page) || 1 },
)

/** 概览文案：4 个页面 · 3 个已发布 · 1 个草稿 · 0 个待发布（轻量一行，不是巨大统计卡） */
const metaSummary = computed(() => {
  const parts = [`${meta.value.totalCount} 个页面`, `${meta.value.publishedCount} 个已发布`]
  if (meta.value.draftCount > 0) parts.push(`${meta.value.draftCount} 个草稿`)
  if (meta.value.scheduledCount > 0) parts.push(`${meta.value.scheduledCount} 个待发布`)
  return parts.join(' · ')
})

/** 是否为移动端（≤768px）：移动端不渲染表格，改卡片列表 */
const isMobile = ref(false)
let mql: MediaQueryList | null = null

function syncMobile(event: MediaQueryList | MediaQueryListEvent) {
  isMobile.value = event.matches
}

watch([page, pageSize], () => {
  void router.replace({
    query: {
      page: page.value > 1 ? String(page.value) : undefined,
      keyword: keyword.value || undefined,
      status: status.value !== '' ? String(status.value) : undefined,
    },
  })
})

function onSearch() {
  keyword.value = keywordInput.value.trim()
  page.value = 1
  void load()
}

function onFilterChange() {
  page.value = 1
  void load()
}

const statusOptions = [
  { label: '草稿', value: 0 },
  { label: '已发布', value: 1 },
  { label: '隐藏', value: 2 },
  { label: '定时发布', value: 3 },
]

/* ---------------- 列定义 ---------------- */

const columns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', key: 'title', width: 260, ellipsis: true },
  { title: 'Slug', dataIndex: 'slug', key: 'slug', width: 180, ellipsis: true },
  { title: '状态', key: 'status', width: 130 },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160 },
  { title: '更新时间', dataIndex: 'updatedAt', key: 'updatedAt', width: 160 },
  { title: '操作', key: 'action', width: 160, fixed: 'right' },
]

/* ---------------- 行操作 ---------------- */

function goCreate() {
  void router.push('/pages/edit')
}

function goEdit(record: PageListItem) {
  void router.push(`/pages/edit/${record.id}`)
}

/**
 * 预览：已发布 → 直接打开前台 /page/<slug>；
 * 未发布 → 跳编辑器并带 preview=1（编辑器用管理端预览接口渲染，不走公开接口）
 */
function onPreview(record: PageListItem) {
  if (record.status === 1) {
    window.open(`/page/${record.slug}`, '_blank', 'noopener')
    return
  }
  void router.push(`/pages/edit/${record.id}?preview=1`)
}

async function togglePublish(record: PageListItem) {
  try {
    const next: PageStatus = record.status === 1 ? 2 : 1
    await updatePageStatus(record.id, { status: next })
    message.success(next === 1 ? '已发布' : '已隐藏')
    await load()
  } catch {
    // 接口层已 toast 具体原因；此处兜底避免未处理 rejection
  }
}

async function onCopy(record: PageListItem) {
  try {
    const result = await import('@/api/pages').then((m) => m.copyPage(record.id))
    message.success(`已复制为「${result.page.title}」，状态为草稿`)
    await load()
  } catch {
    // 接口层已 toast 具体原因；此处兜底避免未处理 rejection
  }
}
function onRowAction(action: string, record: PageListItem) {
  if (action === 'publish') {
    void togglePublish(record)
  } else if (action === 'copy') {
    void onCopy(record)
  } else if (action === 'preview') {
    onPreview(record)
  } else if (action === 'delete') {
    modal.confirm({
      title: '删除页面',
      content: `确定删除"${record.title}"？页面删除后无法在正常页面列表中恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await deletePage(record.id)
        message.success('删除成功')
        await reloadAfterDelete()
      },
    })
  }
}

/** a-menu click：直接双参数调用（模板 @click 内联柯里化表达式会被 Vue 丢弃内层函数，菜单曾整体失效） */
const onMenuClick = (record: PageListItem, info: { key: string | number }) =>
  onRowAction(String(info.key), record)

/* ---------------- 批量操作（走后端 #104，不再循环单条请求） ---------------- */

const selectedRowKeys = ref<number[]>([])
const batchRunning = ref(false)

const rowSelection = computed(() => ({
  selectedRowKeys: selectedRowKeys.value,
  onChange: (keys: (number | string)[]) => {
    selectedRowKeys.value = keys.map(Number)
  },
}))

async function runBatch(action: PageBatchAction) {
  batchRunning.value = true
  try {
    const result = await batchPages({ action, ids: selectedRowKeys.value })
    const label = action === 'publish' ? '发布' : action === 'hide' ? '隐藏' : '删除'
    message.success(`已${label} ${result.updated} 个页面`)
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
    content: `将删除选中的 ${count} 个页面及其版本历史，删除后无法恢复，确定吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: () => runBatch('delete'),
  })
}

/* ---------------- 移动端适配 ---------------- */

onMounted(() => {
  mql = window.matchMedia('(max-width: 768px)')
  syncMobile(mql)
  mql.addEventListener('change', syncMobile)
  void load()
})
</script>

<template>
  <div class="page">
    <PageHeader title="页面管理" description="管理关于、友链说明等独立内容页面">
      <template #actions>
        <a-button type="primary" @click="goCreate">
          <template #icon><PlusOutlined /></template>
          新建页面
        </a-button>
      </template>
    </PageHeader>

    <!-- 轻量概览：单行文字，不是四个巨大统计卡 -->
    <div v-if="!loading && meta.totalCount > 0" class="page-meta">
      <AppstoreOutlined class="page-meta__icon" />
      <span>{{ metaSummary }}</span>
    </div>

    <a-card :bordered="false">
      <div class="table-toolbar">
        <div class="table-toolbar__filters">
          <a-input
            v-model:value="keywordInput"
            placeholder="搜索标题 / Slug"
            style="width: 220px"
            allow-clear
            @press-enter="onSearch"
          />
          <a-select
            v-model:value="status"
            placeholder="状态"
            style="width: 120px"
            allow-clear
            :options="statusOptions"
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

      <BatchBar
        v-if="selectedRowKeys.length > 0"
        :count="selectedRowKeys.length"
        @clear="selectedRowKeys = []"
      >
        <a-button size="small" :loading="batchRunning" @click="runBatch('publish')">批量发布</a-button>
        <a-button size="small" :loading="batchRunning" @click="runBatch('hide')">批量隐藏</a-button>
        <a-button size="small" danger :loading="batchRunning" @click="batchDelete">批量删除</a-button>
      </BatchBar>

      <LoadError v-if="error" @retry="load" />

      <!-- 移动端：卡片列表（不渲染完整表格） -->
      <a-spin v-else-if="isMobile" :spinning="loading">
        <a-empty v-if="list.length === 0" description="还没有自定义页面">
          <template #description>
            <div class="page-empty__title">还没有自定义页面</div>
            <div class="page-empty__desc">创建关于、友链说明等独立内容页面。</div>
          </template>
          <a-button type="primary" @click="goCreate">+ 新建页面</a-button>
        </a-empty>
        <a-list v-else :data-source="list" class="page-mobile-list">
          <template #renderItem="{ item }">
            <a-list-item class="page-mobile-item">
              <div class="page-mobile-item__main">
                <a class="page-mobile-item__title" @click="goEdit(item)">{{ item.title }}</a>
                <div class="cell-secondary">/ {{ item.slug }}</div>
                <div class="page-mobile-item__foot">
                  <a-tag :color="PAGE_STATUS_MAP[item.status as PageStatus].color">
                    {{ PAGE_STATUS_MAP[item.status as PageStatus].text }}
                  </a-tag>
                  <span class="cell-secondary">更新于 {{ formatTime(item.updatedAt) }}</span>
                </div>
              </div>
              <div class="page-mobile-item__actions">
                <a-button type="link" size="small" @click="goEdit(item)">编辑</a-button>
                <a-dropdown :trigger="['click']" placement="bottomRight">
                  <a-button type="link" size="small" aria-label="更多操作">
                    <template #icon><MoreOutlined /></template>
                  </a-button>
                  <template #overlay>
                    <a-menu @click="onMenuClick(item, $event)">
                      <a-menu-item key="publish">
                        {{ item.status === 1 ? '转为隐藏' : '快速发布' }}
                      </a-menu-item>
                      <a-menu-item key="preview">
                        {{ item.status === 1 ? '预览' : '后台预览' }}
                      </a-menu-item>
                      <a-menu-item key="copy">复制页面</a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="delete" danger>删除</a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
              </div>
            </a-list-item>
          </template>
        </a-list>
        <!-- 分页：超过一页才显示 -->
        <a-pagination
          v-if="total > pageSize"
          :current="page"
          :total="total"
          :page-size="pageSize"
          size="small"
          class="page-mobile-pager"
          @change="(p: number) => { page = p; load() }"
        />
        <!-- 单页时只显示总数 -->
        <div v-else-if="total > 0" class="page-total-hint">共 {{ total }} 个页面</div>
      </a-spin>

      <a-table
        v-else
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1080 }"
        :row-selection="rowSelection"
        row-key="id"
        @change="onTableChange"
      >
        <template #emptyText>
          <TableEmpty text="还没有自定义页面，创建关于、友链说明等独立内容页面。" />
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'title'">
            <a class="page-title" @click="goEdit(record)">{{ record.title }}</a>
          </template>

          <template v-else-if="column.key === 'slug'">
            <span class="cell-mono cell-secondary">/{{ record.slug }}</span>
          </template>

          <template v-else-if="column.key === 'status'">
            <a-tag :color="PAGE_STATUS_MAP[record.status as PageStatus].color">
              {{ PAGE_STATUS_MAP[record.status as PageStatus].text }}
            </a-tag>
            <div v-if="record.status === 3 && record.publishAt" class="cell-schedule">
              {{ formatTime(record.publishAt) }} 发布
            </div>
          </template>

          <template v-else-if="column.key === 'createdAt'">
            <span class="cell-secondary">{{ formatTime(record.createdAt) }}</span>
          </template>

          <template v-else-if="column.key === 'updatedAt'">
            <span class="cell-secondary">{{ formatTime(record.updatedAt) }}</span>
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="goEdit(record)">编辑</a-button>
              <a-dropdown :trigger="['click']">
                <a-button type="link" size="small" aria-label="更多操作">
                  <template #icon><MoreOutlined /></template>
                </a-button>
                <template #overlay>
                  <a-menu @click="onMenuClick(record, $event)">
                    <a-menu-item key="publish">
                      {{ record.status === 1 ? '转为隐藏' : '快速发布' }}
                    </a-menu-item>
                    <a-menu-item key="preview">
                      {{ record.status === 1 ? '预览' : '后台预览' }}
                    </a-menu-item>
                    <a-menu-item key="copy">复制页面</a-menu-item>
                    <a-menu-divider />
                    <a-menu-item key="delete" danger>删除</a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<style scoped>
/* 轻量概览行 */
.page-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--admin-muted);
}

.page-meta__icon {
  color: var(--admin-muted);
}

/* 标题：hover 变品牌色 */
.page-title {
  color: var(--admin-text);
  font-weight: 500;
}

.page-title:hover {
  color: var(--admin-brand);
}

.cell-mono {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
}

/* 定时发布：状态下方展示计划发布时间 */
.cell-schedule {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

/* 移动端卡片列表 */
.page-mobile-list :deep(.ant-list-item) {
  padding: 12px 0;
}

.page-mobile-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.page-mobile-item__main {
  flex: 1;
  min-width: 0;
}

.page-mobile-item__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--admin-text);
}

.page-mobile-item__foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  font-size: 12px;
}

.page-mobile-item__actions {
  flex: none;
  display: flex;
  align-items: center;
}

.page-mobile-pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.page-total-hint {
  margin-top: 12px;
  font-size: 13px;
  color: var(--admin-muted);
  text-align: right;
}

/* 空状态 */
.page-empty__title {
  font-size: 15px;
  color: var(--admin-text);
}

.page-empty__desc {
  margin-top: 4px;
  font-size: 13px;
  color: var(--admin-muted);
}
</style>
