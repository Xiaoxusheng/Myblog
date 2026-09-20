<script setup lang="ts">
import { nextTick, onMounted, reactive, ref } from 'vue'
import type { Dayjs } from 'dayjs'
import dayjs from 'dayjs'
import type { FormInstance, TableColumnsType } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import TableEmpty from '@/components/TableEmpty.vue'
import MediaSelectModal from '@/components/MediaSelectModal.vue'
import { useTable } from '@/composables/useTable'
import { useFeedback } from '@/composables/useFeedback'
import { adminTheme } from '@/theme'
import { createTimeline, deleteTimeline, getTimeline, updateTimeline } from '@/api/timeline'
import { getPosts } from '@/api/posts'
import { formatTime } from '@/utils/format'
import type { AdminPostItem, TimelineEventAdmin } from '@/types/api'

const { message } = useFeedback()

const {
  loading,
  error,
  list,
  pagination,
  load,
  onTableChange,
  reloadAfterDelete,
} = useTable<TimelineEventAdmin>(({ page, pageSize }) => getTimeline({ page, pageSize }))

const columns: TableColumnsType = [
  { title: '日期', key: 'eventDate', width: 110 },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true, width: 220 },
  { title: '关联文章', key: 'post', ellipsis: true },
  { title: '项目', dataIndex: 'projectName', key: 'projectName', width: 140, ellipsis: true },
  { title: '可见', key: 'visible', width: 90 },
  { title: '排序', dataIndex: 'sort', key: 'sort', width: 80 },
  { title: '创建时间', key: 'createdAt', width: 150 },
  { title: '操作', key: 'action', width: 170, fixed: 'right' },
]

// 行内可见性开关
const togglingId = ref<number | null>(null)

async function toggleVisible(record: TimelineEventAdmin, checked: boolean | string | number) {
  togglingId.value = record.id
  try {
    await updateTimeline(record.id, toPayload(record, Boolean(checked)))
    record.visible = Boolean(checked)
    message.success(record.visible ? '已显示' : '已隐藏')
  } finally {
    togglingId.value = null
  }
}

function toPayload(record: TimelineEventAdmin, visible: boolean) {
  return {
    title: record.title,
    content: record.content,
    eventDate: record.eventDate,
    image: record.image,
    postId: record.postId,
    projectName: record.projectName,
    projectUrl: record.projectUrl,
    visible,
    sort: record.sort,
  }
}

// 关联文章候选项（编辑弹窗打开时拉取，按标题搜索过滤）
const postOptions = ref<Array<{ value: number; label: string }>>([])
const postOptionsLoading = ref(false)

async function loadPostOptions() {
  if (postOptions.value.length > 0) return
  postOptionsLoading.value = true
  try {
    const result = await getPosts({ page: 1, pageSize: 50 })
    postOptions.value = result.list.map((p: AdminPostItem) => ({
      value: p.id,
      label: `${p.title}（${p.status === 1 ? '已发布' : p.status === 3 ? '定时' : p.status === 2 ? '隐藏' : '草稿'}）`,
    }))
  } catch {
    // 文章下拉为辅助数据：失败时保留空列表，不影响时间线主流程
  } finally {
    postOptionsLoading.value = false
  }
}

// 新建 / 编辑弹窗
const modalOpen = ref(false)
const modalSaving = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()
const mediaOpen = ref(false)

const modalForm = reactive({
  title: '',
  content: '',
  image: '',
  postId: undefined as number | undefined,
  projectName: '',
  projectUrl: '',
  visible: true,
  sort: 0,
})

/** 日期单独用 Dayjs ref（与 PostEditView 同约定），提交时转 RFC3339 */
const eventDateValue = ref<Dayjs | null>(null)

const modalRules = {
  title: [{ required: true, message: '请输入节点标题' }],
  projectUrl: [{ pattern: /^https?:\/\/\S+$/, message: '链接需以 http:// 或 https:// 开头' }],
}

function openCreate() {
  editingId.value = null
  modalForm.title = ''
  modalForm.content = ''
  modalForm.image = ''
  modalForm.postId = undefined
  modalForm.projectName = ''
  modalForm.projectUrl = ''
  modalForm.visible = true
  modalForm.sort = 0
  eventDateValue.value = null
  modalOpen.value = true
  clearFormValidate()
  void loadPostOptions()
}

function openEdit(record: TimelineEventAdmin) {
  editingId.value = record.id
  modalForm.title = record.title
  modalForm.content = record.content
  modalForm.image = record.image
  modalForm.postId = record.postId > 0 ? record.postId : undefined
  modalForm.projectName = record.projectName
  modalForm.projectUrl = record.projectUrl
  modalForm.visible = record.visible
  modalForm.sort = record.sort
  eventDateValue.value = dayjs(record.eventDate)
  modalOpen.value = true
  clearFormValidate()
  void loadPostOptions()
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
  if (!eventDateValue.value) {
    message.warning('请选择节点日期')
    return
  }
  const payload = {
    title: modalForm.title.trim(),
    content: modalForm.content,
    eventDate: eventDateValue.value.toISOString(),
    image: modalForm.image.trim(),
    postId: modalForm.postId ?? 0,
    projectName: modalForm.projectName.trim(),
    projectUrl: modalForm.projectUrl.trim(),
    visible: modalForm.visible,
    sort: modalForm.sort,
  }
  modalSaving.value = true
  try {
    if (editingId.value) {
      await updateTimeline(editingId.value, payload)
      message.success('保存成功')
    } else {
      await createTimeline(payload)
      message.success('创建成功')
    }
    modalOpen.value = false
    await load()
  } catch {
    // 业务错误已由 http 拦截器统一提示，弹窗保持打开
  } finally {
    modalSaving.value = false
  }
}

