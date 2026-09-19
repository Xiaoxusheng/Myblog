<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import PageHeader from '@/components/PageHeader.vue'
import { deleteComment, getComments, replyComment, updateCommentStatus } from '@/api/comments'
import { COMMENT_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { CommentAdmin, CommentStatus } from '@/types/api'

type TabKey = 'all' | '0' | '1' | '2'

const activeTab = ref<TabKey>('all')
const loading = ref(false)
const list = ref<CommentAdmin[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const tabs: { key: TabKey; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: '0', label: '待审核' },
  { key: '1', label: '已通过' },
  { key: '2', label: '已拒绝' },
]

const columns: TableColumnsType = [
  { title: '文章', dataIndex: 'postTitle', key: 'postTitle', width: 160, ellipsis: true },
  { title: '昵称', dataIndex: 'nickname', key: 'nickname', width: 130 },
  { title: '邮箱', dataIndex: 'email', key: 'email', width: 170, ellipsis: true },
  { title: '内容', dataIndex: 'content', key: 'content', ellipsis: true },
  { title: 'IP', dataIndex: 'ip', key: 'ip', width: 120, ellipsis: true },
  { title: '状态', key: 'status', width: 90 },
  { title: '时间', key: 'createdAt', width: 140 },
  { title: '操作', key: 'action', width: 220, fixed: 'right' },
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
      page: page.value,
      pageSize: pageSize.value,
    })
    list.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function onTabChange() {
  page.value = 1
  void load()
}

function onTableChange(paginationConfig: TablePaginationConfig) {
  page.value = paginationConfig.current || 1
  pageSize.value = paginationConfig.pageSize || 10
  void load()
}

async function setStatus(record: CommentAdmin, status: 1 | 2) {
  await updateCommentStatus(record.id, status)
  message.success(status === 1 ? '已通过' : '已拒绝')
  await load()
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

      <a-table
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1200 }"
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
            <a-space :size="0" wrap>
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
              <a-button type="link" size="small" @click="openReply(record)">回复</a-button>
              <a-popconfirm title="该评论的回复将一并删除，确定删除吗？" ok-text="删除" cancel-text="取消" @confirm="onDelete(record)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

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
