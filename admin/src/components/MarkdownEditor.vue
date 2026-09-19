<script setup lang="ts">
import { ref } from 'vue'
import { useFeedback } from '@/composables/useFeedback'
const { message } = useFeedback()
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import { uploadImage } from '@/api/media'

/**
 * Markdown 编辑器封装（md-editor-v3）
 * - 含实时预览
 * - 粘贴/插入图片走 /admin/uploads 上传接口
 */
const props = defineProps<{
  modelValue: string
  height?: number
  placeholder?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const uploading = ref(false)

function onUploadImg(files: File[], callback: (urls: string[]) => void) {
  if (!files || files.length === 0) return
  uploading.value = true
  Promise.all(
    files.map(async (file) => {
      try {
        const result = await uploadImage(file)
        return result.url
      } catch {
        // 失败占位，过滤掉
        return ''
      }
    }),
  )
    .then((urls) => {
      const ok = urls.filter((url) => url)
      if (ok.length > 0) {
        callback(ok)
        message.success(`已上传 ${ok.length} 张图片`)
      }
    })
    .finally(() => {
      uploading.value = false
    })
}
</script>

<template>
  <MdEditor
    :model-value="props.modelValue"
    editor-id="md-editor-admin"
    :style="{ height: `${props.height ?? 520}px` }"
    :placeholder="props.placeholder || '请输入 Markdown 内容'"
    language="zh-CN"
    no-katex
    no-mermaid
    @on-upload-img="onUploadImg"
    @update:model-value="emit('update:modelValue', $event)"
  />
</template>
