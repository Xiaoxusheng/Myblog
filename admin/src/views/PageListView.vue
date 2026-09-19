<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { deletePage, getPages } from '@/api/pages'
import { POST_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { PageItem, PageStatus } from '@/types/api'

const router = useRouter()

const loading = ref(false)
const list = ref<PageItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const columns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', key: 'title', width: 240, ellipsis: true },
  { title: 'Slug', dataIndex: 'slug', key: 'slug', width: 220, ellipsis: true },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', key: 'createdAt', width: 160 },
  { title: '更新时间', key: 'updatedAt', width: 160 },
  { title: '操作', key: 'action', width: 140, fixed: 'right' },
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
    const result = await getPages({ page: page.value, pageSize: pageSize.value })
    list.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function onTableChange(paginationConfig: TablePaginationConfig) {
  page.value = paginationConfig.current || 1
  pageSize.value = paginationConfig.pageSize || 10
  void load()
}

async function onDelete(record: PageItem) {
  await deletePage(record.id)
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
    <PageHeader title="页面管理" description="自定义页面，如“关于”“友链说明”">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="router.push('/pages/edit')">
          <template #icon><PlusOutlined /></template>
          新建页面
        </a-button>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1000 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="POST_STATUS_MAP[record.status as PageStatus].color">
              {{ POST_STATUS_MAP[record.status as PageStatus].text }}
            </a-tag>
          </template>

          <template v-else-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>

          <template v-else-if="column.key === 'updatedAt'">
            {{ formatTime(record.updatedAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="router.push(`/pages/edit/${record.id}`)">
                编辑
              </a-button>
              <a-popconfirm title="确定删除该页面吗？" ok-text="删除" cancel-text="取消" @confirm="onDelete(record)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>
