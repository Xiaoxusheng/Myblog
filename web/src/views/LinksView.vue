<template>
  <div class="container links-page">
    <header class="page-title-bar">
      <h1 class="page-heading">友情链接</h1>
      <p v-if="links.length" class="page-sub">共 {{ links.length }} 位朋友</p>
    </header>

    <div v-if="loading" class="link-grid" aria-hidden="true">
      <span v-for="i in 6" :key="i" class="skeleton sk-link card"></span>
    </div>

    <ErrorState v-else-if="error" :message="error" @retry="load" />

    <div v-else-if="links.length" class="link-grid">
      <a
        v-for="link in links"
        :key="link.id"
        :href="link.url"
        target="_blank"
        rel="noopener noreferrer"
        class="link-card card"
      >
        <span class="link-logo">
          <img v-if="!logoFailed.has(link.id)" :src="link.logo" :alt="link.name" loading="lazy" @error="logoFailed.add(link.id)" />
          <span v-else class="logo-fallback">{{ fallbackChar(link.name) }}</span>
        </span>
        <span class="link-info">
          <span class="link-name">{{ link.name }}</span>
          <span class="link-desc">{{ link.description || '这位朋友还没有留下介绍' }}</span>
        </span>
      </a>
    </div>

    <EmptyState
      v-else
      title="暂无友链"
      description="博主还没有添加任何友情链接。"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { fetchLinks } from '@/api/content'
import { ApiError, isRequestCanceled } from '@/api/http'
import type { LinkItem } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

const links = ref<LinkItem[]>([])
const loading = ref(true)
const error = ref('')
const logoFailed = reactive(new Set<number>())
let controller: AbortController | null = null

function fallbackChar(name: string): string {
  return (name || '链').trim().charAt(0).toUpperCase()
}

async function load(): Promise<void> {
  controller?.abort()
  const localController = new AbortController()
  controller = localController
  loading.value = true
  error.value = ''
  try {
    const data = await fetchLinks(localController.signal)
    if (controller !== localController) return
    links.value = data
  } catch (e) {
    if (controller !== localController || isRequestCanceled(e)) return
    error.value = e instanceof ApiError ? e.message : '加载失败，请稍后重试'
  } finally {
    if (controller === localController) loading.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<style scoped>
.links-page {
  padding-bottom: 56px;
}

.link-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 22px;
}

@media (max-width: 900px) {
  .link-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .link-grid {
    grid-template-columns: 1fr;
  }
}

.link-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  transition: border-color var(--transition), box-shadow 0.2s ease-out, transform 0.2s ease-out;
}

.link-card:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.link-logo {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--border);
  background: var(--surface-2);
}

.link-logo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-fallback {
  width: 100%;
  height: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--brand);
  font-size: 19px;
  font-weight: 600;
}

.link-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.link-name {
  font-size: 14.5px;
  font-weight: 600;
  color: var(--text-1);
  transition: color var(--transition);
}

.link-card:hover .link-name {
  color: var(--brand);
}

.link-desc {
  font-size: 12.5px;
  color: var(--text-3);
  line-height: 1.6;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sk-link {
  display: block;
  height: 82px;
}
</style>
