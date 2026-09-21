<template>
  <!-- /tags 与 /categories 是同一套「分类与标签」视图：路径决定初始 Tab。
       这里直接复用 CategoriesView，避免维护第二套实现。 -->
  <CategoriesView />
</template>

<script setup lang="ts">
import CategoriesView from '@/views/CategoriesView.vue'
</script>
