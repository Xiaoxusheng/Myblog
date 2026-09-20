<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useFeedback } from '@/composables/useFeedback'
const { message } = useFeedback()
import type { FormInstance, TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import { PictureOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import MediaSelectModal from '@/components/MediaSelectModal.vue'
import {
  createSeries,
  deleteSeries,
  getSeriesList,
  getSeriesPosts,
  reorderSeriesPosts,
  updateSeries,
} from '@/api/series'
import { formatTime } from '@/utils/format'
import type { PostStatus, Series, SeriesPostItem } from '@/types/api'

const loading = ref(false)
const list = ref<Series[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)

const columns: TableColumnsType = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 220 },
  { title: '文章数', dataIndex: 'postCount', key: 'postCount', width: 90 },
  { title: '显隐', key: 'visible', width: 90 },
  { title: '排序', dataIndex: 'sort', key: 'sort', width: 80 },
  { title: '创建时间', key: 'createdAt', width: 150 },
  { title: '操作', key: 'action', width: 190, fixed: 'right' },
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
    const result = await getSeriesList({ page: page.value, pageSize: pageSize.value })
    list.value = result.list
    total.value = result.total
  } catch {
    // 接口层已 toast 具体原因；此处兜底避免未处理 rejection
  } finally {
    loading.value = false
  }
}

function onTableChange(paginationConfig: TablePaginationConfig) {
  page.value = paginationConfig.current || 1
  pageSize.value = paginationConfig.pageSize || 50
  void load()
}

// 新建 / 编辑弹窗
const modalOpen = ref(false)
const modalSaving = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()
const coverModalOpen = ref(false)

const modalForm = reactive({
  name: '',
  slug: '',
  description: '',
  cover: '',
  visible: true,
  sort: 0,
})

const modalRules = {
  name: [{ required: true, message: '请输入专题名称' }],
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
  modalForm.cover = ''
  modalForm.visible = true
  modalForm.sort = 0
  modalOpen.value = true
}

function openEdit(record: Series) {
  editingId.value = record.id
  modalForm.name = record.name
  modalForm.slug = record.slug
  modalForm.description = record.description || ''
  modalForm.cover = record.cover || ''
  modalForm.visible = record.visible
  modalForm.sort = record.sort
  modalOpen.value = true
}

async function submitModal() {
  await formRef.value?.validate()
  const payload = {
    name: modalForm.name.trim(),
    slug: modalForm.slug.trim(),
    description: modalForm.description.trim(),
    cover: modalForm.cover.trim(),
    visible: modalForm.visible,
    sort: modalForm.sort,
  }
  modalSaving.value = true
  try {
    if (editingId.value) {
      await updateSeries(editingId.value, payload)
      message.success('保存成功')
    } else {
      await createSeries(payload)
      message.success('创建成功')
    }
    modalOpen.value = false
    await load()
  } finally {
    modalSaving.value = false
  }
}

async function onDelete(record: Series) {
  await deleteSeries(record.id)
  message.success('删除成功')
  if (list.value.length === 1 && page.value > 1) {
    page.value -= 1
  }
  await load()
}

// ---------------------------------------------------------------------------
// 文章排序抽屉：上移/下移调整顺序，保存时按 1 递增提交全部行
// ---------------------------------------------------------------------------
const drawerOpen = ref(false)
const drawerLoading = ref(false)
const drawerSaving = ref(false)
const drawerSeries = ref<Series | null>(null)
const posts = ref<SeriesPostItem[]>([])
const dirty = ref(false)

/** 小屏（≤640px）抽屉占满宽度，避免溢出视口 */
const isCompact = ref(false)
const drawerWidth = computed(() => (isCompact.value ? '100%' : 480))
let mediaQuery: MediaQueryList | null = null

function onMediaChange(e: MediaQueryListEvent): void {
  isCompact.value = e.matches
}

const drawerTitle = computed(() =>
  drawerSeries.value ? `文章排序 · ${drawerSeries.value.name}` : '文章排序',
)

const statusTags: Record<PostStatus, { label: string; color: string }> = {
  0: { label: '草稿', color: 'default' },
  1: { label: '已发布', color: 'success' },
  2: { label: '隐藏', color: 'warning' },
  3: { label: '定时发布', color: 'processing' },
}

function statusTag(status: PostStatus) {
  return statusTags[status]
}

async function openPosts(record: Series) {
  drawerSeries.value = record
  drawerOpen.value = true
  await loadPosts()
}

async function loadPosts() {
  if (!drawerSeries.value) return
  drawerLoading.value = true
  try {
    const result = await getSeriesPosts(drawerSeries.value.id)
    posts.value = result.list
    dirty.value = false
  } finally {
    drawerLoading.value = false
  }
}

function move(index: number, offset: -1 | 1) {
  const target = index + offset
  if (target < 0 || target >= posts.value.length) return
  const next = [...posts.value]
  const current = next[index]
  const swapped = next[target]
  if (!current || !swapped) return
  next[index] = swapped
  next[target] = current
  posts.value = next
  dirty.value = true
}

async function saveOrder() {
  if (!drawerSeries.value) return
  drawerSaving.value = true
  try {
    await reorderSeriesPosts(
      drawerSeries.value.id,
      posts.value.map((item, index) => ({ postId: item.id, sort: index + 1 })),
    )
    message.success('排序已保存')
    dirty.value = false
    await loadPosts()
  } finally {
    drawerSaving.value = false
  }
}

