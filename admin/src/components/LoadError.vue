<script setup lang="ts">
/**
 * 页级加载错误块(docs/09 §5.2):发生了什么 + 能做什么 + 重试
 * 与 Dashboard/Analytics 已有 a-result 错误态同一语言
 */
withDefaults(defineProps<{ message?: string }>(), {
  message: '网络异常或服务暂时不可用，请稍后重试'
})

const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <a-result status="error" title="数据加载失败" :sub-title="message">
    <template #extra>
      <a-button type="primary" @click="emit('retry')">重新加载</a-button>
    </template>
  </a-result>
</template>
