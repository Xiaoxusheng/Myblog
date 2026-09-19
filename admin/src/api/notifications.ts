import http from './http'
import type { NotificationItem } from '@/types/api'

/** 最新 10 条通知 + 未读数（契约 #73） */
export function getNotifications(): Promise<{ list: NotificationItem[]; unreadCount: number }> {
  return http.get('/admin/notifications')
}

export function readAllNotifications(): Promise<null> {
  return http.put('/admin/notifications/read-all')
}

export function readNotification(id: number): Promise<null> {
  return http.put(`/admin/notifications/${id}/read`)
}