onMounted(() => {
  void load()
  mediaQuery = window.matchMedia('(max-width: 640px)')
  isCompact.value = mediaQuery.matches
  mediaQuery.addEventListener('change', onMediaChange)
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', onMediaChange)
})
</script>

<template>
  <div class="page">
    <PageHeader title="专题" description="将相关文章组织成连载专题，按序号顺序展示">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-button type="primary" @click="openCreate">
          <template #icon><PlusOutlined /></template>
          新建专题
        </a-button>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 900 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div class="series-name">
              <span class="series-name__text">{{ record.name }}</span>
              <span class="series-name__slug">{{ record.slug }}</span>
            </div>
          </template>

          <template v-else-if="column.key === 'visible'">
            <a-tag :color="record.visible ? 'success' : 'default'">
              {{ record.visible ? '已展示' : '已隐藏' }}
            </a-tag>
          </template>

          <template v-else-if="column.key === 'createdAt'">
            {{ formatTime(record.createdAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="openEdit(record)">编辑</a-button>
              <a-button type="link" size="small" @click="openPosts(record)">文章排序</a-button>
              <a-popconfirm
                title="删除后不可恢复，该专题下的文章将变为未归属，确定删除吗？"
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

    <!-- 新建 / 编辑弹窗 -->
    <a-modal
      v-model:open="modalOpen"
      :title="editingId ? '编辑专题' : '新建专题'"
      :confirm-loading="modalSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitModal"
    >
      <a-form ref="formRef" :model="modalForm" :rules="modalRules" layout="vertical">
        <a-form-item label="名称" name="name">
          <a-input v-model:value="modalForm.name" placeholder="专题名称，需唯一" :maxlength="50" />
        </a-form-item>
        <a-form-item label="Slug" name="slug">
          <a-input v-model:value="modalForm.slug" placeholder="留空由后端生成" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="modalForm.description" placeholder="选填" :rows="3" :maxlength="200" />
        </a-form-item>
        <a-form-item label="封面" name="cover">
          <a-space direction="vertical" :size="8" style="width: 100%">
            <a-input v-model:value="modalForm.cover" placeholder="输入图片外链地址，留空无封面" allow-clear />
            <a-space wrap>
              <a-button size="small" @click="coverModalOpen = true">
                <template #icon><PictureOutlined /></template>
                从媒体库选择
              </a-button>
              <a-button v-if="modalForm.cover" size="small" @click="modalForm.cover = ''">清除</a-button>
            </a-space>
            <a-image v-if="modalForm.cover" :src="modalForm.cover" :width="160" />
          </a-space>
        </a-form-item>
        <a-form-item label="是否展示" name="visible">
          <a-switch v-model:checked="modalForm.visible" />
        </a-form-item>
        <a-form-item label="排序" name="sort">
          <a-input-number v-model:value="modalForm.sort" :min="0" :max="9999" style="width: 160px" />
        </a-form-item>
      </a-form>
    </a-modal>

    <MediaSelectModal v-model:open="coverModalOpen" @select="modalForm.cover = $event" />

    <!-- 文章排序抽屉 -->
    <a-drawer v-model:open="drawerOpen" :title="drawerTitle" :width="drawerWidth">
      <a-spin :spinning="drawerLoading">
        <a-empty
          v-if="!drawerLoading && posts.length === 0"
          description="该专题暂无文章，可先在文章编辑页将文章归入此专题"
        />
        <template v-else>
          <ul class="sort-list">
            <li v-for="(item, index) in posts" :key="item.id" class="sort-row">
              <div class="sort-row__main">
                <span class="sort-row__title" :title="item.title">{{ item.title }}</span>
                <span class="sort-row__meta">
                  <a-tag :color="statusTag(item.status).color" class="sort-row__tag">
                    {{ statusTag(item.status).label }}
                  </a-tag>
                  <span class="sort-row__sort">序号 {{ item.sort }}</span>
                </span>
              </div>
              <a-space :size="4" class="sort-row__actions">
                <a-button size="small" :disabled="index === 0" @click="move(index, -1)">上移</a-button>
                <a-button size="small" :disabled="index === posts.length - 1" @click="move(index, 1)">
                  下移
                </a-button>
              </a-space>
            </li>
          </ul>
          <div class="sort-footer">
            <a-button type="primary" :loading="drawerSaving" :disabled="!dirty" @click="saveOrder">
              保存排序
            </a-button>
          </div>
        </template>
      </a-spin>
    </a-drawer>
  </div>
</template>

<style scoped>
/* 名称单元格：主名 + slug 灰字两行 */
.series-name {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.series-name__text {
  font-weight: 500;
  color: var(--admin-text);
}

.series-name__slug {
  font-size: 12px;
  color: var(--admin-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 排序抽屉 */
.sort-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.sort-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
}

.sort-row + .sort-row {
  border-top: 1px solid var(--admin-border);
}

.sort-row__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sort-row__title {
  font-size: 14px;
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sort-row__meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sort-row__tag {
  margin: 0;
}

.sort-row__sort {
  font-size: 12px;
  color: var(--admin-muted);
  font-variant-numeric: tabular-nums;
}

.sort-row__actions {
  flex-shrink: 0;
}

.sort-footer {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--admin-border);
  display: flex;
  justify-content: flex-end;
}
</style>
