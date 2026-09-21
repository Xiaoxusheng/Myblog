<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { UploadRequestOption } from 'ant-design-vue/es/vc-upload/interface'
import {
  CopyOutlined,
  DeleteOutlined,
  EyeOutlined,
  ReloadOutlined,
  SearchOutlined,
  UploadOutlined,
} from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import LoadError from '@/components/LoadError.vue'
import { useFeedback } from '@/composables/useFeedback'
import { deleteUpload, getUploads, uploadImage } from '@/api/media'
import { getHealth } from '@/api/health'
import { copyText, formatBytes, formatTime } from '@/utils/format'
import type { UploadItem } from '@/types/api'

type ViewMode = 'grid' | 'list'

const { message } = useFeedback()

const loading = ref(false)
const error = ref('')
const list = ref<UploadItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 24
const uploading = ref(0)
const viewMode = ref<ViewMode>('grid')

// ---------- 存储用量（v2 设计稿第 7 页）----------
// 数据来自 /admin/health 契约 #84 的 mediaCount / uploadSize。
// 契约明确不返回磁盘配额（"不含磁盘空间等平台敏感信息"），因此这里显示**真实用量**
// 而不画假进度条：没有真实分母就不画分母，避免用一个编造的百分比误导判断。
const mediaCount = ref<number | null>(null)
const uploadSize = ref<number | null>(null)

async function loadStorage() {
  try {
    const info = await getHealth()
    mediaCount.value = info.mediaCount
    uploadSize.value = info.uploadSize
  } catch {
    // 用量为辅助信息：失败时整条隐藏，不影响媒体列表主流程
    mediaCount.value = null
    uploadSize.value = null
  }
}

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
  showTotal: (t: number) => `共 ${t} 个文件`,
  onChange: onPageChange,
}))

/** 存储用量摘要：有真实数据才渲染整条 */
const storageReady = computed(() => uploadSize.value !== null)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await getUploads({ page: page.value, pageSize })
    list.value = result.list
    total.value = result.total
  } catch {
    error.value = '媒体列表加载失败，请检查网络后重试'
  } finally {
    loading.value = false
  }
}

/**
 * 文件名过滤。
 *
 * 接口契约 #46 只接受 page/pageSize（无 keyword 参数），且本轮红线是零 API 变更，
 * 因此这里**只过滤当前页**并在 UI 上如实标注「当前页」，不谎称全库搜索。
 * 若后续要在全库范围内搜，需先在 api.md 给 #46 补 keyword 参数再改这里。
 */
const keywordInput = ref('')
const keyword = ref('')

const visibleList = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return list.value
  return list.value.filter((item) => item.filename.toLowerCase().includes(q))
})

/** 当前是否处于过滤态（用于区分「真空库」与「筛选无结果」两种空态文案） */
const filtering = computed(() => keyword.value.trim().length > 0)

function onSearch() {
  keyword.value = keywordInput.value
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

async function deleteAndReload(id: number) {
  await deleteUpload(id)
  message.success('删除成功')
  if (list.value.length === 1 && page.value > 1) {
    page.value -= 1
  }
  await Promise.all([load(), loadStorage()])
}

onMounted(() => {
  void load()
  void loadStorage()
})
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
      <!-- 存储用量（v2 设计稿第 7 页）：只显示真实用量，无配额字段故不画假进度 -->
      <div v-if="storageReady" class="storage">
        <div class="storage__head">
          <span class="storage__label">存储空间</span>
          <span class="storage__value tabular-nums">
            已用 {{ formatBytes(uploadSize ?? 0) }}
            <template v-if="mediaCount !== null"> · {{ mediaCount }} 个文件</template>
          </span>
        </div>
        <!-- 无真实配额：用一条纯示意刻度的轨道表达"当前不是靠容量驱动"，
             不用百分比填充，避免用户把装饰条当成真实的余量读数 -->
        <div class="storage__track" aria-hidden="true"></div>
      </div>

      <!-- 工具栏：搜索与视图/上传分离成行（设计稿为独立工具栏行） -->
      <div class="table-toolbar">
        <div class="table-toolbar__filters">
          <a-input
            v-model:value="keywordInput"
            placeholder="搜索文件名"
            style="width: 240px"
            allow-clear
            @press-enter="onSearch"
          >
            <template #prefix><SearchOutlined /></template>
          </a-input>
          <a-button @click="onSearch">
            <template #icon><SearchOutlined /></template>
            搜索
          </a-button>
        </div>
        <div class="media-toolbar__right">
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
              上传素材
            </a-button>
          </a-upload>
        </div>
      </div>

      <LoadError v-if="error" :message="error" @retry="load" />

      <a-spin v-else :spinning="loading">
        <a-empty
          v-if="!loading && visibleList.length === 0"
          :description="
            filtering ? `当前页没有匹配「${keyword}」的文件，试试调整关键词` : '媒体库暂无图片，点击右上角上传'
          "
        />

        <!-- 网格视图：4:3 缩略图，hover 显示操作 -->
        <template v-else-if="viewMode === 'grid'">
          <a-row :gutter="[16, 16]">
            <a-col v-for="item in visibleList" :key="item.id" :xs="12" :sm="8" :md="6" :lg="4">
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
                      @confirm="deleteAndReload(item.id)"
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
            <div v-for="item in visibleList" :key="item.id" class="media-row">
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
                  @confirm="deleteAndReload(item.id)"
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
/* ---------- 存储用量（v2 设计稿第 7 页）---------- */
.storage {
  padding: 12px 14px;
  margin-bottom: 16px;
  background: var(--admin-surface-2);
  border-radius: var(--admin-radius-md);
}

.storage__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.storage__label {
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-text);
}

.storage__value {
  font-size: 12px;
  color: var(--admin-muted);
}

/* 无真实配额时的刻度轨：只作视觉锚点，不填充百分比 */
.storage__track {
  height: 6px;
  margin-top: 10px;
  border-radius: 3px;
  background: repeating-linear-gradient(
    90deg,
    color-mix(in srgb, var(--admin-border) 90%, transparent) 0 6px,
    transparent 6px 12px
  );
}

/* ---------- 工具栏右组 ---------- */
.media-toolbar__right {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

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
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
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
