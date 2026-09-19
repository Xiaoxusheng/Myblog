import { App } from 'ant-design-vue'

/**
 * 反馈统一入口:走 <a-app> 上下文,message/modal 主题(含暗色)自动跟随(docs/09 §4.3)
 * 新页面一律使用本入口;存量页面的静态 message 在被触碰时逐个迁移。
 */
export function useFeedback() {
  const { message, modal, notification } = App.useApp()
  return { message, modal, notification }
}
