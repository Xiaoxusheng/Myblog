<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { UploadRequestOption } from 'ant-design-vue/es/vc-upload/interface'
import {
  CopyOutlined,
  DeleteOutlined,
  EyeOutlined,
  ReloadOutlined,
  UploadOutlined,
} from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { deleteUpload, getUploads, uploadImage } from '@/api/media'
import { copyText, formatBytes, formatTime } from '@/utils/format'
import type { UploadItem } from '@/types/api'

type ViewMode = 'grid' | 'list'

const loading = ref(false)
const list = ref<UploadItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 24
const uploading = ref(0)
const viewMode = ref<ViewMode>('grid')

const viewOptions = [
  { label: '网格', value: 'grid' },
  { label: '列表', value: 'list' },
]

const pagination = computed(() => ({
  current: page.value,
  pageSize,
  total: total.value,
  size: 'small' as const,
  showSizeChanger: false,
  showTotal: (t: number) => `共 ${t} 张`,
  onChange: onPageChange,
}))

async function load() {
  loading.value = true
  try {
    const result = await getUploads({ page: page.value, pageSize })
    list.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

function onPageChange(current: number) {
  page.value = current
  void load()
}

/** a-upload customRequest → POST /admin/uploads */
async function customRequest(options: UploadRequestOption) {
  uploading.value += 1
  try {
    await uploadImage(options.file as File)
    options.onSuccess?.(options.file)
    message.success('上传成功')
    page.value = 1
    await load()
  } catch (error) {
    options.onError?.(error as Error)
  } finally {
    uploading.value -= 1
  }
}

function beforeUpload(file: File) {
  if (!file.type.startsWith('image/')) {
    message.error('仅支持上传图片文件')
    return false
  }
  if (file.size > 10 * 1024 * 1024) {
    message.error('图片大小不能超过 10MB')
    return false
  }
  return true
}

// 预览（Modal 放大）
const previewOpen = ref(false)
const previewItem = ref<UploadItem | null>(null)

function openPreview(item: UploadItem) {
  previewItem.value = item
  previewOpen.value = true
}

async function onCopy(item: UploadItem) {
  const ok = await copyText(item.url)
  if (ok) {
    message.success('链接已复制')
  }
}

async function onDelete(item: UploadItem) {
  await deleteUpload(item.id)
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
    <PageHeader title="媒体库" description="上传与管理图片素材（仅图片，单张 ≤ 10MB）">
      <template #actions>
        <a-segmented v-model:value="viewMode" :options="viewOptions" />
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-upload
          accept="image/*"
          :show-upload-list="false"
          :custom-request="customRequest"
          :before-upload="beforeUpload"
        >
          <a-button type="primary" :loading="uploading > 0">
            <template #icon><UploadOutlined /></template>
            上传图片
          </a-button>
        </a-upload>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <a-spin :spinning="loading">
        <a-empty v-if="!loading && list.length === 0" description="媒体库暂无图片，点击右上角上传" />

        <!-- 网格视图：4:3 缩略图，hover 显示操作 -->
        <template v-else-if="viewMode === 'grid'">
          <a-row :gutter="[16, 16]">
            <a-col v-for="item in list" :key="item.id" :xs="12" :sm="8" :md="6" :lg="4">
              <div class="media-card">
                <div class="media-card__thumb" @click="openPreview(item)">
                  <img :src="item.url" :alt="item.filename" loading="lazy" />
                  <div class="media-card__overlay" @click.stop>
                    <a-tooltip title="预览">
                      <a-button type="text" size="small" class="media-card__btn" @click="openPreview(item)">
                        <template #icon><EyeOutlined /></template>
                      </a-button>
                    </a-tooltip>
                    <a-tooltip title="复制 URL">
                      <a-button type="text" size="small" class="media-card__btn" @click="onCopy(item)">
                        <template #icon><CopyOutlined /></template>
                      </a-button>
                    </a-tooltip>
                    <a-popconfirm
                      title="删除后不可恢复，文件将一并删除，确定删除该图片吗？"
                      ok-text="删除"
                      cancel-text="取消"
                      @confirm="onDelete(item)"
                    >
                      <a-tooltip title="删除">
                        <a-button type="text" size="small" class="media-card__btn media-card__btn--danger" @click.stop>
                          <template #icon><DeleteOutlined /></template>
                        </a-button>
                      </a-tooltip>
                    </a-popconfirm>
                  </div>
                </div>
                <a-tooltip :title="item.filename">
                  <div class="media-card__name">{{ item.filename }}</div>
                </a-tooltip>
                <div class="media-card__meta">
                  {{ formatBytes(item.size) }} · {{ formatTime(item.createdAt, false) }}
                </div>
              </div>
            </a-col>
          </a-row>
        </template>

        <!-- 列表视图：紧凑行 -->
        <template v-else>
          <div class="media-list">
            <div v-for="item in list" :key="item.id" class="media-row">
              <img class="media-row__thumb" :src="item.url" :alt="item.filename" loading="lazy" @click="openPreview(item)" />
              <div class="media-row__main">
                <a-tooltip :title="item.filename">
                  <div class="media-row__name">{{ item.filename }}</div>
                </a-tooltip>
                <div class="media-row__meta">
                  {{ item.mime }} · {{ formatBytes(item.size) }} · {{ formatTime(item.createdAt) }}
                </div>
              </div>
              <a-space :size="0" class="media-row__actions">
                <a-button type="text" size="small" @click="openPreview(item)">预览</a-button>
                <a-button type="text" size="small" @click="onCopy(item)">复制 URL</a-button>
                <a-popconfirm
                  title="删除后不可恢复，文件将一并删除，确定删除该图片吗？"
                  ok-text="删除"
                  cancel-text="取消"
                  @confirm="onDelete(item)"
                >
                  <a-button type="text" size="small" danger>删除</a-button>
                </a-popconfirm>
              </a-space>
            </div>
          </div>
        </template>

        <!-- 分页 -->
        <div v-if="total > pageSize" class="media-pager">
          <a-pagination v-bind="pagination" />
        </div>
      </a-spin>
    </a-card>

    <!-- 预览放大 -->
    <a-modal
      v-model:open="previewOpen"
      :title="previewItem?.filename"
      :footer="null"
      width="720px"
      centered
    >
      <img
        v-if="previewItem"
        :src="previewItem.url"
        :alt="previewItem.filename"
        style="display: block; width: 100%; height: auto"
      />
      <div v-if="previewItem" class="media-preview__meta">
        {{ previewItem.mime }} · {{ formatBytes(previewItem.size) }} · {{ formatTime(previewItem.createdAt) }}
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
/* ---------- 网格视图 ---------- */
.media-card__thumb {
  position: relative;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border-radius: var(--admin-radius-sm);
  background: var(--admin-surface-2);
  cursor: zoom-in;
}

.media-card__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.media-card__overlay {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  justify-content: center;
  gap: 4px;
  padding: 4px 0;
  background: rgba(0, 0, 0, 0.55);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.media-card__thumb:hover .media-card__overlay {
  opacity: 1;
}

.media-card__btn {
  color: #fff;
}

.media-card__btn--danger {
  color: #ff7875;
}

.media-card__name {
  margin-top: 8px;
  font-size: 13px;
  color: var(--admin-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.media-card__meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

/* ---------- 列表视图 ---------- */
.media-list {
  display: flex;
  flex-direction: column;
}

.media-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 4px;
}

.media-row + .media-row {
  border-top: 1px solid var(--admin-border);
}

.media-row__thumb {
  flex: none;
  width: 60px;
  height: 60px;
  object-fit: cover;
  border-radius: var(--admin-radius-sm);
  background: var(--admin-surface-2);
  cursor: zoom-in;
}

.media-row__main {
  flex: 1;
  min-width: 0;
}

.media-row__name {
  font-size: 13px;
  color: var(--admin-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.media-row__meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

.media-row__actions {
  flex: none;
}

/* ---------- 分页 / 预览 ---------- */
.media-pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.media-preview__meta {
  margin-top: 8px;
  font-size: 12px;
  color: var(--admin-muted);
  text-align: center;
}
</style>
