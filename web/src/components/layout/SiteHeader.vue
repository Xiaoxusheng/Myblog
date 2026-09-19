<template>
  <header class="site-header" :class="{ scrolled }">
    <div class="header-inner container">
      <RouterLink to="/" class="brand">
        <img v-if="site.settings.logo" :src="site.settings.logo" alt="" />
        <span>{{ site.settings.siteName || 'MyBlog' }}</span>
      </RouterLink>

      <nav class="main-nav" aria-label="主导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav-link"
          :class="{ active: item.active }"
        >
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="header-actions">
        <button
          class="search-trigger"
          type="button"
          aria-label="打开搜索"
          aria-haspopup="dialog"
          @click="palette.open()"
        >
          <svg viewBox="0 0 24 24" width="15" height="15" fill="currentColor" aria-hidden="true">
            <path d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 1 0-.7.7l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0A4.5 4.5 0 1 1 14 9.5 4.5 4.5 0 0 1 9.5 14z" />
          </svg>
          <span class="st-text">搜索…</span>
          <span class="kbd st-kbd">{{ kbdHint }}</span>
        </button>
        <ThemeToggle />
        <button
          class="icon-btn menu-toggle"
          :aria-expanded="menuOpen"
          aria-label="打开菜单"
          @click="menuOpen = !menuOpen"
        >
          <svg v-if="!menuOpen" viewBox="0 0 24 24" width="19" height="19" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M4 7h16M4 12h16M4 17h16" />
          </svg>
          <svg v-else viewBox="0 0 24 24" width="19" height="19" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
      </div>
    </div>

    <Teleport to="body">
      <Transition name="mask">
        <div v-if="menuOpen" class="menu-mask" aria-hidden="true" @click="menuOpen = false"></div>
      </Transition>
    </Teleport>

    <Transition name="menu">
      <div v-if="menuOpen" class="mobile-panel">
        <button
          class="search-trigger search-trigger-full"
          type="button"
          aria-label="打开搜索"
          aria-haspopup="dialog"
          @click="onMobileSearch"
        >
          <svg viewBox="0 0 24 24" width="15" height="15" fill="currentColor" aria-hidden="true">
            <path d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 1 0-.7.7l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0A4.5 4.5 0 1 1 14 9.5 4.5 4.5 0 0 1 9.5 14z" />
          </svg>
          <span class="st-text">搜索文章…</span>
        </button>
        <nav class="mobile-nav" aria-label="移动端导航">
          <RouterLink
            v-for="item in navItems"
            :key="item.to"
            :to="item.to"
            :class="{ active: item.active }"
            @click="menuOpen = false"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </div>
    </Transition>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useSiteStore } from '@/stores/site'
import { useCommandPalette } from '@/composables/useCommandPalette'
import ThemeToggle from '@/components/common/ThemeToggle.vue'

const route = useRoute()
const site = useSiteStore()
const palette = useCommandPalette()
const menuOpen = ref(false)
const scrolled = ref(false)

/** kbd 徽标按平台显示：Mac 为 ⌘ K，其余为 Ctrl K（触屏移动端由 CSS 隐藏） */
const kbdHint = computed(() =>
  /mac|iphone|ipad|ipod/i.test(navigator.platform || navigator.userAgent) ? '⌘ K' : 'Ctrl K'
)

function onMobileSearch(): void {
  menuOpen.value = false
  palette.open()
}

function onScroll(): void {
  scrolled.value = window.scrollY > 4
}

const navItems = computed(() => {
  const path = route.path
  return [
    { label: '首页', to: '/', active: path === '/' },
    { label: '归档', to: '/archives', active: path.startsWith('/archives') },
    {
      label: '分类',
      to: '/categories',
      active: path.startsWith('/categories') || path.startsWith('/category/')
    },
    { label: '标签', to: '/tags', active: path.startsWith('/tags') || path.startsWith('/tag/') },
    { label: '专题', to: '/series', active: path.startsWith('/series') },
    { label: '友链', to: '/links', active: path.startsWith('/links') }
  ]
})

