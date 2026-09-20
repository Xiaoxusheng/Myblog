<script setup lang="ts">
/**
 * 表单分组卡片：统一小标题（13px / 600 / muted）+ 16px 内边距
 * 用于文章编辑等 CMS 布局的右侧/纵向分组
 */
defineProps<{
  title: string
  /** 锚点名（渲染为 data-section），供发布前检查「定位」跳转 */
  anchor?: string
}>()
</script>

<template>
  <section class="form-section" :data-section="anchor">
    <h3 class="form-section__title">{{ title }}</h3>
    <div class="form-section__body">
      <slot></slot>
    </div>
  </section>
</template>

<style scoped>
.form-section {
  padding: 16px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-sm);
}

.form-section__title {
  margin: 0 0 12px;
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  color: var(--admin-muted);
}
</style>
