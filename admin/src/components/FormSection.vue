<script setup lang="ts">
/**
 * 表单分组卡片：统一小标题（13px / 600 / muted）+ 16px 内边距
 * 用于文章编辑等 CMS 布局的右侧/纵向分组
 */
defineProps<{
  title: string
  /** 副标题：补充该组字段的用途说明，不与标题抢层级 */
  description?: string
  /** 锚点名（渲染为 data-section），供发布前检查「定位」跳转 */
  anchor?: string
  /**
   * 真实 DOM id，供页面内锚点导航（settings 左导航等）使用。
   * 与 anchor 分开：anchor 是"检查项定位"语义的 data 属性，id 是滚动锚点，
   * 两者用途不同，不合并以免既有调用方语义被污染。
   */
  id?: string
}>()
</script>

<template>
  <section :id="id" class="form-section" :data-section="anchor">
    <header class="form-section__head">
      <h3 class="form-section__title">{{ title }}</h3>
      <p v-if="description" class="form-section__desc">{{ description }}</p>
    </header>
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

.form-section__head {
  margin-bottom: 12px;
}

.form-section__title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  color: var(--admin-text);
}

.form-section__desc {
  margin: 2px 0 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

/* 有副标题时头部与内容拉大一点间距 */
.form-section__desc + .form-section__body {
  margin-top: 0;
}
</style>
