import axios from 'axios'
import { TOKEN_KEY } from '@/constants/status'
import type { PostPayload } from '@/types/api'

/**
 * 服务器自动保存专用「静默」请求通道。
 *
 * 为什么不复用 api/http.ts：其响应拦截器对失败请求统一 message.error 弹窗，
 * 而自动保存失败只允许更新编辑器底部状态条（3s 防抖下弹窗会反复刷屏）。
 * 这里仅复刻最小必要逻辑：baseURL、Bearer token、{code, message, data} 信箱解包。
 * 注意：若 http.ts 的 baseURL / 信箱结构调整，需同步此处；
 * 后续建议给 http.ts 增加 silent 请求配置后合并回唯一通道。
 */
const instance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

instance.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

/** 静默更新文章（auto 自动保存）：成功 resolve；失败 reject，不弹任何全局提示 */
export function silentUpdatePost(id: number, payload: PostPayload): Promise<void> {
  return instance.put(`/admin/posts/${id}`, payload).then((response) => {
    const body = response.data as { code?: number } | null
    if (body && typeof body === 'object' && body.code === 0) {
      return undefined
    }
    throw new Error('自动保存失败')
  })
}
