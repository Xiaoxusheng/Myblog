<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { TableColumnsType } from 'ant-design-vue'
import { CloudDownloadOutlined, DatabaseOutlined, FileZipOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import { useFeedback } from '@/composables/useFeedback'
import { createBackup, deleteBackup, downloadBackup, getBackups } from '@/api/backups'
import { formatTime } from '@/utils/format'
import type { BackupItem } from '@/types/api'

const { message } = useFeedback()

const loading = ref(false)
const error = ref('')
const creating = ref<'database' | 'full' | null>(null)
const list = ref<BackupItem[]>([])

const columns: TableColumnsType = [
  { title: '文件名', dataIndex: 'name', key: 'name', ellipsis: true },
  { title: '类型', key: 'type', width: 110 },
  { title: '大小', key: 'size', width: 110 },
  {
    title: '时间',
    key: 'createdAt',
    width: 160,
    sorter: (a: BackupItem, b: BackupItem) =>
      new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
    defaultSortOrder: 'descend' as const,
  },
  { title: '操作', key: 'action', width: 150, fixed: 'right' },
]

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const totalSize = computed(() => list.value.reduce((sum, item) => sum + item.size, 0))

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await getBackups()
    list.value = data.list
  } catch {
    error.value = '备份列表加载失败，请检查网络后重试'
  } finally {
    loading.value = false
  }
}

async function onCreate(type: 'database' | 'full') {
  creating.value = type
  try {
    await createBackup(type)
    message.success(type === 'database' ? '数据库备份完成' : '全站备份完成')
    await load()
  } finally {
    creating.value = null
  }
}

async function onDownload(record: BackupItem) {
  try {
    await downloadBackup(record.name)
  } catch (err) {
    message.error(err instanceof Error ? err.message : '下载失败')
  }
}

async function onDelete(record: BackupItem) {
  await deleteBackup(record.name)
  message.success('备份已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="备份" description="备份数据库或全站（数据库 + 上传文件打包）；下载后按部署文档可手动恢复" />

    <a-card :bordered="false" class="backup-actions">
      <a-space wrap>
        <a-button type="primary" :loading="creating === 'database'" @click="onCreate('database')">
          <template #icon><DatabaseOutlined /></template>
          备份数据库
        </a-button>
        <a-button :loading="creating === 'full'" @click="onCreate('full')">
          <template #icon><FileZipOutlined /></template>
          备份全站（zip）
        </a-button>
      </a-space>
      <p class="backup-hint">
        「备份数据库」仅支持 SQLite 部署（MySQL 模式请使用「导入导出」页的全站导出）；
        服务运行期间不覆盖数据库文件，恢复请下载备份后按部署文档手动操作。
      </p>
    </a-card>

    <a-card :bordered="false">
      <template #title>备份文件<span v-if="list.length" class="backup-total">（{{ list.length }} 个，共 {{ humanSize(totalSize) }}）</span></template>
      <LoadError v-if="error" :message="error" @retry="load" />

      <a-table
        v-else
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="false"
        :scroll="{ x: 720 }"
        row-key="name"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="record.type === 'database' ? 'blue' : 'purple'">
              {{ record.type === 'database' ? '数据库' : '全站 zip' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'size'">
            {{ humanSize(record.size) }}
          </template>
          <template v-else-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="onDownload(record)">
                <template #icon><CloudDownloadOutlined /></template>
                下载
              </a-button>
              <a-popconfirm
                title="删除后不可恢复，确定删除该备份吗？"
                ok-text="删除"
                cancel-text="取消"
                @confirm="onDelete(record)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
        <template #emptyText>
          <TableEmpty text="暂无备份，点击上方按钮立即创建" />
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<style scoped>
.backup-actions {
  margin-bottom: 16px;
}

.backup-hint {
  margin: 12px 0 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

.backup-total {
  font-size: 12px;
  font-weight: 400;
  color: var(--admin-muted);
  margin-left: 8px;
}
</style>
