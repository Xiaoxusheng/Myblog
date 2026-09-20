<script setup lang="ts">
import { nextTick, onMounted, reactive, ref } from 'vue'
import type { Dayjs } from 'dayjs'
import dayjs from 'dayjs'
import type { FormInstance, RadioChangeEvent, TableColumnsType } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { adminTheme } from '@/theme'
import {
  createChangelog,
  deleteChangelog,
  getChangelogs,
  updateChangelog,
} from '@/api/changelogs'
import { formatTime } from '@/utils/format'
import type { ChangelogItem } from '@/types/api'

const { message, modal } = useFeedback()

/** 状态筛选：undefined=全部；切换后回第一页重查 */
const statusFilter = ref<number | undefined>(undefined)

const {
  loading,
  error,
  list,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<ChangelogItem>(({ page, pageSize }) =>
  getChangelogs({ page, pageSize, status: statusFilter.value }),
)

const columns: TableColumnsType = [
  { title: '版本', dataIndex: 'version', key: 'version', width: 120 },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '发布日期', key: 'releasedAt', width: 120 },
  { title: '状态', key: 'status', width: 90 },
  { title: '排序', dataIndex: 'sort', key: 'sort', width: 80 },
  { title: '更新时间', key: 'updatedAt', width: 150 },
  { title: '操作', key: 'action', width: 170, fixed: 'right' },
]

function onFilterChange(value: number | undefined) {
  statusFilter.value = value
  void load()
}

function onFilterRadio(e: RadioChangeEvent) {
  onFilterChange(e.target.value)
}

// 新建 / 编辑弹窗
const modalOpen = ref(false)
const modalSaving = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const modalForm = reactive({
  version: '',
  title: '',
  content: '',
  status: 0 as number,
  sort: 0,
})

const releasedAtValue = ref<Dayjs | null>(null)

const modalRules = {
  version: [
    { required: true, message: '请输入版本号' },
    {
      pattern: /^v?\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/,
      message: '格式如 1.2.0 / v1.2.0-beta.1',
    },
  ],
}

function openCreate() {
  editingId.value = null
  modalForm.version = ''
  modalForm.title = ''
  modalForm.content = ''
  modalForm.status = 0
  modalForm.sort = 0
  releasedAtValue.value = dayjs()
  modalOpen.value = true
  clearFormValidate()
}

function openEdit(record: ChangelogItem) {
  editingId.value = record.id
  modalForm.version = record.version
  modalForm.title = record.title
  modalForm.content = record.content
  modalForm.status = record.status
  modalForm.sort = record.sort
  releasedAtValue.value = dayjs(record.releasedAt)
  modalOpen.value = true
  clearFormValidate()
}

/** 弹窗重开时清除上一次的校验错误残留 */
function clearFormValidate() {
  void nextTick(() => formRef.value?.clearValidate())
}

async function submitModal() {
  try {
    await formRef.value?.validate()
  } catch {
    return // 表单项已显示行内错误，静默终止
  }
  if (!releasedAtValue.value) {
    message.warning('请选择发布日期')
    return
  }
  const payload = {
    version: modalForm.version.trim(),
    title: modalForm.title.trim(),
    content: modalForm.content,
    releasedAt: releasedAtValue.value.toISOString(),
    status: modalForm.status,
    sort: modalForm.sort,
  }
  modalSaving.value = true
  try {
    if (editingId.value) {
      await updateChangelog(editingId.value, payload)
      message.success('保存成功')
    } else {
      await createChangelog(payload)
      message.success('创建成功')
    }
    modalOpen.value = false
    await load()
  } catch {
    // 业务错误（如版本号已存在）已由 http 拦截器统一提示，弹窗保持打开
  } finally {
    modalSaving.value = false
  }
}

