import axios from 'axios'
import { TOKEN_KEY } from '@/constants/status'
import type { AdminPostItem, PagePayload, PostPayload } from '@/types/api'

/**
 * 静默请求失败时抛出的结构化错误。
 * code 用于上层识别业务语义（如 10005 并发编辑冲突 → 弹出冲突处理）。
 */
export class SilentRequestError extends Error {
  readonly code: number
  constructor(code: number, message: string) {
    super(message)
    this.name = 'SilentRequestError'
    this.code = code
  }
}

/** 内容冲突（文章已在其他窗口被修改）—— 与后端 common.CodeConflict 对齐 */
export const CODE_CONFLICT = 10005

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

/** 从信箱响应中取 data；业务码非 0 时抛 SilentRequestError */
function unwrap<T>(data: unknown): T {
  const body = data as { code?: number; message?: string; data?: T } | null
  if (body && typeof body === 'object' && body.code === 0) {
    return body.data as T
  }
  throw new SilentRequestError(body?.code ?? -1, body?.message || '自动保存失败')
}

/**
 * 静默更新文章（auto 自动保存）。
 * 成功 resolve 服务器定稿后的文章（含新的 updatedAt，供基线回填）；失败 reject
 * SilentRequestError（不弹任何全局提示，由调用方决定 UI 反馈）。
 */
export function silentUpdatePost(id: number, payload: PostPayload): Promise<AdminPostItem> {
  return instance
    .put(`/admin/posts/${id}`, payload)
    .then((response) => unwrap<{ post: AdminPostItem }>(response.data).post)
}

/** 静默更新自定义页面（自动保存）：语义同 silentUpdatePost */
export function silentUpdatePage(id: number, payload: PagePayload): Promise<void> {
  return instance.put(`/admin/pages/${id}`, payload).then((response) => {
    unwrap<unknown>(response.data)
  })
}
