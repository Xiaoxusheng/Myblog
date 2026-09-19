<template>
  <div class="container custom-page">
    <div v-if="loading" class="page-skeleton card" aria-hidden="true">
      <span class="skeleton sk-title"></span>
      <span v-for="i in 6" :key="i" class="skeleton sk-line" :style="{ width: i % 2 ? '94%' : '72%' }"></span>
    </div>

    <ErrorState v-else-if="error && !notFound" :message="error" @retry="load" />

    <EmptyState
      v-else-if="notFound"
      title="页面不存在"
      description="它可能已被删除，或者链接有误。"
    >
      <RouterLink class="btn" to="/">返回首页</RouterLink>
    </EmptyState>

    <template v-else-if="page">
      <header class="page-title-bar">
        <h1 class="page-heading">{{ page.title }}</h1>
      </header>
      <div class="page-content card">
        <!-- 内容来自管理员维护的 Markdown，渲染时不放行内嵌 HTML -->
        <div class="markdown-body" v-html="html"></div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { fetchPage } from '@/api/content'
import { ApiError } from '@/api/http'
import { renderMarkdown } from '@/utils/markdown'
import { applyDocumentTitle } from '@/utils/title'
import type { CustomPage } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

const route = useRoute()
const slug = computed(() => String(route.params.slug))

const page = ref<CustomPage | null>(null)
const html = ref('')
const loading = ref(true)
const error = ref('')
const notFound = ref(false)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  notFound.value = false
  try {
    const data = await fetchPage(slug.value)
    page.value = data
    html.value = renderMarkdown(data.content ?? '').html
    applyDocumentTitle(data.title)
  } catch (e) {
    if (e instanceof ApiError && e.code === 10004) {
      notFound.value = true
    } else {
      error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

watch(slug, () => void load(), { immediate: true })
</script>

<style scoped>
.custom-page {
  max-width: 860px;
  padding-top: 28px;
  padding-bottom: 56px;
}

.page-content {
  margin-top: 18px;
  padding: 28px 32px;
}

@media (max-width: 640px) {
  .page-content {
    padding: 20px 18px;
  }
}

.page-skeleton {
  margin-top: 28px;
  padding: 28px 32px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sk-title {
  height: 26px;
  width: 40%;
  border-radius: 8px;
  margin-bottom: 8px;
}

.sk-line {
  display: block;
  height: 13px;
}
</style>
