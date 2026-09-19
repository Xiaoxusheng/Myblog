<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance, TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { createLink, deleteLink, getLinks, updateLink } from '@/api/links'
import { formatTime } from '@/utils/format'
import type { Link } from '@/types/api'

const loading = ref(false)
const list = ref<Link[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const columns: TableColumnsType = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 160 },
  { title: '地址', dataIndex: 'url', key: 'url', width: 240, ellipsis: true },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '可见', key: 'visible', width: 90 },
  { title: '排序', dataIndex: 'sort', key: 'sort', width: 80 },
  { title: '创建时间', key: 'createdAt', width: 150 },
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
    const result = await getLinks({ page: page.value, pageSize: pageSize.value })
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

// 行内可见性开关
const togglingId = ref<number | null>(null)

async function toggleVisible(record: Link, checked: boolean | string | number) {
  togglingId.value = record.id
  try {
    await updateLink(record.id, {
      name: record.name,
      url: record.url,
      logo: record.logo,
      description: record.description,
      visible: Boolean(checked),
      sort: record.sort,
    })
    record.visible = Boolean(checked)
    message.success(record.visible ? '已显示' : '已隐藏')
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
  name: '',
  url: '',
  logo: '',
  description: '',
  visible: true,
  sort: 0,
})

const modalRules = {
  name: [{ required: true, message: '请输入友链名称' }],
  url: [
    { required: true, message: '请输入网站地址' },
    {
      pattern: /^https?:\/\/\S+$/,
      message: '地址需以 http:// 或 https:// 开头',
    },
  ],
}

function openCreate() {
  editingId.value = null
  modalForm.name = ''
  modalForm.url = ''
  modalForm.logo = ''
  modalForm.description = ''
  modalForm.visible = true
  modalForm.sort = 0
  modalOpen.value = true
}

function openEdit(record: Link) {
  editingId.value = record.id
  modalForm.name = record.name
  modalForm.url = record.url
  modalForm.logo = record.logo
  modalForm.description = record.description
  modalForm.visible = record.visible
  modalForm.sort = record.sort
  modalOpen.value = true
}

async function submitModal() {
  await formRef.value?.validate()
  const payload = {
    name: modalForm.name.trim(),
    url: modalForm.url.trim(),
    logo: modalForm.logo.trim(),
    description: modalForm.description.trim(),
    visible: modalForm.visible,
    sort: modalForm.sort,
  }
  modalSaving.value = true
  try {
    if (editingId.value) {
      await updateLink(editingId.value, payload)
      message.success('保存成功')
    } else {
      await createLink(payload)
      message.success('创建成功')
    }
    modalOpen.value = false
    await load()
  } finally {
    modalSaving.value = false
  }
}

async function onDelete(record: Link) {
  await deleteLink(record.id)
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
    <PageHeader title="友链管理" description="管理友情链接的展示与排序">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新建友链
        </a-button>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1100 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a-space :size="8">
              <a-avatar v-if="record.logo" :size="24" :src="record.logo" shape="square" />
              <span>{{ record.name }}</span>
            </a-space>
          </template>

          <template v-else-if="column.key === 'url'">
            <a :href="record.url" target="_blank" rel="noopener noreferrer">{{ record.url }}</a>
          </template>

          <template v-else-if="column.key === 'visible'">
            <a-switch
              size="small"
              :checked="record.visible"
              :loading="togglingId === record.id"
              @change="(checked: boolean | string | number) => toggleVisible(record, checked)"
            />
          </template>

          <template v-else-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
              <a-popconfirm title="删除后不可恢复，确定删除该友链吗？" ok-text="删除" cancel-text="取消" @confirm="onDelete(record)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="modalOpen"
      :title="editingId ? '编辑友链' : '新建友链'"
      :confirm-loading="modalSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitModal"
    >
      <a-form ref="formRef" :model="modalForm" :rules="modalRules" layout="vertical">
        <a-form-item label="名称" name="name">
          <a-input v-model:value="modalForm.name" placeholder="对方站点名称" :maxlength="50" />
        </a-form-item>
        <a-form-item label="地址" name="url">
          <a-input v-model:value="modalForm.url" placeholder="https://example.com" />
        </a-form-item>
        <a-form-item label="Logo 地址" name="logo">
          <a-input v-model:value="modalForm.logo" placeholder="选填，图片地址" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="modalForm.description" placeholder="选填" :rows="2" :maxlength="200" />
        </a-form-item>
        <a-form-item label="是否显示" name="visible">
          <a-switch v-model:checked="modalForm.visible" />
        </a-form-item>
        <a-form-item label="排序" name="sort">
          <a-input-number v-model:value="modalForm.sort" :min="0" :max="9999" style="width: 160px" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
