<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
const { modal } = useFeedback()
import { getNotifications, readAllNotifications, readNotification } from '@/api/notifications'
import { getHealth } from '@/api/health'
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
  SearchOutlined,
  SettingOutlined,
  StopOutlined,
  BulbFilled,
  SwapOutlined,
  TagsOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useAdminTheme } from '@/theme'
import CommandPalette, { type PaletteItem } from '@/components/CommandPalette.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, toggle: toggleTheme } = useAdminTheme()

const collapsed = ref(false)
/** 小屏（≤768px）侧边栏改抽屉，不常驻占宽 */
const isMobile = ref(false)
const drawerOpen = ref(false)
let mediaQuery: MediaQueryList | null = null

// ---------- 顶栏「服务正常」状态点（v2 设计稿常驻位）----------
// 复用 /admin/health 契约 #84 的 dbStatus，不新增接口。
// 失败与"检查中"都不说谎：只有明确 ok 才显示绿色「服务正常」。
type ServiceState = 'checking' | 'ok' | 'degraded'
const serviceState = ref<ServiceState>('checking')

async function loadServiceState() {
  try {
    const info = await getHealth()
    serviceState.value = info.dbStatus === 'ok' ? 'ok' : 'degraded'
  } catch {
    // 拦截器已提示；失败即视为异常，但静态提示不打扰（不弹 toast）
    serviceState.value = 'degraded'
  }
}

const SERVICE_TEXT: Record<ServiceState, string> = {
  checking: '检查中',
  ok: '服务正常',
  degraded: '服务异常',
}

/** 命令面板（⌘K / Ctrl+K）：跳转与快捷动作入口 */
const paletteOpen = ref(false)

/** 面板数据源 = 菜单树（唯一真相来源，不另维护一份跳转表） */
const paletteItems = computed<PaletteItem[]>(() =>
  menuGroups.flatMap((group) =>
    group.items.map((item) => ({ key: item.key, title: item.title, group: group.label, path: item.key })),
  ),
)

function openPalette() {
  paletteOpen.value = true
}

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
  void loadServiceState()
  notifTimer = setInterval(() => void loadNotifications(), 60_000)
  window.addEventListener('keydown', onGlobalKeydown)
})

onBeforeUnmount(() => {
  if (notifTimer) {
    clearInterval(notifTimer)
    notifTimer = null
  }
  mediaQuery?.removeEventListener('change', onMediaChange)
  window.removeEventListener('keydown', onGlobalKeydown)
})

/**
 * ⌘K / Ctrl+K 唤起命令面板。
 * 输入框内不劫持（用户在搜索框里按 Ctrl+K 应保持原生行为）。
 */
function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    const target = e.target as HTMLElement | null
    const tag = target?.tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || target?.isContentEditable) return
    e.preventDefault()
    openPalette()
  }
}
</script>

<template>
  <a-layout class="admin-layout">
    <!-- 桌面端常驻侧边栏 -->
    <a-layout-sider
      v-if="!isMobile"
      v-model:collapsed="collapsed"
      collapsible
      breakpoint="lg"
      :width="232"
      theme="light"
      class="admin-sider"
    >
      <div class="sider-logo">
        <span class="sider-logo__mark" aria-hidden="true">M</span>
        <span v-if="!collapsed" class="sider-logo__text">
          MyBlog<span class="sider-logo__sub">管理后台</span>
        </span>
      </div>

      <!-- 命令面板入口（v2：侧栏顶部「搜索或跳转 Ctrl K」） -->
      <button
        v-if="!collapsed"
        type="button"
        class="sider-search"
        aria-label="搜索或跳转（Ctrl K）"
        @click="openPalette"
      >
        <SearchOutlined class="sider-search__icon" />
        <span class="sider-search__text">搜索或跳转</span>
        <kbd class="sider-search__kbd">Ctrl K</kbd>
      </button>

      <nav class="sidebar-nav" aria-label="管理菜单" tabindex="0" @keydown="onMenuKeydown">
        <a-menu theme="light" mode="inline" :selected-keys="[activeKey]">
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
          theme="light"
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

        <div class="admin-header__actions">
          <!-- 服务状态点（v2 常驻位）：数据来自 /admin/health 契约 #84 -->
          <RouterLink
            to="/health"
            class="service-pill"
            :class="`service-pill--${serviceState}`"
            :title="`${SERVICE_TEXT[serviceState]}（点击查看系统状态）`"
          >
            <span class="service-pill__dot" aria-hidden="true"></span>
            {{ SERVICE_TEXT[serviceState] }}
          </RouterLink>

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
        </div>
      </a-layout-header>

      <a-layout-content class="admin-content">
        <RouterView />
      </a-layout-content>
    </a-layout>

    <!-- 命令面板（⌘K）：菜单树驱动的跳转入口 -->
    <CommandPalette v-model:open="paletteOpen" :items="paletteItems" />
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

