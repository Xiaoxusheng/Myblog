<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
const { modal } = useFeedback()
import { getNotifications, readAllNotifications, readNotification } from '@/api/notifications'
import { formatTime } from '@/utils/format'
import type { NotificationItem } from '@/types/api'
import {
  AppstoreOutlined,
  BarChartOutlined,
  BellOutlined,
  BookOutlined,
  CommentOutlined,
  ClockCircleOutlined,
  DashboardOutlined,
  EditOutlined,
  FileTextOutlined,
  LinkOutlined,
  LogoutOutlined,
  MenuOutlined,
  BulbOutlined,
  PictureOutlined,
  ProfileOutlined,
  FileDoneOutlined,
  HeartOutlined,
  ImportOutlined,
  RocketOutlined,
  SaveOutlined,
  SettingOutlined,
  StopOutlined,
  BulbFilled,
  SwapOutlined,
  TagsOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useAdminTheme } from '@/theme'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, toggle: toggleTheme } = useAdminTheme()

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

interface MenuGroup {
  key: string
  label: string
  items: MenuItem[]
}

/** 侧边栏视觉分组：数据 / 内容 / 互动 / 资源 / 系统 */
const menuGroups: MenuGroup[] = [
  {
    key: 'data',
    label: '数据',
    items: [{ key: '/analytics', title: '访问分析', icon: BarChartOutlined }],
  },
  {
    key: 'content',
    label: '内容',
    items: [
      { key: '/dashboard', title: '仪表盘', icon: DashboardOutlined },
      { key: '/posts', title: '文章管理', icon: FileTextOutlined },
      { key: '/posts/drafts', title: '草稿工作区', icon: EditOutlined },
      { key: '/pages', title: '页面管理', icon: ProfileOutlined },
      { key: '/series', title: '专题管理', icon: BookOutlined },
      { key: '/timeline', title: '时间线管理', icon: ClockCircleOutlined },
      { key: '/changelogs', title: '版本记录', icon: RocketOutlined },
    ],
  },
  {
    key: 'interact',
    label: '互动',
    items: [
      { key: '/comments', title: '评论管理', icon: CommentOutlined },
      { key: '/links', title: '友链管理', icon: LinkOutlined },
    ],
  },
  {
    key: 'resource',
    label: '资源',
    items: [
      { key: '/categories', title: '分类管理', icon: AppstoreOutlined },
      { key: '/tags', title: '标签管理', icon: TagsOutlined },
      { key: '/media', title: '媒体库', icon: PictureOutlined },
    ],
  },
  {
    key: 'system',
    label: '系统',
    items: [
      { key: '/redirects', title: '重定向管理', icon: SwapOutlined },
      { key: '/comment-blacklist', title: '评论防护', icon: StopOutlined },
      { key: '/backups', title: '备份', icon: SaveOutlined },
      { key: '/import-export', title: '导入导出', icon: ImportOutlined },
      { key: '/audit-logs', title: '操作日志', icon: FileDoneOutlined },
      { key: '/health', title: '系统状态', icon: HeartOutlined },
      { key: '/settings', title: '系统设置', icon: SettingOutlined },
    ],
  },
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

/** 用户菜单触发器键盘路径:合成 click 交给 Dropdown 的 click 触发器(docs/09 §9.2) */
function openUserMenu(e: KeyboardEvent) {
  ;(e.currentTarget as HTMLElement)?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
}

function onChangePassword() {
  // 个人资料页包含修改密码表单
  void router.push({ path: '/profile', query: { focus: 'password' } })
}

function onLogout() {
  modal.confirm({
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

// ---------- 通知中心（契约 #73-75）：进入后台拉取 + 60s 轮询 ----------
const notifOpen = ref(false)
const notifLoading = ref(false)
const notifList = ref<NotificationItem[]>([])
const notifUnread = ref(0)
let notifTimer: ReturnType<typeof setInterval> | null = null

async function loadNotifications() {
  notifLoading.value = true
  try {
    const data = await getNotifications()
    notifList.value = data.list
    notifUnread.value = data.unreadCount
  } catch {
    // 接口层已标记 silent，不弹全局 toast。
    // 这里必须吞掉异常：60s 轮询是 fire-and-forget（void），若不捕获会变成
    // 未处理 rejection，且失败时不应清空已有列表、打断用户。
  } finally {
    notifLoading.value = false
  }
}

async function onOpenNotification(n: NotificationItem) {
  if (!n.read) {
    await readNotification(n.id)
    n.read = true
    notifUnread.value = Math.max(0, notifUnread.value - 1)
  }
  notifOpen.value = false
  if (n.link) {
    void router.push(n.link)
  }
}

async function onReadAll() {
  await readAllNotifications()
  notifList.value = notifList.value.map((n) => ({ ...n, read: true }))
  notifUnread.value = 0
}

onMounted(() => {
  // 刷新用户信息（头像/昵称），401 由拦截器统一处理
  if (auth.isLoggedIn && !auth.user) {
    void auth.fetchMe()
  }
  mediaQuery = window.matchMedia('(max-width: 768px)')
  isMobile.value = mediaQuery.matches
  mediaQuery.addEventListener('change', onMediaChange)
  void loadNotifications()
  notifTimer = setInterval(() => void loadNotifications(), 60_000)
})

onBeforeUnmount(() => {
  if (notifTimer) {
    clearInterval(notifTimer)
    notifTimer = null
  }
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
      :width="200"
      theme="dark"
      class="admin-sider"
    >
      <div class="sider-logo">
        <span v-if="!collapsed">MyBlog 管理后台</span>
        <span v-else>MB</span>
      </div>
      <a-menu theme="dark" mode="inline" :selected-keys="[activeKey]">
        <a-menu-item-group v-for="group in menuGroups" :key="group.key" :title="group.label">
          <a-menu-item v-for="item in group.items" :key="item.key" @click="onMenuClick(item)">
            <component :is="item.icon" />
            <span>{{ item.title }}</span>
          </a-menu-item>
        </a-menu-item-group>
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
        <a-menu-item-group v-for="group in menuGroups" :key="group.key" :title="group.label">
          <a-menu-item v-for="item in group.items" :key="item.key" @click="onMenuClick(item)">
            <component :is="item.icon" />
            <span>{{ item.title }}</span>
          </a-menu-item>
        </a-menu-item-group>
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

        <!-- 通知中心（契约 #73）：60s 轮询未读数 -->
        <!-- 主题切换:light/dark 两态,偏好持久化(docs/09 §8.2) -->
        <a-tooltip :title="isDark ? '切换到亮色' : '切换到暗色'">
          <a-button type="text" shape="circle" :aria-label="isDark ? '切换到亮色' : '切换到暗色'" @click="toggleTheme">
            <template #icon>
              <BulbFilled v-if="isDark" />
              <BulbOutlined v-else />
            </template>
          </a-button>
        </a-tooltip>

        <a-popover
          v-model:open="notifOpen"
          trigger="click"
          placement="bottomRight"
          :width="330"
        >
          <template #content>
            <div class="notif-panel">
              <div class="notif-panel__head">
                <span>通知</span>
                <a-button v-if="notifList.length > 0" type="link" size="small" @click="onReadAll">
                  全部已读
                </a-button>
              </div>
              <a-spin :spinning="notifLoading">
                <div v-if="notifList.length === 0" class="notif-panel__empty">暂无通知</div>
                <div
                  v-for="n in notifList"
                  :key="n.id"
                  class="notif-panel__item"
                  :class="{ 'notif-panel__item--unread': !n.read }"
                  role="button"
                  tabindex="0"
                  :aria-label="`打开通知：${n.title}`"
                  @click="onOpenNotification(n)"
                  @keydown.enter.prevent="onOpenNotification(n)"
                >
                  <div class="notif-panel__title">
                    <a-badge v-if="!n.read" status="processing" />
                    {{ n.title }}
                  </div>
                  <div class="notif-panel__content">{{ n.content }}</div>
                  <div class="notif-panel__time">{{ formatTime(n.createdAt) }}</div>
                </div>
              </a-spin>
            </div>
          </template>
          <a-badge :count="notifUnread" :offset="[-2, 2]" size="small">
            <a-button type="text" shape="circle" aria-label="通知">
              <template #icon><BellOutlined /></template>
            </a-button>
          </a-badge>
        </a-popover>

        <a-dropdown>
          <div
            class="admin-header__user"
            role="button"
            tabindex="0"
            aria-label="账号菜单"
            @keydown.enter.prevent="openUserMenu"
            @keydown.space.prevent="openUserMenu"
          >
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
  background: var(--admin-bg);
}

/* 内层 a-layout 的 colorBgLayout 对齐页面底色 token */
.admin-layout :deep(.ant-layout) {
  background: var(--admin-bg);
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
  position: sticky;
  top: 0;
  z-index: 20;
  height: 56px;
  line-height: normal;
  padding: 0 24px;
  background: var(--admin-surface);
  border-bottom: 1px solid var(--admin-border);
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
  border-radius: var(--admin-radius-sm);
}

.admin-header__user:hover {
  background: rgba(0, 0, 0, 0.04);
}

.admin-header__username {
  color: var(--admin-text);
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
  background: var(--admin-sidebar);
}

/* 侧栏/抽屉背景单一来源 --admin-sidebar（light #001529 / dark 面板色），菜单透明继承（docs/09 §8.1） */
.admin-sider.ant-layout-sider-dark {
  background: var(--admin-sidebar);
}

.admin-sider .ant-menu,
.admin-drawer .ant-menu {
  background: transparent;
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

/* 菜单分组标题：小号灰字，组间不加大留白 */
.admin-sider .ant-menu-item-group-title,
.admin-drawer .ant-menu-item-group-title {
  padding: 10px 16px 4px;
  font-size: 12px;
  line-height: 18px;
  letter-spacing: 0.5px;
  color: rgba(255, 255, 255, 0.38);
}

/* 折叠态隐藏分组标题，仅保留图标项 */
.admin-sider .ant-menu-inline-collapsed .ant-menu-item-group-title {
  display: none;
}

/* 菜单项：38~40px 行高；选中态 = 浅色底 + 左侧 2px 指示线，避免大面积厚重蓝色块 */
.admin-sider .ant-menu-item,
.admin-drawer .ant-menu-item {
  height: 40px;
  line-height: 40px;
  margin-inline: 0;
  margin-block: 1px;
  width: 100%;
  border-radius: 0;
  color: rgba(255, 255, 255, 0.68);
}

.admin-sider .ant-menu-item:hover,
.admin-drawer .ant-menu-item:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.06);
}

.admin-sider .ant-menu-item-selected,
.admin-drawer .ant-menu-item-selected {
  position: relative;
  color: #fff;
  background: rgba(22, 119, 255, 0.18);
}

.admin-sider .ant-menu-item-selected::before,
.admin-drawer .ant-menu-item-selected::before {
  content: '';
  position: absolute;
  inset-inline-start: 0;
  inset-block: 0;
  width: 2px;
  background: var(--admin-brand);
}

/* 选中项右侧不再叠加 AntD 默认指示条 */
.admin-sider .ant-menu-item-selected::after,
.admin-drawer .ant-menu-item-selected::after {
  display: none;
}

/* 通知面板 */
.notif-panel__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  margin-bottom: 4px;
}

.notif-panel__empty {
  padding: 24px 0;
  text-align: center;
  color: var(--admin-muted);
  font-size: 13px;
}

.notif-panel__item {
  padding: 8px 4px;
  border-bottom: 1px solid var(--admin-border);
  cursor: pointer;
}

.notif-panel__item:last-child {
  border-bottom: none;
}

.notif-panel__item--unread {
  background: color-mix(in srgb, var(--admin-brand) 6%, transparent);
}

.notif-panel__title {
  font-size: 13px;
  font-weight: 500;
  display: flex;
  align-items: center;
}

.notif-panel__content {
  font-size: 12px;
  color: var(--admin-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
}

.notif-panel__time {
  font-size: 11px;
  color: var(--admin-muted);
  margin-top: 2px;
}

.admin-header :deep(.ant-btn) {
  margin-right: 8px;
}
</style>
