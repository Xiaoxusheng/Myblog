import http from './http'
import type { LoginResult, ProfilePayload, User } from '@/types/api'

export function login(payload: {
  username: string
  password: string
  remember?: boolean
}): Promise<LoginResult> {
  return http.post('/admin/auth/login', payload)
}

export function getMe(): Promise<{ user: User }> {
  return http.get('/admin/auth/me')
}

export function updatePassword(payload: { oldPassword: string; newPassword: string }): Promise<null> {
  return http.put('/admin/auth/password', payload)
}

export function updateProfile(payload: ProfilePayload): Promise<{ user: User }> {
  return http.put('/admin/auth/profile', payload)
}
