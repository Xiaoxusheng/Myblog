<script setup lang="ts">
/**
 * 草稿工作区（模块五）。
 *
 * 定位：把「未定稿的文章」集中到一处，覆盖三类真实工作状态——
 *   1. 草稿（status=0）：还没写完的
 *   2. 定时发布（status=3）：写完了但排在未来的
 *   3. 带本地未同步改动的：浏览器里有草稿但没提交到服务器
 * 排序固定为「最近编辑」，并把本地草稿的存在显式标出来，避免用户忘记自己改过什么。
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { TableColumnsType } from 'ant-design-vue'
import {
  ClockCircleOutlined,
  CloudUploadOutlined,
  FileDoneOutlined,
  FileTextOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import StatCard from '@/components/StatCard.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { deletePost, getPosts } from '@/api/posts'
import { getCategories, getTags } from '@/api/taxonomy'
import { getStats } from '@/api/stats'
import { POST_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import { countWords, estimateReadingMinutes } from '@/utils/postCheck'
import type { AdminPostItem, Category, PostStatus, Stats, Tag } from '@/types/api'

const route = useRoute()
const router = useRouter()
const { message, modal } = useFeedback()

const DRAFT_PREFIX = 'blog_admin_post_draft'

// ---------------------------------------------------------------------------
// 筛选条件（与文章管理一致：keyword 回车生效，其余即时生效）
// ---------------------------------------------------------------------------
/** 状态 Tab：草稿 / 定时发布；'' 表示两者都要 */
type DraftTab = '' | '0' | '3'

const activeTab = ref<DraftTab>(
  route.query.status === '3' ? '3' : route.query.status === '0' ? '0' : '',
)
const keywordInput = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const keyword = ref(keywordInput.value)
const categoryId = ref<number | ''>(
  route.query.categoryId !== undefined && route.query.categoryId !== ''
    ? Number(route.query.categoryId)
    : '',
)
const tagId = ref<number | ''>(
  route.query.tagId !== undefined && route.query.tagId !== '' ? Number(route.query.tagId) : '',
)

const categories = ref<Category[]>([])
const tags = ref<Tag[]>([])
const stats = ref<Stats | null>(null)

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
      // 不选 Tab 时同时取草稿与定时发布（后端 status 只接受单值）
      status: activeTab.value === '' ? undefined : (Number(activeTab.value) as PostStatus),
      categoryId: categoryId.value === '' ? undefined : categoryId.value,
      tagId: tagId.value === '' ? undefined : tagId.value,
      sort: 'updatedAt',
      page,
      pageSize,
    }),
  { initialPage: Number(route.query.page) || 1 },
)

/** 不选 Tab 时要排除已发布/隐藏，这里在客户端二次过滤兜底 */
const visibleList = computed(() =>
  activeTab.value === ''
    ? list.value.filter((item) => item.status === 0 || item.status === 3)
    : list.value,
)

// ---------------------------------------------------------------------------
// 本地草稿标记：扫 localStorage 里 blog_admin_post_draft_* 存在哪些文章 id
// 在加载后与窗口重新聚焦时重扫（另一标签页保存后本地草稿会被清掉）
// ---------------------------------------------------------------------------
const localDraftIds = ref<Set<number>>(new Set())

function scanLocalDrafts() {
  const ids = new Set<number>()
  try {
    for (let i = 0; i < localStorage.length; i += 1) {
      const key = localStorage.key(i)
      if (!key || !key.startsWith(`${DRAFT_PREFIX}_`)) continue
      const suffix = key.slice(DRAFT_PREFIX.length + 1)
      // _new 是无 id 的新建草稿，不属于任何已存文章
      if (suffix === 'new') continue
      const id = Number(suffix)
      if (Number.isFinite(id) && id > 0) ids.add(id)
    }
  } catch {
    // localStorage 不可用（隐私模式）时视为无本地草稿
  }
  localDraftIds.value = ids
}

// ---------------------------------------------------------------------------
// 表格列
// ---------------------------------------------------------------------------
const columns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', key: 'title', width: 260, ellipsis: true },
  { title: '分类', dataIndex: ['category', 'name'], key: 'category', width: 110 },
  { title: '标签', key: 'tags', width: 150 },
  { title: '字数', key: 'wordCount', width: 110 },
  { title: '状态', key: 'status', width: 120 },
  {
    title: '最后保存',
    key: 'updatedAt',
    width: 170,
    // 服务端已按 updatedAt 倒序返回；列内排序仅作当前页交互（接口无正序参数）
    sorter: (a: AdminPostItem, b: AdminPostItem) =>
      new Date(a.updatedAt || a.createdAt).getTime() - new Date(b.updatedAt || b.createdAt).getTime(),
  },
  { title: '操作', key: 'action', width: 150, fixed: 'right' },
]

