<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import type { TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import PageHeader from '@/components/PageHeader.vue'
import { batchComments, deleteComment, getComments, replyComment, updateCommentStatus } from '@/api/comments'
import { COMMENT_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { CommentAdmin, CommentStatus } from '@/types/api'

type TabKey = 'all' | '0' | '1' | '2' | '3' | '4'

const route = useRoute()
const router = useRouter()

// 支持仪表盘快捷入口 /comments?status=0 直达对应 Tab
const initialTab = (() => {
  const raw = route.query.status
  return typeof raw === 'string' && ['0', '1', '2', '3', '4'].includes(raw) ? (raw as TabKey) : ('all' as TabKey)
})()

const activeTab = ref<TabKey>(initialTab)
const loading = ref(false)
const list = ref<CommentAdmin[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

/** 仅看某篇文章的评论（drawer 入口） */
const postFilter = ref<number | undefined>(undefined)
const postFilterTitle = ref('')

const tabs: { key: TabKey; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: '0', label: '待审核' },
  { key: '1', label: '已通过' },
  { key: '2', label: '已拒绝' },
  { key: '3', label: '垃圾' },
  { key: '4', label: '回收站' },
]

const columns: TableColumnsType = [
  { title: '文章', dataIndex: 'postTitle', key: 'postTitle', width: 160, ellipsis: true },
  { title: '昵称', dataIndex: 'nickname', key: 'nickname', width: 130 },
  { title: '邮箱', dataIndex: 'email', key: 'email', width: 170, ellipsis: true },
  { title: '内容', dataIndex: 'content', key: 'content', ellipsis: true },
  { title: 'IP', dataIndex: 'ip', key: 'ip', width: 120, ellipsis: true },
  { title: '状态', key: 'status', width: 90 },
  { title: '时间', key: 'createdAt', width: 140 },
  { title: '操作', key: 'action', width: 250, fixed: 'right' },
]

const pagination = computed<TablePaginationConfig>(() => ({
  current: page.value,
  pageSize: pageSize.value,
  total: total.value,
  showSizeChanger: true,
  pageSizeOptions: ['10', '20', '50'],
  showTotal: (t: number) => `共 ${t} 条`,
}))

async function load() {
  loading.value = true
  try {
    const result = await getComments({
      status: activeTab.value === 'all' ? undefined : (Number(activeTab.value) as CommentStatus),
      postId: postFilter.value,
      page: page.value,
      pageSize: pageSize.value,
    })
    list.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function syncQuery() {
  void router.replace({
    query: {
      status: activeTab.value !== 'all' ? activeTab.value : undefined,
      postId: postFilter.value ? String(postFilter.value) : undefined,
    },
  })
}

function onTabChange() {
  page.value = 1
  syncQuery()
  void load()
}

function onTableChange(paginationConfig: TablePaginationConfig) {
  page.value = paginationConfig.current || 1
  pageSize.value = paginationConfig.pageSize || 10
  void load()
}

async function setStatus(record: CommentAdmin, status: CommentStatus) {
  await updateCommentStatus(record.id, status)
  message.success(COMMENT_STATUS_MAP[status].text)
  record.status = status
  if (drawerRecord.value?.id === record.id) {
    drawerRecord.value.status = status
  }
  await load()
}

// ---------- 批量操作（契约 #72） ----------
const selectedRowKeys = ref<number[]>([])
const batchRunning = ref(false)

const rowSelection = computed({
  get: () => ({ selectedRowKeys: selectedRowKeys.value, onChange: (keys: (number | string)[]) => {
    selectedRowKeys.value = keys.map(Number)
  } }),
  set: () => {},
})

async function runBatch(action: 'approve' | 'reject' | 'spam' | 'delete') {
  if (selectedRowKeys.value.length === 0) return
  const doRun = async () => {
    batchRunning.value = true
    try {
      await batchComments(action, selectedRowKeys.value)
      message.success(`已${batchActionText[action]} ${selectedRowKeys.value.length} 条`)
      selectedRowKeys.value = []
      await load()
    } finally {
      batchRunning.value = false
    }
  }
  if (action === 'delete') {
    Modal.confirm({
      title: '批量删除',
      content: `将删除选中的 ${selectedRowKeys.value.length} 条评论及其回复，删除后不可恢复，确定吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: doRun,
    })
    return
  }
  await doRun()
}

const batchActionText: Record<string, string> = {
  approve: '通过', reject: '拒绝', spam: '标记垃圾', delete: '删除',
}

// 评论详情 Drawer（行点击 / 详情）
const drawerOpen = ref(false)
const drawerRecord = ref<CommentAdmin | null>(null)

function openDrawer(record: CommentAdmin) {
  drawerRecord.value = record
  drawerOpen.value = true
}

function onRow(record: CommentAdmin) {
  return { onClick: () => openDrawer(record), style: { cursor: 'pointer' } }
}

function initial(nickname: string): string {
  return nickname.trim().charAt(0).toUpperCase()
}

function viewPostComments(record: CommentAdmin) {
  postFilter.value = record.postId
  postFilterTitle.value = record.postTitle
  page.value = 1
  drawerOpen.value = false
  syncQuery()
  void load()
}

function clearPostFilter() {
  postFilter.value = undefined
  postFilterTitle.value = ''
  page.value = 1
  syncQuery()
  void load()
}

// 回复
const replyOpen = ref(false)
const replyTarget = ref<CommentAdmin | null>(null)
const replyContent = ref('')
const replySubmitting = ref(false)

function openReply(record: CommentAdmin) {
  replyTarget.value = record
  replyContent.value = ''
  replyOpen.value = true
}

async function submitReply() {
  const content = replyContent.value.trim()
  if (!content) {
    message.error('请输入回复内容')
    return
  }
  if (!replyTarget.value) return
  replySubmitting.value = true
  try {
    await replyComment(replyTarget.value.id, content)
    message.success('回复成功，回复将直接展示')
    replyOpen.value = false
    await load()
  } finally {
    replySubmitting.value = false
  }
}

async function onDelete(record: CommentAdmin) {
  await deleteComment(record.id)
  message.success('删除成功')
  if (drawerRecord.value?.id === record.id) {
    drawerOpen.value = false
  }
  if (list.value.length === 1 && page.value > 1) {
    page.value -= 1
  }
  await load()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="评论管理" description="审核游客评论，可回复、拒绝或删除" />

    <a-card :bordered="false">
      <a-tabs v-model:active-key="activeTab" @change="onTabChange">
        <a-tab-pane v-for="tab in tabs" :key="tab.key" :tab="tab.label" />
      </a-tabs>

      <div v-if="postFilter" class="comment-post-filter">
        <a-tag closable color="blue" @close.prevent="clearPostFilter">
          仅看《{{ postFilterTitle }}》的评论
        </a-tag>
      </div>

      <div v-if="selectedRowKeys.length > 0" class="comment-batch-bar">
        <a-space wrap>
          <span class="comment-batch-bar__count">已选 {{ selectedRowKeys.length }} 条</span>
          <a-button size="small" :loading="batchRunning" @click="runBatch('approve')">批量通过</a-button>
          <a-button size="small" :loading="batchRunning" @click="runBatch('reject')">批量拒绝</a-button>
          <a-button size="small" :loading="batchRunning" @click="runBatch('spam')">批量标垃圾</a-button>
          <a-button size="small" danger :loading="batchRunning" @click="runBatch('delete')">批量删除</a-button>
          <a-button size="small" type="text" @click="selectedRowKeys = []">取消选择</a-button>
        </a-space>
      </div>

      <a-table
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1280 }"
        :custom-row="onRow"
        :row-selection="rowSelection"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'nickname'">
            <span>{{ record.nickname }}</span>
            <a-space :size="4" style="margin-left: 4px">
              <a-tag v-if="record.isAdmin" color="blue">管理员</a-tag>
              <a-tag v-else-if="record.parentId > 0">回复</a-tag>
            </a-space>
          </template>

          <template v-else-if="column.key === 'content'">
            <a-tooltip :title="record.content">
              <span>{{ record.content }}</span>
            </a-tooltip>
          </template>

          <template v-else-if="column.key === 'status'">
            <a-tag :color="COMMENT_STATUS_MAP[record.status as CommentStatus].color">
              {{ COMMENT_STATUS_MAP[record.status as CommentStatus].text }}
            </a-tag>
          </template>

          <template v-else-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <!-- 阻止冒泡：动作点击不应触发行打开详情 -->
            <a-space :size="0" wrap @click.stop>
              <a-button type="link" size="small" @click="openDrawer(record)">详情</a-button>
              <a-button v-if="record.status !== 1" type="link" size="small" @click="setStatus(record, 1)">
                通过
              </a-button>
              <a-button
                v-if="record.status !== 2 && !record.isAdmin"
                type="link"
                size="small"
                @click="setStatus(record, 2)"
              >
                拒绝
              </a-button>
              <a-button
                v-if="record.status !== 3 && !record.isAdmin"
                type="link"
                size="small"
                @click="setStatus(record, 3)"
              >
                标垃圾
              </a-button>
              <a-button
                v-if="record.status === 3 || record.status === 4"
                type="link"
                size="small"
                @click="setStatus(record, 0)"
              >
                恢复待审
              </a-button>
              <a-button type="link" size="small" @click="openReply(record)">回复</a-button>
              <a-popconfirm
                title="删除后不可恢复，该评论的回复将一并删除，确定删除吗？"
                ok-text="删除"
                cancel-text="取消"
                @confirm="onDelete(record)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 评论详情 -->
    <a-drawer
      v-model:open="drawerOpen"
      title="评论详情"
      width="min(480px, 92vw)"
      class="comment-drawer"
    >
      <div v-if="drawerRecord" class="comment-detail">
        <div class="comment-detail__head">
          <a-avatar :size="36" class="comment-detail__avatar">{{ initial(drawerRecord.nickname) }}</a-avatar>
          <div class="comment-detail__who">
            <div class="comment-detail__name">
              {{ drawerRecord.nickname }}
              <a-tag v-if="drawerRecord.isAdmin" color="blue">管理员</a-tag>
              <a-tag v-else-if="drawerRecord.parentId > 0">回复</a-tag>
            </div>
            <div class="comment-detail__time">{{ formatTime(drawerRecord.createdAt) }}</div>
          </div>
          <a-tag :color="COMMENT_STATUS_MAP[drawerRecord.status].color">
            {{ COMMENT_STATUS_MAP[drawerRecord.status].text }}
          </a-tag>
        </div>

        <div class="comment-detail__content">{{ drawerRecord.content }}</div>

        <a-descriptions :column="1" size="small" class="comment-detail__meta">
          <a-descriptions-item label="文章">《{{ drawerRecord.postTitle }}》</a-descriptions-item>
          <a-descriptions-item label="邮箱">{{ drawerRecord.email || '-' }}</a-descriptions-item>
          <a-descriptions-item label="网站">
            <a v-if="drawerRecord.website" :href="drawerRecord.website" target="_blank" rel="noopener noreferrer">
              {{ drawerRecord.website }}
            </a>
            <span v-else>-</span>
          </a-descriptions-item>
          <a-descriptions-item label="IP">{{ drawerRecord.ip || '-' }}</a-descriptions-item>
        </a-descriptions>

        <a-space wrap class="comment-detail__actions">
          <a-button
            v-if="drawerRecord.status !== 1"
            type="primary"
            size="small"
            @click="setStatus(drawerRecord, 1)"
          >
            通过
          </a-button>
          <a-button
            v-if="drawerRecord.status !== 2 && !drawerRecord.isAdmin"
            size="small"
            @click="setStatus(drawerRecord, 2)"
          >
            拒绝
          </a-button>
          <a-button size="small" @click="openReply(drawerRecord)">回复</a-button>
          <a-popconfirm
            title="删除后不可恢复，该评论的回复将一并删除，确定删除吗？"
            ok-text="删除"
            cancel-text="取消"
            @confirm="onDelete(drawerRecord)"
          >
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
        </a-space>

        <a-button type="link" size="small" class="comment-detail__same-post" @click="viewPostComments(drawerRecord)">
          查看该文章的其他评论
        </a-button>
      </div>
    </a-drawer>

    <a-modal
      v-model:open="replyOpen"
      title="回复评论"
      :confirm-loading="replySubmitting"
      ok-text="回复"
      cancel-text="取消"
      @ok="submitReply"
    >
      <a-alert
        v-if="replyTarget"
        type="info"
        show-icon
        style="margin-bottom: 16px"
        :message="`${replyTarget.nickname}：${replyTarget.content}`"
      />
      <a-textarea
        v-model:value="replyContent"
        placeholder="以管理员身份回复，回复将直接通过审核"
        :rows="4"
        :maxlength="1000"
        show-count
      />
    </a-modal>
  </div>
</template>

<style scoped>
.comment-post-filter {
  margin-bottom: 12px;
}

.comment-detail__head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.comment-detail__avatar {
  flex: none;
  background: var(--admin-brand-bg);
  color: var(--admin-brand);
  font-weight: 600;
}

.comment-detail__who {
  flex: 1;
  min-width: 0;
}

.comment-detail__name {
  font-weight: 600;
  color: var(--admin-text);
}

.comment-detail__time {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

.comment-detail__content {
  margin-top: 16px;
  padding: 12px;
  font-size: 14px;
  line-height: 22px;
  color: var(--admin-text);
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--admin-surface-2);
  border-radius: var(--admin-radius-sm);
}

.comment-detail__meta {
  margin-top: 16px;
}

.comment-detail__actions {
  margin-top: 16px;
}

.comment-detail__same-post {
  margin-top: 16px;
  padding: 0;
}
.comment-batch-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: var(--admin-surface-2);
  border-radius: var(--admin-radius-sm);
}

.comment-batch-bar__count {
  font-size: 13px;
  color: var(--admin-text);
}
</style>