/** 删除保护：已发布记录先二次确认，再带 force 删除 */
async function onDelete(record: ChangelogItem) {
  if (record.status === 1) {
    modal.confirm({
      title: '删除已发布的版本记录？',
      content: `v${record.version} 已在前台展示，删除后访客将无法看到该版本的更新说明，且不可恢复。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await deleteChangelog(record.id, true)
        message.success('删除成功')
        await reloadAfterDelete()
      },
    })
    return
  }
  await deleteChangelog(record.id)
  message.success('删除成功')
  await reloadAfterDelete()
}

// 内容预览
const previewOpen = ref(false)
const previewVersion = ref('')
const previewContent = ref('')

function openPreview(record: ChangelogItem) {
  previewVersion.value = record.version
  previewContent.value = record.content
  previewOpen.value = true
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="版本发布记录" description="维护前台「更新日志」页面的版本记录，支持草稿与发布状态">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新建版本
        </a-button>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <template #title>
        <a-radio-group :value="statusFilter" @change="onFilterRadio">
          <a-radio-button :value="undefined">全部</a-radio-button>
          <a-radio-button :value="0">草稿</a-radio-button>
          <a-radio-button :value="1">已发布</a-radio-button>
        </a-radio-group>
      </template>

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
          <TableEmpty text="暂无版本记录，点击右上角「新建版本」添加" />
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'version'">
            <span class="version-tag">v{{ record.version }}</span>
          </template>

          <template v-else-if="column.key === 'releasedAt'">
            {{ formatTime(record.releasedAt, false) }}
          </template>

          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'green' : 'default'">
              {{ record.status === 1 ? '已发布' : '草稿' }}
            </a-tag>
          </template>

          <template v-else-if="column.key === 'updatedAt'">
            {{ formatTime(record.updatedAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
              <a-button type="link" size="small" @click="openPreview(record)">预览</a-button>
              <a-button type="link" size="small" danger @click="onDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="modalOpen"
      :title="editingId ? '编辑版本记录' : '新建版本记录'"
      :confirm-loading="modalSaving"
      ok-text="保存"
      cancel-text="取消"
      width="640px"
      @ok="submitModal"
    >
      <a-form ref="formRef" :model="modalForm" :rules="modalRules" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="版本号" name="version">
              <a-input v-model:value="modalForm.version" placeholder="如 1.2.0" :maxlength="32" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="发布日期" required>
              <!-- 日期存于独立 releasedAtValue ref，不走 form model 校验，提交前手动判空 -->
              <a-date-picker
                :value="releasedAtValue"
                style="width: 100%"
                placeholder="发布日期"
                @update:value="(v: Dayjs | null) => (releasedAtValue = v)"
              />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="标题" name="title">
          <a-input v-model:value="modalForm.title" placeholder="选填，如：编辑器与安全升级" :maxlength="100" show-count />
        </a-form-item>
        <a-form-item label="更新说明（Markdown）" name="content">
          <a-textarea v-model:value="modalForm.content" placeholder="支持 Markdown，如：&#10;## 新增&#10;- 时间线页面" :rows="8" :maxlength="20000" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="状态" name="status">
              <a-radio-group v-model:value="modalForm.status">
                <a-radio :value="0">草稿</a-radio>
                <a-radio :value="1">发布</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="排序" name="sort">
              <a-input-number v-model:value="modalForm.sort" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <p class="form-hint">排序升序在前，默认 0 即按发布日期倒序（新版本在前）</p>
      </a-form>
    </a-modal>

    <a-modal v-model:open="previewOpen" :title="`预览：v${previewVersion}`" :footer="null" width="720px">
      <MdPreview
        :model-value="previewContent"
        :theme="adminTheme === 'dark' ? 'dark' : 'light'"
        language="zh-CN"
        no-katex
        no-mermaid
      />
    </a-modal>
  </div>
</template>

<style scoped>
.version-tag {
  font-family: var(--admin-font-mono, monospace);
  font-weight: 600;
}

.form-hint {
  margin: -8px 0 0;
  font-size: 12px;
  color: var(--admin-muted);
}
</style>
