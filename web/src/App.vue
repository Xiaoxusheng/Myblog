<template>
  <div class="app-shell">
    <SiteHeader />
    <main class="app-main">
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" :key="route.path" />
        </Transition>
      </RouterView>
    </main>
    <SiteFooter />
    <BackTop />
    <ToastHost />
    <CommandPalette />
  </div>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import SiteHeader from '@/components/layout/SiteHeader.vue'
import SiteFooter from '@/components/layout/SiteFooter.vue'
import BackTop from '@/components/common/BackTop.vue'
import ToastHost from '@/components/common/ToastHost.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import { useSiteStore } from '@/stores/site'
import { useThemeStore } from '@/stores/theme'
import { applyDocumentTitle } from '@/utils/title'

const site = useSiteStore()
useThemeStore() // 初始化三态主题（立即应用 + 监听系统变化）
const route = useRoute()

onMounted(() => {
  void site.ensureLoaded()
})

// 站点名异步加载完成后，同步刷新当前页标题
watch(
  () => site.settings.siteName,
  () => applyDocumentTitle(route.meta.title)
)
</script>