watch([page, pageSize], () => {
  void router.replace({
    query: {
      page: page.value > 1 ? String(page.value) : undefined,
      status: activeTab.value || undefined,
      keyword: keyword.value || undefined,
      categoryId: categoryId.value !== '' ? String(categoryId.value) : undefined,
      tagId: tagId.value !== '' ? String(tagId.value) : undefined,
    },
  })
})

// ---------------------------------------------------------------------------
// 数据加载
// ---------------------------------------------------------------------------
async function loadOptions() {
  const [catResult, tagResult, statsResult] = await Promise.allSettled([
    getCategories({ page: 1, pageSize: 100 }),
    getTags({ page: 1, pageSize: 100 }),
    getStats(),
  ])
  if (catResult.status === 'fulfilled') categories.value = catResult.value.list
  if (tagResult.status === 'fulfilled') tags.value = tagResult.value.list
  if (statsResult.status === 'fulfilled') stats.value = statsResult.value
}

async function refresh() {
  try {
    scanLocalDrafts()
    await load()
  } catch {
    // scanLocalDrafts 解析本地草稿可能抛错；load 本身已有兜底。
    // 这里统一捕获，避免 onMounted 的 fire-and-forget 调用产生未处理 rejection
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

/** Tab 切换：清空 status 筛选语义由 activeTab 决定 */
function onTabChange() {
  page.value = 1
  void load()
}

function goCreate() {
  void router.push('/posts/edit')
}

function goEdit(record: AdminPostItem) {
  void router.push(`/posts/edit/${record.id}`)
}

function onRowAction(action: string, record: AdminPostItem) {
  if (action === 'edit') goEdit(record)
  else if (action === 'view') void router.push(`/post/${record.slug}`)
  else if (action === 'discardLocal') {
    modal.confirm({
      title: '丢弃本地草稿',
      content: '将删除这篇文章在本浏览器中未同步的改动，服务器上的内容不受影响。确定吗？',
      okText: '丢弃',
      okType: 'danger',
      cancelText: '取消',
      onOk: () => {
        try {
          localStorage.removeItem(`${DRAFT_PREFIX}_${record.id}`)
        } catch {
          // 忽略（localStorage 不可用）
        }
        scanLocalDrafts()
        message.success('已丢弃本地草稿')
      },
    })
  } else if (action === 'delete') {
    modal.confirm({
      title: '删除文章',
      content: '删除后不可恢复，标签关联与评论将一并删除，确定删除吗？',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await deletePost(record.id)
        message.success('删除成功')
        await reloadAfterDelete()
      },
    })
  }
}

const onMenuClick = (record: AdminPostItem) => (info: { key: string | number }) =>
  onRowAction(String(info.key), record)

/** 窗口重新聚焦时重扫本地草稿（另一标签页保存后会清理） */
function onFocus() {
  scanLocalDrafts()
}

onMounted(() => {
  void refresh()
  void loadOptions()
  window.addEventListener('focus', onFocus)
})
onUnmounted(() => {
  window.removeEventListener('focus', onFocus)
})
</script>

<template>
  <div class="page">
    <PageHeader title="草稿工作区" description="集中处理未定稿的文章：草稿与定时发布，按最近编辑排序">
      <template #actions>
        <a-button type="primary" @click="goCreate">
          <template #icon><PlusOutlined /></template>
          新建文章
        </a-button>
      </template>
    </PageHeader>

    <a-row :gutter="[12, 12]" class="drafts__stats">
      <a-col :xs="12" :sm="6">
        <StatCard title="草稿" :value="stats?.draftCount ?? 0" :icon="FileTextOutlined" />
      </a-col>
      <a-col :xs="12" :sm="6">
        <StatCard title="定时发布" :value="stats?.scheduledCount ?? 0" :icon="ClockCircleOutlined" />
      </a-col>
      <a-col :xs="12" :sm="6">
        <StatCard title="已发布" :value="stats?.postCount ?? 0" :icon="FileDoneOutlined" />
      </a-col>
      <a-col :xs="12" :sm="6">
        <StatCard title="本地未同步" :value="localDraftIds.size" :icon="CloudUploadOutlined" />
      </a-col>
    </a-row>

    <a-card :bordered="false">
      <a-tabs v-model:activeKey="activeTab" class="drafts__tabs" @change="onTabChange">
        <a-tab-pane key="" tab="全部未定稿" />
        <a-tab-pane key="0" tab="草稿" />
        <a-tab-pane key="3" tab="定时发布" />
      </a-tabs>

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
            v-model:value="categoryId"
            placeholder="分类"
            style="width: 140px"
            allow-clear
            :options="categories.map((item) => ({ label: item.name, value: item.id }))"
            @change="onFilterChange"
          />
          <a-select
            v-model:value="tagId"
            placeholder="标签"
            style="width: 140px"
            allow-clear
            show-search
            option-filter-prop="label"
            :options="tags.map((item) => ({ label: item.name, value: item.id }))"
            @change="onFilterChange"
          />
          <a-button @click="onSearch">
            <template #icon><SearchOutlined /></template>
            搜索
          </a-button>
        </div>
        <a-button :loading="loading" @click="refresh">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </div>

      <LoadError v-if="error" @retry="load" />

      <a-table
        v-else
        :columns="columns"
        :data-source="visibleList"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1000 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #emptyText>
          <TableEmpty text="没有未定稿的文章，都写完了" />
        </template>
        <template #headerCell="{ column }">
          <template v-if="column.key === 'wordCount'">
            <a-tooltip title="正文字数（已去除 Markdown 标记与代码块）">
              <span>字数</span>
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'updatedAt'">
            <a-tooltip title="服务端分页下仅对当前页排序；列表默认按最近编辑倒序">
              <span>最后保存</span>
            </a-tooltip>
          </template>
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'title'">
            <a class="drafts__title" @click="goEdit(record)">{{ record.title || '（无标题）' }}</a>
            <div class="cell-secondary">/ {{ record.slug }}</div>
            <a-tag v-if="localDraftIds.has(record.id)" color="orange" class="drafts__local-tag">
              本地未同步
            </a-tag>
          </template>

          <template v-else-if="column.key === 'category'">
            {{ record.category ? record.category.name : '未分类' }}
          </template>

          <template v-else-if="column.key === 'tags'">
            <a-space :size="4" wrap>
              <a-tag v-for="tag in record.tags.slice(0, 2)" :key="tag.id">{{ tag.name }}</a-tag>
              <a-tag v-if="record.tags.length > 2">+{{ record.tags.length - 2 }}</a-tag>
              <span v-if="record.tags.length === 0" class="cell-secondary">-</span>
            </a-space>
          </template>

          <template v-else-if="column.key === 'wordCount'">
            <span class="tabular-nums">
              {{ countWords(record.content) }} 字
            </span>
            <div class="cell-secondary">
              约 {{ estimateReadingMinutes(countWords(record.content)) }} 分钟
            </div>
          </template>

          <template v-else-if="column.key === 'status'">
            <a-tag :color="POST_STATUS_MAP[record.status as PostStatus].color">
              {{ POST_STATUS_MAP[record.status as PostStatus].text }}
            </a-tag>
            <div v-if="record.status === 3 && record.publishAt" class="cell-secondary tabular-nums">
              {{ formatTime(record.publishAt) }}
            </div>
          </template>

          <template v-else-if="column.key === 'updatedAt'">
            {{ formatTime(record.updatedAt || record.createdAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="goEdit(record)">继续编辑</a-button>
              <a-dropdown :trigger="['click']">
                <a-button type="link" size="small" aria-label="更多操作">
                  <template #icon><MoreOutlined /></template>
                </a-button>
                <template #overlay>
                  <a-menu @click="onMenuClick(record)">
                    <a-menu-item key="view" :disabled="record.status !== 1">前台查看</a-menu-item>
                    <a-menu-item key="discardLocal" :disabled="!localDraftIds.has(record.id)">
                      丢弃本地草稿
                    </a-menu-item>
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
.drafts__stats {
  margin-bottom: 16px;
}

.drafts__tabs {
  margin-bottom: 4px;
}

.drafts__title {
  color: var(--admin-text);
}

.drafts__title:hover {
  color: var(--admin-brand);
}

.drafts__local-tag {
  margin-top: 4px;
}
</style>
