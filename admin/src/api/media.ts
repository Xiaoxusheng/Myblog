import http from './http'
import type { PageResult, Settings, UploadItem } from '@/types/api'

/** 上传图片（multipart 字段名 file），仅图片、≤10MB；契约返回 {upload:Upload}，在此解包 */
export async function uploadImage(file: File): Promise<UploadItem> {
  const formData = new FormData()
  formData.append('file', file)
  // http 拦截器运行时已解包 body.data，这里按契约断言为 {upload}
  const res = (await http.post('/admin/uploads', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })) as unknown as { upload: UploadItem }
  return res.upload
}

export function getUploads(params: { page?: number; pageSize?: number }): Promise<PageResult<UploadItem>> {
  return http.get('/admin/uploads', { params })
}

export function deleteUpload(id: number): Promise<null> {
  return http.delete(`/admin/uploads/${id}`)
}

// ---------- 系统设置 ----------

export function getSettings(): Promise<{ settings: Settings }> {
  return http.get('/admin/settings')
}

export function updateSettings(payload: Settings): Promise<null> {
  return http.put('/admin/settings', payload)
}
