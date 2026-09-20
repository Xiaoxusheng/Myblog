<script setup lang="ts">
import { ref, watch } from 'vue'
import { useFeedback } from '@/composables/useFeedback'
const { message } = useFeedback()
import { CopyOutlined, UploadOutlined } from '@ant-design/icons-vue'
import { deleteUpload, getUploads, uploadImage } from '@/api/media'
import { copyText, formatBytes } from '@/utils/format'
import type { UploadItem } from '@/types/api'

/**
 * 从媒体库选择图片（文章封面 / Logo 等）
 */
const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'select', url: string): void
}>()

const list = ref<UploadItem[]>([])
const loading = ref(false)
const page = ref(1)
const total = ref(0)
const pageSize = 24
const uploading = ref(false)

async function load() {
  loading.value = true
  try {
    const result = await getUploads({ page: page.value, pageSize })
    list.value = result.list
    total.value = result.total
  } catch {
    // 接口层已 toast 具体原因，这里仅保证不产生未处理的 rejection
  } finally {
    loading.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      page.value = 1
      void load()
    }
  },
)

function onSelect(url: string) {
  emit('select', url)
  emit('update:open', false)
}

async function onCopy(url: string) {
  const ok = await copyText(url)
  if (ok) {
    message.success('链接已复制')
  }
}

async function onDelete(item: UploadItem) {
  await deleteUpload(item.id)
  message.success('已删除')
  await load()
}

async function onUpload(file: File) {
  uploading.value = true
  try {
    await uploadImage(file)
    message.success('上传成功')
    page.value = 1
    await load()
  } catch {
    // 接口层已 toast 具体原因（上传失败原因），这里吞掉避免未处理 rejection
  } finally {
    uploading.value = false
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
  void onUpload(file)
  return false
}
</script>

<template>
  <a-modal
    :open="props.open"
    title="从媒体库选择"
    width="880px"
    :footer="null"
    @update:open="emit('update:open', $event)"
  >
    <div class="media-select">
      <div class="media-select__toolbar">
        <a-typography-text type="secondary">点击图片即可选中</a-typography-text>
        <a-upload accept="image/*" :show-upload-list="false" :before-upload="beforeUpload">
          <a-button type="primary" :loading="uploading">
            <template #icon><UploadOutlined /></template>
            上传图片
          </a-button>
        </a-upload>
      </div>

      <a-spin :spinning="loading">
        <a-empty v-if="!loading && list.length === 0" description="媒体库暂无图片" />
        <template v-else>
          <a-row :gutter="[12, 12]">
            <a-col v-for="item in list" :key="item.id" :xs="12" :sm="8" :md="6" :lg="6">
              <div class="media-select__item" @click="onSelect(item.url)">
                <div class="media-select__thumb">
                  <img :src="item.url" :alt="item.filename" loading="lazy" />
                </div>
                <div class="media-select__name">
                  <a-tooltip :title="item.filename">
                    <span class="media-select__filename">{{ item.filename }}</span>
                  </a-tooltip>
                </div>
                <div class="media-select__meta">
                  <span>{{ formatBytes(item.size) }}</span>
                  <a-space :size="4">
                    <a-tooltip title="复制链接">
                      <a-button type="text" size="small" @click.stop="onCopy(item.url)">
                        <template #icon><CopyOutlined /></template>
                      </a-button>
                    </a-tooltip>
                    <a-popconfirm
                      title="删除后不可恢复，确定删除该图片吗？"
                      ok-text="删除"
                      cancel-text="取消"
                      @confirm="onDelete(item)"
                    >
                      <a-button type="text" size="small" danger @click.stop>删除</a-button>
                    </a-popconfirm>
                  </a-space>
                </div>
              </div>
            </a-col>
          </a-row>
          <div class="media-select__pager">
            <a-pagination
              v-model:current="page"
              size="small"
              :total="total"
              :page-size="pageSize"
              :show-size-changer="false"
              @change="load"
            />
          </div>
        </template>
      </a-spin>
    </div>
  </a-modal>
</template>

<style scoped>
.media-select__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.media-select__item {
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-sm);
  padding: 8px;
  cursor: pointer;
  transition: border-color 0.2s;
}

.media-select__item:hover {
  border-color: var(--admin-brand);
}

.media-select__thumb {
  height: 96px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: var(--admin-surface-2);
  border-radius: 4px;
}

.media-select__thumb img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.media-select__name {
  margin-top: 8px;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.media-select__filename {
  color: var(--admin-text);
}

.media-select__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 12px;
  color: var(--admin-muted);
}

.media-select__pager {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}
</style>
