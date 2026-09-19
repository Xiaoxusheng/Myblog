<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import type { FormInstance, TableColumnsType } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { createTag, deleteTag, getTags, updateTag } from '@/api/taxonomy'
import type { Tag } from '@/types/api'

const { message } = useFeedback()

const {
  loading,
  error,
  list,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<Tag>(({ page, pageSize }) =>
  getTags({ page, pageSize }),
)

const columns: TableColumnsType = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 220 },
  { title: 'Slug', dataIndex: 'slug', key: 'slug', width: 240, ellipsis: true },
  { title: '文章数', dataIndex: 'postCount', key: 'postCount', width: 120 },
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
})

const modalRules = {
  name: [{ required: true, message: '请输入标签名称' }],
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
  modalOpen.value = true
}

function openEdit(record: Tag) {
  editingId.value = record.id
  modalForm.name = record.name
  modalForm.slug = record.slug
  modalOpen.value = true
}

async function submitModal() {
  await formRef.value?.validate()
  const payload = {
    name: modalForm.name.trim(),
    slug: modalForm.slug.trim(),
  }
  modalSaving.value = true
  try {
    if (editingId.value) {
      await updateTag(editingId.value, payload)
      message.success('保存成功')
    } else {
      await createTag(payload)
      message.success('创建成功')
    }
    modalOpen.value = false
    await load()
  } finally {
    modalSaving.value = false
  }
}

async function onDelete(record: Tag) {
  await deleteTag(record.id)
  message.success('删除成功')
  await reloadAfterDelete()
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="标签管理" description="文章的标签集合">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新建标签
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
        :scroll="{ x: 800 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #emptyText>
          <TableEmpty text="暂无标签，点击右上角「新建标签」创建" />
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
              <a-popconfirm
                title="删除后不可恢复，将同步解除文章关联，确定删除吗？"
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
      :title="editingId ? '编辑标签' : '新建标签'"
      :confirm-loading="modalSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitModal"
    >
      <a-form ref="formRef" :model="modalForm" :rules="modalRules" layout="vertical">
        <a-form-item label="名称" name="name">
          <a-input v-model:value="modalForm.name" placeholder="标签名称" :maxlength="50" />
        </a-form-item>
        <a-form-item label="Slug" name="slug">
          <a-input v-model:value="modalForm.slug" placeholder="留空由后端生成" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