// 路由变化时收起移动端菜单
watch(
  () => route.fullPath,
  () => {
    menuOpen.value = false
  }
)

// 菜单展开时锁 body 滚动（与 TOC 抽屉同机制）
watch(menuOpen, (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
})

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>

<style scoped>
.site-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--header-bg);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid transparent;
  transition: background 0.25s ease-out, border-color 0.25s ease-out;
}

/* 滚动后只出现 hairline，不加投影——编辑部式分层 */
.site-header.scrolled {
  background: var(--header-bg-scrolled);
  border-bottom-color: var(--border);
}

.header-inner {
  display: flex;
  align-items: center;
  gap: 20px;
  height: var(--header-height);
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  color: var(--text-1);
  /* 报头化：衬线站名（docs/05 §5.1） */
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: 17.5px;
  line-height: 1.3;
}

.brand:hover {
  color: var(--text-1);
}

.brand img {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  object-fit: cover;
}

.main-nav {
  display: flex;
  gap: 4px;
  margin-left: 6px;
}

/* 编辑部式导航：hover 与 active 只动文字与下划线，不做底色胶囊。
   hover 是墨色下划线，active 才是 accent——弱化引导、强化当前位置（docs/07 §6.3） */
.nav-link {
  position: relative;
  padding: 6px 10px;
  color: var(--text-2);
  font-size: 14px;
  transition: color var(--transition);
}

.nav-link::after {
  content: '';
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: -2px;
  height: 2px;
  border-radius: 1px;
  background: var(--border-strong);
  opacity: 0;
  transform: scaleX(0.4);
  transform-origin: center;
  transition: opacity var(--transition), transform var(--transition),
    background var(--transition);
}

.nav-link:hover {
  color: var(--text-1);
}

.nav-link:hover::after {
  opacity: 1;
  transform: scaleX(1);
}

.nav-link.active {
  color: var(--text-1);
  font-weight: 500;
}

.nav-link.active::after {
  opacity: 1;
  transform: none;
  background: var(--brand);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

/* 搜索入口：外观延续搜索框形态，语义为打开命令面板的按钮（docs/06 §5） */
.search-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 190px;
  height: 34px;
  padding: 0 8px 0 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--surface-2);
  color: var(--text-3);
  font-size: 13.5px;
  transition: border-color var(--transition), background var(--transition),
    color var(--transition), transform var(--transition);
}

.search-trigger:hover {
  background: var(--border);
  color: var(--text-2);
}

.search-trigger:active {
  transform: scale(0.96);
}

.st-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}

.st-kbd {
  flex-shrink: 0;
}

.menu-toggle {
  display: none;
}

.mobile-panel {
  display: none;
}

/* 菜单遮罩：Teleport 到 body，z-index 位于 header(100) 之下、内容之上 */
.menu-mask {
  position: fixed;
  inset: 0;
  z-index: 99;
  background: rgba(0, 0, 0, 0.4);
}

.mask-enter-active,
.mask-leave-active {
  transition: opacity 0.18s ease;
}

.mask-enter-from,
.mask-leave-to {
  opacity: 0;
}

@media (max-width: 767px) {
  .main-nav,
  .search-trigger:not(.search-trigger-full) {
    display: none;
  }

  .search-trigger-full {
    width: 100%;
  }

  .st-kbd {
    display: none;
  }

  .menu-toggle {
    display: inline-flex;
  }

  .mobile-panel {
    display: block;
    border-bottom: 1px solid var(--border);
    background: var(--bg);
    padding: 12px 20px 16px;
  }

  .mobile-nav {
    display: flex;
    flex-direction: column;
    margin-top: 10px;
  }

  .mobile-nav a {
    padding: 12px 4px;
    color: var(--text-1);
    font-size: 15px;
    border-bottom: 1px solid var(--border);
  }

  .mobile-nav a.active {
    color: var(--brand);
    font-weight: 500;
  }

  /* 末项去分隔线，避免与面板底边 hairline 叠成双线（docs/08 §6.4） */
  .mobile-nav a:last-child {
    border-bottom: none;
  }
}

.menu-enter-active,
.menu-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.menu-enter-from,
.menu-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
