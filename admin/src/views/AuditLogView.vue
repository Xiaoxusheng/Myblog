<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import { SearchOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { getAuditLogs } from '@/api/audit'
import { formatTime } from '@/utils/format'
import type { AuditLogItem } from '@/types/api'

const loading = ref(false)
const list = ref<AuditLogItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const actionFilter = ref('')

const columns: TableColumnsType = [
  { title: '时间', key: 'createdAt', width: 160 },
  { title: '动作', key: 'action', width: 150 },
  { title: '对象', key: 'resource', width: 160, ellipsis: true },
  { title: '说明', dataIndex: 'description', key: 'description', ellipsis: true },
]

const pagination = computed<TablePaginationConfig>(() => ({
  current: page.value,
  pageSize: pageSize.value,
  total: total.value,
  showSizeChanger: true,
  pageSizeOptions: ['20', '50'],
  showTotal: (t: number) => `共 ${t} 条`,
}))

async function load() {
  loading.value = true
  try {
    const result = await getAuditLogs({
      action: actionFilter.value.trim() || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    list.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  void load()
}

function onTableChange(p: TablePaginationConfig) {
  page.value = p.current || 1
  pageSize.value = p.pageSize || 20
  void load()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="操作日志" description="记录后台关键操作（IP 已哈希化，不记录敏感内容）" />

    <a-card :bordered="false">
      <div class="audit-toolbar">
        <a-input
          v-model:value="actionFilter"
          placeholder="按动作前缀过滤，如 post. / comment."
          style="width: 260px"
          allow-clear
          @press-enter="onSearch"
        />
        <a-button @click="onSearch">
          <template #icon><SearchOutlined /></template>
          筛选
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
          <template v-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-tag>{{ record.action }}</a-tag>
          </template>
          <template v-else-if="column.key === 'resource'">
            <span v-if="record.resourceType || record.resourceId">
              {{ record.resourceType }}{{ record.resourceId ? `#${record.resourceId}` : '' }}
            </span>
            <span v-else>-</span>
          </template>
        </template>
        <template #emptyText>
          <a-empty description="暂无操作记录" />
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<style scoped>
.audit-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}
</style>
