import http from './http'
import type { BackupItem } from '@/types/api'
import { TOKEN_KEY } from '@/constants/status'

export function getBackups(): Promise<{ list: BackupItem[] }> {
  return http.get('/admin/backups')
}

export function createBackup(type: 'database' | 'full'): Promise<{ item: BackupItem }> {
  return http.post('/admin/backups', { type })
}

export function deleteBackup(name: string): Promise<null> {
  return http.delete(`/admin/backups/${encodeURIComponent(name)}`)
}

/** blob 下载备份文件（需 Bearer 头，故不走静态链接） */
export async function downloadBackup(name: string): Promise<void> {
  const resp = await fetch(`/api/v1/admin/backups/${encodeURIComponent(name)}/download`, {
    headers: { Authorization: `Bearer ${localStorage.getItem(TOKEN_KEY) ?? ''}` },
  })
  if (!resp.ok) throw new Error(`下载失败（${resp.status}）`)
  const blob = await resp.blob()
  triggerDownload(blob, name)
}

/** 全站导出 zip（契约 #81） */
export async function exportSite(): Promise<void> {
  const resp = await fetch('/api/v1/admin/export', {
    headers: { Authorization: `Bearer ${localStorage.getItem(TOKEN_KEY) ?? ''}` },
  })
  if (!resp.ok) throw new Error(`导出失败（${resp.status}）`)
  const blob = await resp.blob()
  triggerDownload(blob, `myblog-export-${new Date().toISOString().slice(0, 10)}.zip`)
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
