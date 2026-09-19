<template>
  <header class="site-header">
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
        <SearchBox class="header-search" />
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

    <Transition name="menu">
      <div v-if="menuOpen" class="mobile-panel">
        <SearchBox />
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
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useSiteStore } from '@/stores/site'
import SearchBox from '@/components/common/SearchBox.vue'
import ThemeToggle from '@/components/common/ThemeToggle.vue'

const route = useRoute()
const site = useSiteStore()
const menuOpen = ref(false)

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
</script>

<style scoped>
.site-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--header-bg);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--border);
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
  font-weight: 700;
  font-size: 17px;
  letter-spacing: 0.2px;
}

.brand:hover {
  color: var(--text-1);
}

.brand img {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  object-fit: cover;
}

.main-nav {
  display: flex;
  gap: 2px;
  margin-left: 6px;
}

.nav-link {
  padding: 6px 12px;
  border-radius: 8px;
  color: var(--text-2);
  font-size: 14px;
  transition: background var(--transition), color var(--transition);
}

.nav-link:hover {
  color: var(--text-1);
  background: var(--surface-2);
}

.nav-link.active {
  color: var(--brand);
  background: var(--brand-soft);
  font-weight: 500;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

.header-search {
  width: 190px;
}

.menu-toggle {
  display: none;
}

.mobile-panel {
  display: none;
}

@media (max-width: 767px) {
  .main-nav,
  .header-search {
    display: none;
  }

  .menu-toggle {
    display: inline-flex;
  }

  .mobile-panel {
    display: block;
    border-top: 1px solid var(--border);
    background: var(--surface);
    padding: 12px 20px 16px;
  }

  .mobile-nav {
    display: flex;
    flex-direction: column;
    margin-top: 10px;
  }

  .mobile-nav a {
    padding: 11px 4px;
    color: var(--text-1);
    font-size: 15px;
    border-bottom: 1px solid var(--border);
  }

  .mobile-nav a.active {
    color: var(--brand);
    font-weight: 500;
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
