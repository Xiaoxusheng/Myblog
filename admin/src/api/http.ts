import axios, { AxiosError } from 'axios'
import type { AxiosRequestConfig } from 'axios'
import { message } from 'ant-design-vue'
import router from '@/router'
import { TOKEN_KEY } from '@/constants/status'

/**
 * 请求级扩展配置：silent=true 时拦截器不弹全局错误 toast。
 * 用于「后台轮询 / 次级面板」这类失败不应打断用户的请求，由其调用方自行决定降级展示。
 * 注意：业务错误与 401 的跳转逻辑不受 silent 影响，仍会照常执行。
 *
 * 通过 module augmentation 合进 AxiosRequestConfig，这样 `http.get(url, { silent: true })`
 * 在 TS 下也能通过类型检查（若只导出独立 interface，实例方法签名不认这个字段）。
 */
declare module 'axios' {
  export interface AxiosRequestConfig {
    silent?: boolean
  }
}

export type RequestConfig = AxiosRequestConfig

/** 业务错误（携带契约错误码） */
export class ApiError extends Error {
  code: number

  constructor(code: number, messageText: string) {
    super(messageText)
    this.name = 'ApiError'
    this.code = code
  }
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

/** 401 / 10002 统一处理：清 token 跳登录 */
function handleUnauthorized(): void {
  clearToken()
  const current = router.currentRoute.value
  if (current.path === '/login') return
  message.warning('登录已过期，请重新登录')
  void router.push({ path: '/login', query: { redirect: current.fullPath } })
}

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 20000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const body = response.data
    const silent = response.config.silent === true
    // 契约约定：所有业务响应 HTTP 200，body 为 {code, message, data}
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) {
        return body.data
      }
      if (body.code === 10002) {
        handleUnauthorized()
        return Promise.reject(new ApiError(body.code, body.message || '未认证'))
      }
      if (!silent) {
        message.error(body.message || '请求失败')
      }
      return Promise.reject(new ApiError(body.code, body.message || '请求失败'))
    }
    return body
  },
  (error: AxiosError) => {
    const silent = error.config?.silent === true
    const status = error.response?.status
    const body = error.response?.data as { code?: number; message?: string } | undefined
    if (status === 401 || body?.code === 10002) {
      handleUnauthorized()
      return Promise.reject(new ApiError(10002, '未认证'))
    }
    const text =
      body?.message ||
      (status === 500
        ? '服务器错误，请稍后重试'
        : status
          ? `请求失败（HTTP ${status}）`
          : '网络连接失败，请检查网络或后端服务')
    if (!silent) {
      message.error(text)
    }
    return Promise.reject(new ApiError(body?.code ?? -1, text))
  },
)

export default http
