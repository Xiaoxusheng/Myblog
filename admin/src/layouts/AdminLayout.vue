<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Modal } from 'ant-design-vue'
import {
  AppstoreOutlined,
  CommentOutlined,
  DashboardOutlined,
  FileTextOutlined,
  LinkOutlined,
  LogoutOutlined,
  MenuOutlined,
  PictureOutlined,
  ProfileOutlined,
  SettingOutlined,
  TagsOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const collapsed = ref(false)
/** 小屏（≤768px）侧边栏改抽屉，不常驻占宽 */
const isMobile = ref(false)
const drawerOpen = ref(false)
let mediaQuery: MediaQueryList | null = null

function onMediaChange(e: MediaQueryListEvent): void {
  isMobile.value = e.matches
  if (!isMobile.value) drawerOpen.value = false
}

interface MenuItem {
  key: string
  title: string
  icon: typeof DashboardOutlined
}

const menuItems: MenuItem[] = [
  { key: '/dashboard', title: '仪表盘', icon: DashboardOutlined },
  { key: '/posts', title: '文章管理', icon: FileTextOutlined },
  { key: '/comments', title: '评论管理', icon: CommentOutlined },
  { key: '/categories', title: '分类管理', icon: AppstoreOutlined },
  { key: '/tags', title: '标签管理', icon: TagsOutlined },
  { key: '/pages', title: '页面管理', icon: ProfileOutlined },
  { key: '/links', title: '友链管理', icon: LinkOutlined },
  { key: '/media', title: '媒体库', icon: PictureOutlined },
  { key: '/settings', title: '系统设置', icon: SettingOutlined },
]

const activeKey = computed(() => (route.meta.activeMenu as string) || route.path)

interface Crumb {
  title: string
  path?: string
}

const breadcrumbs = computed<Crumb[]>(() => {
  const items: Crumb[] = []
  if (route.meta.parentTitle) {
    items.push({ title: route.meta.parentTitle as string, path: route.meta.parentPath as string })
  }
  if (route.meta.title) {
    items.push({ title: route.meta.title as string })
  }
  return items
})

function onMenuClick(item: MenuItem) {
  if (route.path !== item.key) {
    void router.push(item.key)
  }
  if (isMobile.value) drawerOpen.value = false
}

function goProfile() {
  void router.push('/profile')
}

function onChangePassword() {
  // 个人资料页包含修改密码表单
  void router.push({ path: '/profile', query: { focus: 'password' } })
}

function onLogout() {
  Modal.confirm({
    title: '退出登录',
    content: '确定要退出登录吗？',
    okText: '退出',
    cancelText: '取消',
    onOk: () => {
      auth.logout()
      void router.replace('/login')
    },
  })
}

onMounted(() => {
  // 刷新用户信息（头像/昵称），401 由拦截器统一处理
  if (auth.isLoggedIn && !auth.user) {
    void auth.fetchMe()
  }
  mediaQuery = window.matchMedia('(max-width: 768px)')
  isMobile.value = mediaQuery.matches
  mediaQuery.addEventListener('change', onMediaChange)
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', onMediaChange)
})
</script>

<template>
  <a-layout class="admin-layout">
    <!-- 桌面端常驻侧边栏 -->
    <a-layout-sider
      v-if="!isMobile"
      v-model:collapsed="collapsed"
      collapsible
      breakpoint="lg"
      :width="208"
      theme="dark"
    >
      <div class="sider-logo">
        <span v-if="!collapsed">MyBlog 管理后台</span>
        <span v-else>MB</span>
      </div>
      <a-menu theme="dark" mode="inline" :selected-keys="[activeKey]">
        <a-menu-item v-for="item in menuItems" :key="item.key" @click="onMenuClick(item)">
          <component :is="item.icon" />
          <span>{{ item.title }}</span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>

    <!-- 移动端抽屉侧边栏 -->
    <a-drawer
      v-if="isMobile"
      v-model:open="drawerOpen"
      placement="left"
      :width="232"
      :closable="false"
      class="admin-drawer"
      root-class-name="admin-drawer"
    >
      <div class="sider-logo drawer-logo">
        <span>MyBlog 管理后台</span>
      </div>
      <a-menu
        theme="dark"
        mode="inline"
        :selected-keys="[activeKey]"
        style="border-inline-end: 0"
      >
        <a-menu-item v-for="item in menuItems" :key="item.key" @click="onMenuClick(item)">
          <component :is="item.icon" />
          <span>{{ item.title }}</span>
        </a-menu-item>
      </a-menu>
    </a-drawer>

    <a-layout>
      <a-layout-header class="admin-header">
        <MenuOutlined v-if="isMobile" class="admin-header__menu" @click="drawerOpen = true" />

        <a-breadcrumb class="admin-header__crumbs">
          <a-breadcrumb-item>
            <RouterLink to="/dashboard">首页</RouterLink>
          </a-breadcrumb-item>
          <a-breadcrumb-item v-for="(crumb, index) in breadcrumbs" :key="index">
            <RouterLink v-if="crumb.path" :to="crumb.path">{{ crumb.title }}</RouterLink>
            <span v-else>{{ crumb.title }}</span>
          </a-breadcrumb-item>
        </a-breadcrumb>

        <a-dropdown>
          <div class="admin-header__user" role="button" tabindex="0">
            <a-avatar :size="28" :src="auth.user?.avatar || undefined">
              <template #icon><UserOutlined /></template>
            </a-avatar>
            <span class="admin-header__username">{{ auth.displayName }}</span>
          </div>
          <template #overlay>
            <a-menu>
              <a-menu-item key="profile" @click="goProfile">
                <UserOutlined />
                <span style="margin-left: 8px">个人资料</span>
              </a-menu-item>
              <a-menu-item key="password" @click="onChangePassword">
                <SettingOutlined />
                <span style="margin-left: 8px">修改密码</span>
              </a-menu-item>
              <a-menu-divider />
              <a-menu-item key="logout" danger @click="onLogout">
                <LogoutOutlined />
                <span style="margin-left: 8px">退出登录</span>
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </a-layout-header>

      <a-layout-content class="admin-content">
        <RouterView />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.admin-layout {
  min-height: 100%;
}

.sider-logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 1px;
  white-space: nowrap;
  overflow: hidden;
}

.admin-header {
  height: 56px;
  line-height: normal;
  padding: 0 24px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.admin-header__crumbs {
  flex: 1;
  min-width: 0;
}

.admin-header__user {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
}

.admin-header__user:hover {
  background: rgba(0, 0, 0, 0.04);
}

.admin-header__username {
  color: rgba(0, 0, 0, 0.88);
}

.admin-content {
  /* 内容自然撑开，页面级滚动 */
  min-height: auto;
}

@media (max-width: 768px) {
  .admin-header {
    padding: 0 12px;
    gap: 8px;
  }

  .admin-header__menu {
    font-size: 18px;
    padding: 6px;
    cursor: pointer;
    color: rgba(0, 0, 0, 0.72);
  }

  .admin-header__username {
    display: none;
  }
}
</style>

<style>
/* 抽屉传送到 body，需全局样式：深色底 + 菜单贴合边缘 */
.admin-drawer .ant-drawer-body {
  padding: 0;
  background: #001529;
}

.admin-drawer .drawer-logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 1px;
  white-space: nowrap;
}
</style>
