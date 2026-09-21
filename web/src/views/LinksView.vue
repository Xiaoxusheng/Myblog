<template>
  <div class="container links-page">
    <header class="page-title-bar">
      <p class="kicker">LINKS · 交换友链</p>
      <h1 class="page-heading">友情链接</h1>
      <p v-if="links.length" class="page-sub">共 {{ links.length }} 位朋友，欢迎互相交换。</p>
    </header>

    <div v-if="loading" class="link-grid" aria-hidden="true">
      <span v-for="i in 6" :key="i" class="skeleton sk-link"></span>
    </div>

    <ErrorState v-else-if="error" :message="error" @retry="load" />

    <template v-else-if="links.length">
      <div class="link-grid">
        <a
          v-for="(link, index) in links"
          :key="link.id"
          :href="link.url"
          target="_blank"
          rel="noopener noreferrer"
          class="link-card"
        >
          <span class="link-logo" :class="`tint-${TINTS[index % TINTS.length]}`">
            <img
              v-if="link.logo && !logoFailed.has(link.id)"
              :src="link.logo"
              :alt="link.name"
              loading="lazy"
              @error="logoFailed.add(link.id)"
            />
            <span v-else class="logo-fallback">{{ fallbackChar(link.name) }}</span>
          </span>
          <span class="link-info">
            <span class="link-name">{{ link.name }}</span>
            <span class="link-desc">{{ link.description || '这位朋友还没有留下介绍' }}</span>
          </span>
        </a>
      </div>

      <!-- 底部 CTA：交换友链（p11） -->
      <section class="cta-block">
        <div>
          <h2 class="cta-title">想交换友链？</h2>
          <p class="cta-desc">把你的站点信息发给我，合适的话很快加上。</p>
        </div>
        <a class="btn btn-primary" :href="mailtoHref">发送邮件</a>
      </section>
    </template>

    <EmptyState
      v-else
      title="暂无友链"
      description="博主还没有添加任何友情链接。"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { fetchLinks } from '@/api/content'
import { ApiError, isRequestCanceled } from '@/api/http'
import { useSiteStore } from '@/stores/site'
import type { LinkItem } from '@/types'
import EmptyState from '@/components/common/EmptyState.vue'
import ErrorState from '@/components/common/ErrorState.vue'

const TINTS = ['sky', 'mint', 'peach', 'lavender'] as const

const site = useSiteStore()
const links = ref<LinkItem[]>([])
const loading = ref(true)
const error = ref('')
const logoFailed = reactive(new Set<number>())
let controller: AbortController | null = null

/** 邮件地址未在设置中提供，用站点域名兜底 */
const mailtoHref = computed(() => {
  const host = site.settings.siteUrl?.replace(/^https?:\/\//, '').replace(/\/$/, '')
  return `mailto:hi@${host || 'myblog.dev'}?subject=${encodeURIComponent('友链申请')}`
})

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
  padding-bottom: 64px;
}

.link-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 24px;
}

.link-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg);
  transition: border-color var(--transition), background var(--transition);
}

.link-card:hover {
  border-color: var(--border-strong);
  background: var(--surface);
}

/* 首字母头像块：tint 色 + 深色字母（p11） */
.link-logo {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  overflow: hidden;
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
  color: var(--text-1);
  font-size: 18px;
  font-weight: 700;
}

.tint-sky {
  background: var(--tint-sky);
}

.tint-mint {
  background: var(--tint-mint);
}

.tint-peach {
  background: var(--tint-peach);
}

.tint-lavender {
  background: var(--tint-lavender);
}

.link-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.link-name {
  align-self: flex-start;
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--text-1);
  transition: color var(--transition);
}

.link-card:hover .link-name {
  color: var(--brand);
}

.link-desc {
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 底部 CTA（p11） */
.cta-block {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-top: 32px;
  padding: 26px 28px;
  border-radius: var(--radius-lg);
  background: var(--surface);
}

.cta-title {
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: var(--fs-lg);
  line-height: var(--lh-heading-2);
  letter-spacing: -0.01em;
}

.cta-desc {
  margin-top: 6px;
  font-size: var(--fs-sm);
  color: var(--text-2);
}

.sk-link {
  display: block;
  height: 82px;
  border-radius: var(--radius-lg);
}

@media (max-width: 640px) {
  .link-grid {
    grid-template-columns: 1fr;
  }

  .cta-block {
    flex-direction: column;
    align-items: flex-start;
    padding: 20px;
  }

  .cta-block .btn {
    width: 100%;
  }
}
</style>
