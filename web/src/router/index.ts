import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { applyDocumentTitle } from '@/utils/title'
import { setSeo } from '@/utils/seo'
import { sendTrack } from '@/api/track'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: { title: '首页' }
  },
  {
    path: '/post/:slug',
    name: 'post-detail',
    component: () => import('@/views/PostDetailView.vue'),
    meta: { title: '文章' }
  },
  {
    path: '/archives',
    name: 'archives',
    component: () => import('@/views/ArchivesView.vue'),
    meta: { title: '归档' }
  },
  {
    path: '/categories',
    name: 'categories',
    component: () => import('@/views/CategoriesView.vue'),
    meta: { title: '分类' }
  },
  {
    path: '/category/:slug',
    name: 'category-posts',
    component: () => import('@/views/CategoryPostsView.vue'),
    meta: { title: '分类' }
  },
  {
    path: '/tags',
    name: 'tags',
    component: () => import('@/views/TagsView.vue'),
    meta: { title: '标签' }
  },
  {
    path: '/tag/:slug',
    name: 'tag-posts',
    component: () => import('@/views/TagPostsView.vue'),
    meta: { title: '标签' }
  },
  {
    path: '/series',
    name: 'series',
    component: () => import('@/views/SeriesView.vue'),
    meta: { title: '专题' }
  },
  {
    path: '/series/:slug',
    name: 'series-detail',
    component: () => import('@/views/SeriesDetailView.vue'),
    meta: { title: '专题' }
  },
  {
    path: '/search',
    name: 'search',
    component: () => import('@/views/SearchView.vue'),
    meta: { title: '搜索' }
  },
  {
    path: '/links',
    name: 'links',
    component: () => import('@/views/LinksView.vue'),
    meta: { title: '友情链接' }
  },
  {
    path: '/timeline',
    name: 'timeline',
    component: () => import('@/views/TimelineView.vue'),
    meta: { title: '时间线' }
  },
  {
    path: '/changelog',
    name: 'changelog',
    component: () => import('@/views/ChangelogView.vue'),
    meta: { title: '更新日志' }
  },
  {
    path: '/page/:slug',
    name: 'custom-page',
    component: () => import('@/views/PageView.vue'),
    meta: { title: '页面' }
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: '页面不存在' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  // 切换页面回到顶部;浏览器前进/后退保留位置
  scrollBehavior(_to, _from, savedPosition) {
    if (savedPosition) return savedPosition
    return { top: 0 }
  }
})

router.afterEach((to) => {
  applyDocumentTitle(to.meta.title)
  // 默认清空 SEO meta，具体页面（首页/文章/自定义页）加载后自行覆盖
  setSeo({})
  // 访问埋点：路由切换上报一次；文章详情页跳过（由 PostDetailView 数据加载
  // 成功后带 postId 上报），避免无 postId 的重复计数
  if (to.name !== 'post-detail') {
    sendTrack({ path: to.path })
  }
})

export default router
