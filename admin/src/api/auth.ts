import http from './http'
import type { LoginResult, ProfilePayload, User } from '@/types/api'

export function login(payload: {
  username: string
  password: string
  remember?: boolean
  captchaId: string
  captchaCode: string
}): Promise<LoginResult> {
  return http.post('/admin/auth/login', payload)
}

/** 图形验证码（契约 #86）：image 为 SVG 字符串，单次有效，5 分钟过期 */
export function getCaptcha(): Promise<{ captchaId: string; image: string }> {
  return http.get('/admin/auth/captcha')
}

export function getMe(): Promise<{ user: User }> {
  return http.get('/admin/auth/me')
}

/** 改密成功后旧 token 全部吊销（契约 #13），响应携带 24h 新 token 供无缝续期 */
export function updatePassword(payload: { oldPassword: string; newPassword: string }): Promise<{ token: string }> {
  return http.put('/admin/auth/password', payload)
}

export function updateProfile(payload: ProfilePayload): Promise<{ user: User }> {
  return http.put('/admin/auth/profile', payload)
}
