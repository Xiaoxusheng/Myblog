import axios, { AxiosError, type AxiosInstance, type AxiosRequestConfig } from 'axios'

/** 契约统一响应结构 */
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

/** 业务错误：code 来自契约错误码，message 为可展示文案 */
export class ApiError extends Error {
  code: number

  constructor(code: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

function friendlyHttpMessage(error: AxiosError): string {
  const status = error.response?.status
  if (error.code === 'ECONNABORTED') return '请求超时，请稍后重试'
  if (!error.response) return '网络连接失败，请检查网络后重试'
  if (status === 404) return '请求的资源不存在'
  if (status === 500) return '服务器开小差了，请稍后重试'
  return `请求失败(${status})`
}

const http: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000
})

// 响应拦截器：契约约定 HTTP 200 + body {code,message,data};code!==0 视为业务失败
http.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse<unknown> | undefined
    if (!body || typeof body.code !== 'number') {
      return Promise.reject(new ApiError(-1, '响应格式错误'))
    }
    if (body.code !== 0) {
      return Promise.reject(new ApiError(body.code, body.message || '请求失败'))
    }
    return response
  },
  (error: AxiosError) => {
    // 请求被取消（组件切换 / 连续请求）不算错误，原样抛出让调用方识别
    if (error.code === 'ERR_CANCELED') return Promise.reject(error)
    const status = error.response?.status ?? -1
    const body = error.response?.data as Partial<ApiResponse<unknown>> | undefined
    const message = body?.message || friendlyHttpMessage(error)
    return Promise.reject(new ApiError(status, message))
  }
)

/** 发起请求并解包契约响应中的 data */
async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await http.request<ApiResponse<T>>(config)
  return response.data.data
}

/** 带解包的请求方法：api.get<SiteData>('/site') 直接得到业务数据 */
export const api = {
  get: <T>(url: string, config?: AxiosRequestConfig): Promise<T> =>
    request<T>({ ...config, url, method: 'GET' }),
  post: <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> =>
    request<T>({ ...config, url, data, method: 'POST' })
}

/** 判断是否为请求取消（静默处理） */
export function isRequestCanceled(error: unknown): boolean {
  return (
    !!error &&
    typeof error === 'object' &&
    ((error as { code?: string }).code === 'ERR_CANCELED' ||
      (error as { name?: string }).name === 'CanceledError' ||
      (error as { name?: string }).name === 'AbortError')
  )
}
