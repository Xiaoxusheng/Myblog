<script setup lang="ts">
import { Empty } from 'ant-design-vue'

/**
 * 表格空态统一(docs/09 §5.2):SIMPLE 插图 + 业务文案
 */
withDefaults(defineProps<{ text?: string }>(), {
  text: '暂无数据'
})

const simpleImage = Empty.PRESENTED_IMAGE_SIMPLE
</script>

<template>
  <a-empty :image="simpleImage" :description="text" />
</template>
