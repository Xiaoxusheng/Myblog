import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    /** 页面标题（用于面包屑与 document.title） */
    title?: string
    /** 无需登录即可访问（登录页） */
    public?: boolean
    /** 面包屑上级标题 */
    parentTitle?: string
    /** 面包屑上级路径 */
    parentPath?: string
    /** 列表页激活的菜单 key（编辑页与列表页共用高亮） */
    activeMenu?: string
    /** 是否在侧边菜单中隐藏（编辑页） */
    hidden?: boolean
  }
}

export {}