/* 品牌区：与 favicon 同一套视觉（品牌色方块 + 衬线 M）；
   v2 侧栏改浅色，主标随之为深色文字，副标降一级明度 */
.sider-logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  padding-inline: 20px;
  color: var(--admin-sidebar-text-strong);
  white-space: nowrap;
  overflow: hidden;
}

.sider-logo__mark {
  flex: none;
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--admin-brand);
  /* 内描边让方块边缘清晰，不靠加重投影 */
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.16);
  border-radius: 7px;
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 15px;
  font-weight: 600;
  line-height: 1;
  color: #fff;
  letter-spacing: 0;
}

.sider-logo__text {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.1px;
}

.sider-logo__sub {
  margin-left: 7px;
  font-size: 11px;
  font-weight: 400;
  letter-spacing: 0.6px;
  color: var(--admin-muted);
}

/* 命令面板入口：一体化搜索条 + 右侧 kbd 提示（v2 设计稿侧栏顶部） */
.sider-search {
  display: flex;
  align-items: center;
  gap: 8px;
  width: calc(100% - 24px);
  margin: 0 12px 12px;
  padding: 0 8px 0 10px;
  height: 34px;
  font-family: inherit;
  font-size: 13px;
  color: var(--admin-muted);
  text-align: left;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-sm);
  cursor: pointer;
  transition:
    border-color var(--admin-dur) var(--admin-ease),
    background-color var(--admin-dur) var(--admin-ease);
}

.sider-search:hover {
  border-color: color-mix(in srgb, var(--admin-brand) 40%, var(--admin-border));
}

.sider-search:focus-visible {
  outline: 2px solid var(--admin-brand);
  outline-offset: 1px;
}

.sider-search__icon {
  flex: none;
  font-size: 13px;
}

.sider-search__text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sider-search__kbd {
  flex: none;
  padding: 1px 5px;
  font-family: inherit;
  font-size: 10px;
  line-height: 16px;
  letter-spacing: 0.2px;
  color: var(--admin-muted);
  background: var(--admin-surface-2);
  border: 1px solid var(--admin-border);
  border-radius: 4px;
}

.admin-header {
  position: sticky;
  top: 0;
  z-index: 20;
  /* v2：64px（原 56px），与侧栏同色，靠 1px 描边分隔 */
  height: 64px;
  line-height: normal;
  padding: 0 24px;
  background: var(--admin-bg);
  border-bottom: 1px solid var(--admin-border);
  display: flex;
  align-items: center;
  gap: 16px;
}

.admin-header__crumbs {
  flex: 1;
  min-width: 0;
}

/* 顶栏右侧动作区：状态胶囊 + 图标按钮 + 头像，统一 16px 间距 */
.admin-header__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.admin-header__actions :deep(.ant-btn) {
  margin-right: 0;
}

/* 服务状态胶囊：色点 + 文案，点击进系统状态页 */
.service-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 10px;
  margin-right: 8px;
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
  border-radius: 999px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  transition: border-color var(--admin-dur) var(--admin-ease);
}

.service-pill:hover {
  border-color: color-mix(in srgb, var(--admin-brand) 40%, var(--admin-border));
}

.service-pill__dot {
  flex: none;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--admin-muted);
}

.service-pill--ok {
  color: var(--admin-muted);
}

.service-pill--ok .service-pill__dot {
  background: var(--admin-success);
  /* 呼吸圈暗示"活着"，不刺眼 */
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--admin-success) 16%, transparent);
}

.service-pill--degraded {
  color: var(--admin-warning-text);
  border-color: color-mix(in srgb, var(--admin-warning) 45%, var(--admin-border));
}

.service-pill--degraded .service-pill__dot {
  background: var(--admin-warning);
}

