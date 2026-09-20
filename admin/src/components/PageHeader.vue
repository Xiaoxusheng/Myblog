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
      <a-typography-title :level="4" class="page-header__title">{{ title }}</a-typography-title>
      <a-typography-text v-if="description" type="secondary" class="page-header__desc">
        {{ description }}
      </a-typography-text>
    </div>
    <a-space>
      <slot name="actions"></slot>
    </a-space>
  </div>
</template>

<style scoped>
/* 标题：收紧字距提升排印质感；描述与标题拉开一级明度差 */
.page-header__title {
  margin: 0 !important;
  letter-spacing: -0.2px;
}

.page-header__desc {
  margin-top: 4px;
  font-size: 13px;
}
</style>
