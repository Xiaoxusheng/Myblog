<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { UploadRequestOption } from 'ant-design-vue/es/vc-upload/interface'
import { CopyOutlined, DeleteOutlined, UploadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { deleteUpload, getUploads, uploadImage } from '@/api/media'
import { copyText, formatBytes, formatTime } from '@/utils/format'
import type { UploadItem } from '@/types/api'

const loading = ref(false)
const list = ref<UploadItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 24
const uploading = ref(0)

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
        <a-list
          v-else
          :grid="{ gutter: 16, xs: 1, sm: 2, md: 3, lg: 4, xl: 6 }"
          :data-source="list"
          :pagination="total > pageSize ? pagination : false"
        >
          <template #renderItem="{ item }">
            <a-list-item>
              <div class="media-item">
                <div class="media-item__thumb">
                  <a-image
                    :src="item.url"
                    :alt="item.filename"
                    style="width: 100%; height: 120px; object-fit: contain"
                  />
                </div>
                <a-tooltip :title="item.filename">
                  <div class="media-item__name">{{ item.filename }}</div>
                </a-tooltip>
                <div class="media-item__meta">
                  <span>{{ formatBytes(item.size) }} · {{ formatTime(item.createdAt, false) }}</span>
                </div>
                <div class="media-item__actions">
                  <a-button type="link" size="small" @click="onCopy(item)">
                    <template #icon><CopyOutlined /></template>
                    复制链接
                  </a-button>
                  <a-popconfirm title="删除后将同时删除文件，确定吗？" ok-text="删除" cancel-text="取消" @confirm="onDelete(item)">
                    <a-button type="link" size="small" danger>
                      <template #icon><DeleteOutlined /></template>
                      删除
                    </a-button>
                  </a-popconfirm>
                </div>
              </div>
            </a-list-item>
          </template>
        </a-list>
      </a-spin>
    </a-card>
  </div>
</template>

<style scoped>
.media-item {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 8px;
}

.media-item__thumb {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  border-radius: 4px;
  overflow: hidden;
}

.media-item__name {
  margin-top: 8px;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.88);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.media-item__meta {
  margin-top: 2px;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.media-item__actions {
  margin-top: 4px;
  display: flex;
  justify-content: space-between;
}

.media-item__actions :deep(.ant-btn) {
  padding-inline: 4px;
}
</style>
