<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import type { FormInstance, TableColumnsType } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { createCategory, deleteCategory, getCategories, updateCategory } from '@/api/taxonomy'
import type { Category } from '@/types/api'

const { message } = useFeedback()

const {
  loading,
  error,
  list,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<Category>(({ page, pageSize }) =>
  getCategories({ page, pageSize }),
)

const columns: TableColumnsType = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 180 },
  { title: 'Slug', dataIndex: 'slug', key: 'slug', width: 200, ellipsis: true },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '文章数', dataIndex: 'postCount', key: 'postCount', width: 100 },
  { title: '操作', key: 'action', width: 140, fixed: 'right' },
]

// 新建 / 编辑弹窗
const modalOpen = ref(false)
const modalSaving = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const modalForm = reactive({
  name: '',
  slug: '',
  description: '',
})

const modalRules = {
  name: [{ required: true, message: '请输入分类名称' }],
  slug: [
    {
      pattern: /^[a-zA-Z0-9_-]*$/,
      message: '仅支持字母、数字、短横线和下划线',
    },
  ],
}

function openCreate() {
  editingId.value = null
  modalForm.name = ''
  modalForm.slug = ''
  modalForm.description = ''
  modalOpen.value = true
}

function openEdit(record: Category) {
  editingId.value = record.id
  modalForm.name = record.name
  modalForm.slug = record.slug
  modalForm.description = record.description || ''
  modalOpen.value = true
}

async function submitModal() {
  await formRef.value?.validate()
  const payload = {
    name: modalForm.name.trim(),
    slug: modalForm.slug.trim(),
    description: modalForm.description.trim(),
  }
  modalSaving.value = true
  try {
    if (editingId.value) {
      await updateCategory(editingId.value, payload)
      message.success('保存成功')
    } else {
      await createCategory(payload)
      message.success('创建成功')
    }
    modalOpen.value = false
    await load()
  } finally {
    modalSaving.value = false
  }
}

async function onDelete(record: Category) {
  await deleteCategory(record.id)
  message.success('删除成功')
  await reloadAfterDelete()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="分类管理" description="组织文章的分类目录">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新建分类
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
        :scroll="{ x: 900 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #emptyText>
          <TableEmpty text="暂无分类，点击右上角「新建分类」创建" />
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'description'">
            <a-tooltip v-if="record.description" :title="record.description">
              <span>{{ record.description }}</span>
            </a-tooltip>
            <span v-else class="cell-secondary">-</span>
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
              <a-popconfirm
                title="删除后不可恢复，该分类下的文章将变为未分类，确定删除吗？"
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
      :title="editingId ? '编辑分类' : '新建分类'"
      :confirm-loading="modalSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitModal"
    >
      <a-form ref="formRef" :model="modalForm" :rules="modalRules" layout="vertical">
        <a-form-item label="名称" name="name">
          <a-input v-model:value="modalForm.name" placeholder="分类名称，需唯一" :maxlength="50" />
        </a-form-item>
        <a-form-item label="Slug" name="slug">
          <a-input v-model:value="modalForm.slug" placeholder="留空由后端生成" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="modalForm.description" placeholder="选填" :rows="3" :maxlength="200" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
