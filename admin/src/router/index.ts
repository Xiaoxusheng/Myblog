import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { title: '登录', public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/AdminLayout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: '仪表盘' },
        },
        {
          path: 'analytics',
          name: 'analytics',
          component: () => import('@/views/AnalyticsView.vue'),
          meta: { title: '访问分析' },
        },
        {
          path: 'posts',
          name: 'posts',
          component: () => import('@/views/PostListView.vue'),
          meta: { title: '文章管理' },
        },
        {
          path: 'posts/drafts',
          name: 'post-drafts',
          component: () => import('@/views/DraftsView.vue'),
          meta: { title: '草稿工作区', parentTitle: '文章管理', parentPath: '/posts', activeMenu: '/posts' },
        },
        {
          path: 'posts/edit',
          name: 'post-create',
          component: () => import('@/views/PostEditView.vue'),
          meta: { title: '新建文章', parentTitle: '文章管理', parentPath: '/posts', activeMenu: '/posts' },
        },
        {
          path: 'posts/edit/:id',
          name: 'post-edit',
          component: () => import('@/views/PostEditView.vue'),
          meta: { title: '编辑文章', parentTitle: '文章管理', parentPath: '/posts', activeMenu: '/posts' },
        },
        {
          path: 'comments',
          name: 'comments',
          component: () => import('@/views/CommentView.vue'),
          meta: { title: '评论管理' },
        },
        {
          path: 'categories',
          name: 'categories',
          component: () => import('@/views/CategoryView.vue'),
          meta: { title: '分类管理' },
        },
        {
          path: 'tags',
          name: 'tags',
          component: () => import('@/views/TagView.vue'),
          meta: { title: '标签管理' },
        },
        {
          path: 'pages',
          name: 'pages',
          component: () => import('@/views/PageListView.vue'),
          meta: { title: '页面管理' },
        },
        {
          path: 'pages/edit',
          name: 'page-create',
          component: () => import('@/views/PageEditView.vue'),
          meta: { title: '新建页面', parentTitle: '页面管理', parentPath: '/pages', activeMenu: '/pages' },
        },
        {
          path: 'pages/edit/:id',
          name: 'page-edit',
          component: () => import('@/views/PageEditView.vue'),
          meta: { title: '编辑页面', parentTitle: '页面管理', parentPath: '/pages', activeMenu: '/pages' },
        },
        {
          path: 'series',
          name: 'series',
          component: () => import('@/views/SeriesView.vue'),
          meta: { title: '专题管理' },
        },
        {
          path: 'timeline',
          name: 'timeline',
          component: () => import('@/views/TimelineView.vue'),
          meta: { title: '时间线管理' },
        },
        {
          path: 'changelogs',
          name: 'changelogs',
          component: () => import('@/views/ChangelogView.vue'),
          meta: { title: '版本记录' },
        },
        {
          path: 'redirects',
          name: 'redirects',
          component: () => import('@/views/RedirectView.vue'),
          meta: { title: '重定向管理' },
        },
        {
          path: 'comment-blacklist',
          name: 'comment-blacklist',
          component: () => import('@/views/CommentBlacklistView.vue'),
          meta: { title: '评论防护' },
        },
        {
          path: 'backups',
          name: 'backups',
          component: () => import('@/views/BackupView.vue'),
          meta: { title: '备份' },
        },
        {
          path: 'import-export',
          name: 'import-export',
          component: () => import('@/views/ImportExportView.vue'),
          meta: { title: '导入导出' },
        },
        {
          path: 'audit-logs',
          name: 'audit-logs',
          component: () => import('@/views/AuditLogView.vue'),
          meta: { title: '操作日志' },
        },
        {
          path: 'health',
          name: 'health',
          component: () => import('@/views/HealthView.vue'),
          meta: { title: '系统状态' },
        },
        {
          path: 'links',
          name: 'links',
          component: () => import('@/views/LinkView.vue'),
          meta: { title: '友链管理' },
        },
        {
          path: 'media',
          name: 'media',
          component: () => import('@/views/MediaView.vue'),
          meta: { title: '媒体库' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
          meta: { title: '系统设置' },
        },
        {
          path: 'profile',
          name: 'profile',
          component: () => import('@/views/ProfileView.vue'),
          meta: { title: '个人资料' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard',
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.public) {
    // 已登录访问登录页 → 回仪表盘
    if (auth.isLoggedIn && to.path === '/login') {
      return { path: '/dashboard' }
    }
    return true
  }
  if (!auth.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  return true
})

router.afterEach((to) => {
  const title = to.meta.title
  document.title = title ? `${title} - MyBlog 管理后台` : 'MyBlog 管理后台'
})

export default router
