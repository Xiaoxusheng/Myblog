<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import type { TableColumnsType } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { deletePage, getPages } from '@/api/pages'
import { POST_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { PageItem, PageStatus } from '@/types/api'

const router = useRouter()
const { message } = useFeedback()

const {
  loading,
  error,
  list,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<PageItem>(({ page, pageSize }) =>
  getPages({ page, pageSize }),
)

const columns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', key: 'title', width: 240, ellipsis: true },
  { title: 'Slug', dataIndex: 'slug', key: 'slug', width: 220, ellipsis: true },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', key: 'createdAt', width: 160 },
  { title: '更新时间', key: 'updatedAt', width: 160 },
  { title: '操作', key: 'action', width: 140, fixed: 'right' },
]

async function onDelete(record: PageItem) {
  await deletePage(record.id)
  message.success('删除成功')
  await reloadAfterDelete()
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
      <LoadError v-if="error" @retry="load" />

      <a-table
        v-else
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1000 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #emptyText>
          <TableEmpty text="暂无页面，点击右上角「新建页面」创建" />
        </template>
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
              <a-popconfirm title="删除后不可恢复，确定删除该页面吗？" ok-text="删除" cancel-text="取消" @confirm="onDelete(record)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>