async function onDelete(record: TimelineEventAdmin) {
  await deleteTimeline(record.id)
  message.success('删除成功')
  await reloadAfterDelete()
}

// 内容预览
const previewOpen = ref(false)
const previewTitle = ref('')
const previewContent = ref('')

function openPreview(record: TimelineEventAdmin) {
  previewTitle.value = record.title
  previewContent.value = record.content
  previewOpen.value = true
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="时间线管理" description="维护前台「时间线」页面的节点：日期、描述、关联文章与项目">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新建节点
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
        :scroll="{ x: 1160 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #emptyText>
          <TableEmpty text="暂无时间线节点，点击右上角「新建节点」添加" />
        </template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'eventDate'">
            {{ formatTime(record.eventDate, false) }}
          </template>

          <template v-else-if="column.key === 'post'">
            <a v-if="record.post" :href="`/post/${record.post.slug}`" target="_blank" rel="noopener noreferrer">
              {{ record.post.title }}
            </a>
            <span v-else class="text-muted">—</span>
          </template>

          <template v-else-if="column.key === 'projectName'">
            <a v-if="record.projectUrl" :href="record.projectUrl" target="_blank" rel="noopener noreferrer">
              {{ record.projectName || record.projectUrl }}
            </a>
            <template v-else>{{ record.projectName || '—' }}</template>
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
              <a-button type="link" size="small" @click="openPreview(record)">预览</a-button>
              <a-popconfirm title="删除后不可恢复，确定删除该节点吗？" ok-text="删除" cancel-text="取消" @confirm="onDelete(record)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:open="modalOpen"
      :title="editingId ? '编辑节点' : '新建节点'"
      :confirm-loading="modalSaving"
      ok-text="保存"
      cancel-text="取消"
      width="640px"
      @ok="submitModal"
    >
      <a-form ref="formRef" :model="modalForm" :rules="modalRules" layout="vertical">
        <a-form-item label="标题" name="title">
          <a-input v-model:value="modalForm.title" placeholder="节点标题，如：博客上线" :maxlength="100" show-count />
        </a-form-item>
        <a-form-item label="日期" required>
          <!-- 日期存于独立 eventDateValue ref（与 PostEditView 同约定），不走 form model 校验，提交前手动判空 -->
          <a-date-picker
            :value="eventDateValue"
            show-time
            style="width: 100%"
            placeholder="节点日期"
            @update:value="(v: Dayjs | null) => (eventDateValue = v)"
          />
        </a-form-item>
        <a-form-item label="描述（Markdown）" name="content">
          <a-textarea v-model:value="modalForm.content" placeholder="选填，支持 Markdown" :rows="5" :maxlength="5000" show-count />
        </a-form-item>
        <a-form-item label="配图" name="image">
          <a-input v-model:value="modalForm.image" placeholder="选填，图片地址">
            <template #addon-after>
              <a-button type="link" size="small" style="padding: 0" @click="mediaOpen = true">从媒体库选择</a-button>
            </template>
          </a-input>
        </a-form-item>
        <a-form-item label="关联文章" name="postId">
          <a-select
            v-model:value="modalForm.postId"
            :options="postOptions"
            :loading="postOptionsLoading"
            show-search
            option-filter-prop="label"
            allow-clear
            placeholder="选填，在前台展示文章入口"
          />
        </a-form-item>
        <a-form-item label="关联项目名称" name="projectName">
          <a-input v-model:value="modalForm.projectName" placeholder="选填，如：MyBlog" :maxlength="100" />
        </a-form-item>
        <a-form-item label="关联项目链接" name="projectUrl">
          <a-input v-model:value="modalForm.projectUrl" placeholder="选填，https://github.com/..." />
        </a-form-item>
        <a-form-item label="是否显示" name="visible">
          <a-switch v-model:checked="modalForm.visible" />
        </a-form-item>
        <a-form-item label="排序" name="sort">
          <a-input-number v-model:value="modalForm.sort" style="width: 160px" />
          <p class="form-hint">升序在前；默认 0 即按日期倒序，负值可置顶</p>
        </a-form-item>
      </a-form>
    </a-modal>

    <MediaSelectModal v-model:open="mediaOpen" @select="(url: string) => (modalForm.image = url)" />

    <a-modal v-model:open="previewOpen" :title="`预览：${previewTitle}`" :footer="null" width="720px">
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
.form-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--admin-muted);
}
</style>
