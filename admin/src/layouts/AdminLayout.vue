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
  MenuFoldOutlined,
  MenuOutlined,
  MenuUnfoldOutlined,
  BulbOutlined,
  FundOutlined,
  PictureOutlined,
  ProfileOutlined,
  FileDoneOutlined,
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

/**
 * 侧栏导航的键盘可达性补齐。
 *
 * 上游缺口（实测，AntD Vue 4.2）：
 * 1. Menu 声明了 `tabindex` prop，但内置 Overflow 渲染并未把它写到 ul 上 → 侧栏进不了 Tab 顺序；
 * 2. 菜单项固定 `tabindex="-1"`，且 inline 模式未实现方向键 roving focus → 即使容器可聚焦也选不中任何一项。
 *
 * 做法：把焦点与键盘处理挂在自己拥有的 `<nav>` 包装元素上（原生属性 + Vue 事件，随组件生命周期自动回收），
 * 由它代理方向键移动焦点、Enter/Space 触发当前项。不改变任何路由与权限逻辑。
 */
function onMenuKeydown(event: KeyboardEvent): void {
  const nav = event.currentTarget as HTMLElement | null
  if (!nav) return
  const items = Array.from(nav.querySelectorAll<HTMLElement>('.ant-menu-item'))
  if (items.length === 0) return

  const active = document.activeElement as HTMLElement | null
  const index = active ? items.indexOf(active) : -1

  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    const step = event.key === 'ArrowDown' ? 1 : -1
    const next =
      index < 0 ? (step === 1 ? 0 : items.length - 1) : (index + step + items.length) % items.length
    items[next].focus()
    return
  }

  if ((event.key === 'Enter' || event.key === ' ') && index >= 0) {
    event.preventDefault()
    items[index].click()
  }
}

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
      { key: '/health', title: '系统状态', icon: FundOutlined },
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
        <span class="sider-logo__mark" aria-hidden="true">M</span>
        <span v-if="!collapsed" class="sider-logo__text">
          MyBlog<span class="sider-logo__sub">管理后台</span>
        </span>
      </div>
      <nav class="sidebar-nav" aria-label="管理菜单" tabindex="0" @keydown="onMenuKeydown">
        <a-menu theme="dark" mode="inline" :selected-keys="[activeKey]">
          <a-menu-item-group v-for="group in menuGroups" :key="group.key" :title="group.label">
            <a-menu-item v-for="item in group.items" :key="item.key" @click="onMenuClick(item)">
              <!-- 图标走 icon 插槽：作为子节点传入会被包进 title-content，折叠态下会被一起隐藏 -->
              <template #icon><component :is="item.icon" /></template>
              <span>{{ item.title }}</span>
            </a-menu-item>
          </a-menu-item-group>
        </a-menu>
      </nav>
      <template #trigger>
        <button
          type="button"
          class="sider-trigger"
          :aria-label="collapsed ? '展开侧边栏' : '收起侧边栏'"
          :aria-expanded="!collapsed"
          @click.stop="collapsed = !collapsed"
        >
          <MenuUnfoldOutlined v-if="collapsed" />
          <MenuFoldOutlined v-else />
        </button>
      </template>
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
        <span class="sider-logo__mark" aria-hidden="true">M</span>
        <span class="sider-logo__text">
          MyBlog<span class="sider-logo__sub">管理后台</span>
        </span>
      </div>
      <nav class="sidebar-nav" aria-label="管理菜单" tabindex="0" @keydown="onMenuKeydown">
        <a-menu
          theme="dark"
          mode="inline"
          :selected-keys="[activeKey]"
          style="border-inline-end: 0"
        >
          <a-menu-item-group v-for="group in menuGroups" :key="group.key" :title="group.label">
            <a-menu-item v-for="item in group.items" :key="item.key" @click="onMenuClick(item)">
              <template #icon><component :is="item.icon" /></template>
              <span>{{ item.title }}</span>
            </a-menu-item>
          </a-menu-item-group>
        </a-menu>
      </nav>
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

/* 品牌区：与 favicon 同一套视觉（品牌色方块 + 衬线 M），是唯一允许渐变的品牌位；
   文案双字重排版，副标与主标拉开明度差 */
.sider-logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  margin-bottom: 4px;
}

