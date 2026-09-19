<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import type { FormInstance, TableColumnsType } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { createRedirect, deleteRedirect, getRedirects, updateRedirect } from '@/api/redirects'
import { formatTime } from '@/utils/format'
import type { RedirectItem } from '@/types/api'

const { message } = useFeedback()

const {
  loading,
  error,
  list,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<RedirectItem>(({ page, pageSize }) =>
  getRedirects({ page, pageSize }),
)

const columns: TableColumnsType = [
  { title: '来源路径', dataIndex: 'source', key: 'source', width: 260, ellipsis: true },
  { title: '目标路径', dataIndex: 'target', key: 'target', width: 260, ellipsis: true },
  { title: '类型', key: 'type', width: 110 },
  { title: '状态', key: 'enabled', width: 80 },
  { title: '创建时间', key: 'createdAt', width: 150 },
  { title: '操作', key: 'action', width: 140, fixed: 'right' },
]

// 行内启用开关
const togglingId = ref<number | null>(null)

async function toggleEnabled(record: RedirectItem, checked: boolean | string | number) {
  togglingId.value = record.id
  try {
    await updateRedirect(record.id, {
      source: record.source,
      target: record.target,
      type: record.type,
      enabled: Boolean(checked),
    })
    record.enabled = Boolean(checked)
    message.success(record.enabled ? '已启用' : '已停用')
  } finally {
    togglingId.value = null
  }
}

// 新建 / 编辑弹窗
const modalOpen = ref(false)
const modalSaving = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const modalForm = reactive({
  source: '',
  target: '',
  type: 301 as 301 | 302,
  enabled: true,
})

/** 站内路径：以 / 开头，不含 query/hash（环检测、source≠target 由后端校验） */
const pathPattern = /^\/[^?#]*$/

const modalRules = {
  source: [
    { required: true, message: '请输入来源路径' },
    { pattern: pathPattern, message: '需以 / 开头，且不含 query 或 hash' },
  ],
  target: [
    { required: true, message: '请输入目标路径' },
    { pattern: pathPattern, message: '需以 / 开头，且不含 query 或 hash' },
  ],
}

function openCreate() {
  editingId.value = null
  modalForm.source = ''
  modalForm.target = ''
  modalForm.type = 301
  modalForm.enabled = true
  modalOpen.value = true
}

function openEdit(record: RedirectItem) {
  editingId.value = record.id
  modalForm.source = record.source
  modalForm.target = record.target
  modalForm.type = record.type
  modalForm.enabled = record.enabled
  modalOpen.value = true
}

async function submitModal() {
  await formRef.value?.validate()
  const payload = {
    source: modalForm.source.trim(),
    target: modalForm.target.trim(),
    type: modalForm.type,
    enabled: modalForm.enabled,
  }
  modalSaving.value = true
  try {
    if (editingId.value) {
      await updateRedirect(editingId.value, payload)
      message.success('保存成功')
    } else {
      await createRedirect(payload)
      message.success('创建成功')
    }
    modalOpen.value = false
    await load()
  } finally {
    modalSaving.value = false
  }
}

async function onDelete(record: RedirectItem) {
  await deleteRedirect(record.id)
  message.success('删除成功')
  await reloadAfterDelete()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="重定向" description="旧地址自动跳转到新地址，避免链接失效">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新建重定向
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
          <TableEmpty text="暂无重定向规则，点击右上角「新建重定向」创建" />
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'source'">
            <a-tooltip :title="record.source">
              <code class="path-code">{{ record.source }}</code>
            </a-tooltip>
          </template>

          <template v-else-if="column.key === 'target'">
            <a-tooltip :title="record.target">
              <code class="path-code">{{ record.target }}</code>
            </a-tooltip>
          </template>

          <template v-else-if="column.key === 'type'">
            <a-tag :color="record.type === 301 ? 'success' : 'processing'">
              {{ record.type === 301 ? '301 永久' : '302 临时' }}
            </a-tag>
          </template>

          <template v-else-if="column.key === 'enabled'">
            <a-switch
              size="small"
              :checked="record.enabled"
              :loading="togglingId === record.id"
              @change="(checked: boolean | string | number) => toggleEnabled(record, checked)"
            />
          </template>

          <template v-else-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
              <a-popconfirm
                title="删除后该旧地址将 404，确定删除吗？"
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

    <a-modal
      v-model:open="modalOpen"
      :title="editingId ? '编辑重定向' : '新建重定向'"
      :confirm-loading="modalSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitModal"
    >
      <a-form ref="formRef" :model="modalForm" :rules="modalRules" layout="vertical">
        <a-form-item label="来源路径" name="source" extra="访问旧地址时跳转到目标路径">
          <a-input v-model:value="modalForm.source" placeholder="/post/old-slug" />
        </a-form-item>
        <a-form-item label="目标路径" name="target">
          <a-input v-model:value="modalForm.target" placeholder="/post/new-slug" />
        </a-form-item>
        <a-form-item label="类型" name="type">
          <a-radio-group v-model:value="modalForm.type">
            <a-radio :value="301">301 永久重定向</a-radio>
            <a-radio :value="302">302 临时重定向</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="启用" name="enabled">
          <a-switch v-model:checked="modalForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
/* 路径单元格：等宽 code 风格 */
.path-code {
  padding: 1px 6px;
  border-radius: var(--admin-radius-sm);
  background: var(--admin-surface-2);
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12.5px;
  color: var(--admin-text);
}
</style>
