<script setup lang="ts">
/**
 * 页面头部三件套：标题 + 描述 + 动作
 */
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="page-header">
    <div class="page-header__meta">
      <a-typography-title :level="4" style="margin: 0">{{ title }}</a-typography-title>
      <a-typography-text v-if="description" type="secondary" class="page-header__desc">
        {{ description }}
      </a-typography-text>
    </div>
    <a-space>
      <slot name="actions"></slot>
    </a-space>
  </div>
</template>