.sider-logo__mark {
  flex: none;
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--admin-brand), var(--admin-brand-hover));
  /* 内描边让方块在深色侧栏上边缘清晰，不靠加重投影 */
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.16);
  border-radius: 7px;
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
  color: #fff;
  letter-spacing: 0;
}

.sider-logo__text {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.2px;
}

.sider-logo__sub {
  margin-left: 7px;
  font-size: 11px;
  font-weight: 400;
  letter-spacing: 1px;
  color: rgba(255, 255, 255, 0.5);
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
  transition: background-color var(--admin-dur) var(--admin-ease);
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

/* 侧栏/抽屉背景单一来源 --admin-sidebar（light 深藏青 / dark 面板色），菜单透明继承（docs/09 §8.1） */
.admin-sider.ant-layout-sider-dark {
  background: var(--admin-sidebar);
}

/* 折叠 trigger 与侧栏同色系，hover 轻提亮 */
.admin-sider .ant-layout-sider-trigger {
  background: rgba(255, 255, 255, 0.04);
  transition: background-color var(--admin-dur) var(--admin-ease);
}

.admin-sider .ant-layout-sider-trigger:hover {
  background: rgba(255, 255, 255, 0.1);
}

.admin-sider .ant-menu,
.admin-drawer .ant-menu {
  background: transparent;
  border-inline-end: 0;
}

/* 抽屉 logo 复用 .sider-logo 品牌区结构，无需额外样式 */

/* 菜单分组标题：统一 12px / 500 / 0.6px 字距，组间靠同一档上下留白拉节奏 */
.admin-sider .ant-menu-item-group-title,
.admin-drawer .ant-menu-item-group-title {
  padding: 12px 16px 6px;
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  letter-spacing: 0.6px;
  /* 0.38 在深藏青/深灰底上对比不足，提到 0.52 保证小字可读 */
  color: rgba(255, 255, 255, 0.52);
}

/* 折叠态隐藏分组标题，仅保留图标项 */
.admin-sider .ant-menu-inline-collapsed .ant-menu-item-group-title {
  display: none;
}

/* 菜单项：40px 行高；选中态 = 柔和品牌底 + 左侧 2px 指示条 + 提亮文字，
   三重表达而非只靠一根蓝色描边 */
.admin-sider .ant-menu-item,
.admin-drawer .ant-menu-item {
  height: 40px;
  line-height: 40px;
  margin-inline: 0;
  margin-block: 1px;
  width: 100%;
  padding-inline: 16px;
  border-radius: 0;
  color: rgba(255, 255, 255, 0.74);
  transition:
    color var(--admin-dur) var(--admin-ease),
    background-color var(--admin-dur) var(--admin-ease);
}

/* 图标盒宽与字号统一：不同图标字宽不同，固定 16px 盒保证文字起点对齐。
   水平间距交给 AntD 原生规则（.ant-menu-item-icon 已带 10px），此处不再叠加。 */
.admin-sider .ant-menu-item .anticon,
.admin-drawer .ant-menu-item .anticon {
  width: 16px;
  min-width: 16px;
  font-size: 16px;
  vertical-align: -0.2em;
}

.admin-sider .ant-menu-item:hover,
.admin-drawer .ant-menu-item:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.06);
}

.admin-sider .ant-menu-item-selected,
.admin-drawer .ant-menu-item-selected {
  position: relative;
  font-weight: 600;
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

/* 键盘焦点：深色侧栏上必须有可见焦点环。
   焦点环挂在自有的 nav 包装元素上（不依赖 AntD 内部节点是否可聚焦）。 */
.sidebar-nav:focus-visible,
.admin-sider .ant-menu-item:focus-visible,
.admin-drawer .ant-menu-item:focus-visible {
  outline: 2px solid var(--admin-brand);
  outline-offset: -2px;
}

/* 折叠触发器：改用真实 button 承载，键盘可聚焦可触发（原实现是不可聚焦的 div） */
.sider-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 0;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.72);
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: color var(--admin-dur) var(--admin-ease);
}

.sider-trigger:hover {
  color: #fff;
}

.sider-trigger:focus-visible {
  outline: 2px solid var(--admin-brand);
  outline-offset: -2px;
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