@media (prefers-reduced-motion: reduce) {
  .service-pill__dot {
    box-shadow: none;
  }
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
  background: var(--admin-sidebar-hover);
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
    color: var(--admin-text);
  }

  .admin-header__username {
    display: none;
  }

  /* 窄屏空间紧张：状态胶囊只留色点 */
  .service-pill {
    padding: 0 8px;
    margin-right: 4px;
    font-size: 0;
    gap: 0;
  }
}
</style>

<style>
/* 抽屉传送到 body，需全局样式：浅色底 + 菜单贴合边缘 */
.admin-drawer .ant-drawer-body {
  padding: 0;
  background: var(--admin-sidebar);
}

/* 侧栏/抽屉背景单一来源 --admin-sidebar（light #f5f7f7 / dark #000003），
   菜单透明继承（docs/09 §8.1 + v2 设计稿） */
.admin-sider.ant-layout-sider-light {
  background: var(--admin-sidebar);
  border-inline-end: 1px solid var(--admin-sidebar-border);
}

/* 折叠 trigger 与侧栏同色系，hover 轻提亮 */
.admin-sider .ant-layout-sider-trigger {
  background: transparent;
  color: var(--admin-muted);
  border-top: 1px solid var(--admin-sidebar-border);
  transition: background-color var(--admin-dur) var(--admin-ease);
}

.admin-sider .ant-layout-sider-trigger:hover {
  background: var(--admin-sidebar-hover);
  color: var(--admin-text);
}

.admin-sider .ant-menu,
.admin-drawer .ant-menu {
  background: transparent;
  border-inline-end: 0;
}

/* 菜单分组标题：统一 11px / 500 字距，组间靠同一档上下留白拉节奏。
   浅色侧栏上分组标题必须是次级灰，不能沿用原白色系。 */
.admin-sider .ant-menu-item-group-title,
.admin-drawer .ant-menu-item-group-title {
  padding: 14px 20px 6px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  letter-spacing: 0.6px;
  text-transform: uppercase;
  color: var(--admin-muted);
}

/* 折叠态隐藏分组标题，仅保留图标项 */
.admin-sider .ant-menu-inline-collapsed .ant-menu-item-group-title {
  display: none;
}

/* 菜单项：36px 行高 + 两端 8px 外距，形成圆角胶囊（v2 设计稿为圆角胶囊而非通栏）。
   选中态 = 深色胶囊 #1a1b20 + 反白字（实测值），暗色下改为品牌色文字。 */
.admin-sider .ant-menu-item,
.admin-drawer .ant-menu-item {
  height: 36px;
  line-height: 36px;
  margin-block: 2px;
  margin-inline: 8px;
  width: calc(100% - 16px);
  padding-inline: 10px;
  border-radius: var(--admin-radius-sm);
  color: var(--admin-sidebar-text);
  transition:
    color var(--admin-dur) var(--admin-ease),
    background-color var(--admin-dur) var(--admin-ease);
}

/* 图标盒宽与字号统一：不同图标字宽不同，固定 16px 盒保证文字起点对齐。 */
.admin-sider .ant-menu-item .anticon,
.admin-drawer .ant-menu-item .anticon {
  width: 16px;
  min-width: 16px;
  font-size: 15px;
  vertical-align: -0.2em;
}

.admin-sider .ant-menu-item:hover,
.admin-drawer .ant-menu-item:hover {
  color: var(--admin-sidebar-text-strong);
  background: var(--admin-sidebar-hover);
}

.admin-sider .ant-menu-item-selected,
.admin-drawer .ant-menu-item-selected {
  font-weight: 600;
  color: var(--admin-sidebar-active-text);
  background: var(--admin-sidebar-active-bg);
}

.admin-sider .ant-menu-item-selected:hover,
.admin-drawer .ant-menu-item-selected:hover {
  color: var(--admin-sidebar-active-text);
  background: var(--admin-sidebar-active-bg);
}

/* 选中项右侧不再叠加 AntD 默认指示条 */
.admin-sider .ant-menu-item-selected::after,
.admin-drawer .ant-menu-item-selected::after {
  display: none;
}

/* 折叠态：图标项收窄，胶囊退回居中方块 */
.admin-sider .ant-menu-inline-collapsed .ant-menu-item {
  width: calc(100% - 16px);
  padding-inline: calc(50% - 24px);
}

/* 键盘焦点：侧栏上必须有可见焦点环。
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
  color: var(--admin-muted);
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: color var(--admin-dur) var(--admin-ease);
}

.sider-trigger:hover {
  color: var(--admin-text);
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
</style>
