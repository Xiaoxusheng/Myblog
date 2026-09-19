<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { createBlacklist, deleteBlacklist, getBlacklist } from '@/api/moderation'
import { formatTime } from '@/utils/format'
import type { BlacklistItem } from '@/types/api'

const loading = ref(false)
const list = ref<BlacklistItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const typeFilter = ref<'ip' | 'email' | 'keyword' | ''>('')

const columns: TableColumnsType = [
  { title: '类型', key: 'type', width: 110 },
  { title: '内容', dataIndex: 'value', key: 'value', ellipsis: true },
  { title: '加入时间', key: 'createdAt', width: 160 },
  { title: '操作', key: 'action', width: 90, fixed: 'right' },
]

const typeText: Record<BlacklistItem['type'], string> = {
  ip: 'IP',
  email: '邮箱',
  keyword: '关键词',
}
const typeColor: Record<BlacklistItem['type'], string> = {
  ip: 'blue',
  email: 'purple',
  keyword: 'orange',
}

const pagination = computed<TablePaginationConfig>(() => ({
  current: page.value,
  pageSize: pageSize.value,
  total: total.value,
  showTotal: (t: number) => `共 ${t} 条`,
}))

async function load() {
  loading.value = true
  try {
    const result = await getBlacklist({
      type: typeFilter.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    list.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function onTableChange(p: TablePaginationConfig) {
  page.value = p.current || 1
  pageSize.value = p.pageSize || 10
  void load()
}

function onFilterChange() {
  page.value = 1
  void load()
}

// 新建
const modalOpen = ref(false)
const saving = ref(false)
const form = reactive({ type: 'ip' as BlacklistItem['type'], value: '' })

function openCreate() {
  form.type = 'ip'
  form.value = ''
  modalOpen.value = true
}

async function submitCreate() {
  if (!form.value.trim()) {
    message.error('请输入内容')
    return
  }
  saving.value = true
  try {
    await createBlacklist({ type: form.type, value: form.value.trim() })
    message.success('已加入黑名单')
    modalOpen.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onDelete(record: BlacklistItem) {
  await deleteBlacklist(record.id)
  message.success('已移出黑名单')
  if (list.value.length === 1 && page.value > 1) {
    page.value -= 1
  }
  await load()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="评论防护" description="命中黑名单的评论将被直接标记为垃圾，前台不可见" />

    <a-card :bordered="false">
      <div class="blacklist-toolbar">
        <a-radio-group v-model:value="typeFilter" button-style="solid" @change="onFilterChange">
          <a-radio-button value="">全部</a-radio-button>
          <a-radio-button value="ip">IP</a-radio-button>
          <a-radio-button value="email">邮箱</a-radio-button>
          <a-radio-button value="keyword">关键词</a-radio-button>
        </a-radio-group>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          添加黑名单
        </a-button>
      </div>

      <a-table
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 720 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="typeColor[record.type as BlacklistItem['type']]">
              {{ typeText[record.type as BlacklistItem['type']] }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-popconfirm
              title="移出黑名单后该来源可再次评论，确定移出吗？"
              ok-text="移出"
              cancel-text="取消"
              @confirm="onDelete(record)"
            >
              <a-button type="link" size="small" danger>移出</a-button>
            </a-popconfirm>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:open="modalOpen" title="添加黑名单" :confirm-loading="saving" @ok="submitCreate">
      <a-form layout="vertical" style="margin-top: 12px">
        <a-form-item label="类型" required>
          <a-radio-group v-model:value="form.type">
            <a-radio value="ip">IP</a-radio>
            <a-radio value="email">邮箱</a-radio>
            <a-radio value="keyword">关键词</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item
          label="内容"
          required
          :extra="form.type === 'keyword' ? '评论内容包含该关键词（不区分大小写）即判为垃圾' : '完全匹配时生效'"
        >
          <a-input v-model:value="form.value" :maxlength="200" placeholder="最多 200 字" allow-clear />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.blacklist-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}
</style>
