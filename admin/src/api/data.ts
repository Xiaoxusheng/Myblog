import http from './http'

export interface ImportSummary {
  posts?: number
  pages?: number
  categories?: number
  tags?: number
  links?: number
  comments?: number
  media?: number
}

export interface ImportResult {
  updated: number
  summary: ImportSummary
  conflicts: { type: string; value: string }[]
}

/** 导入（契约 #82）：dryRun=true 仅预览；strategy=skip(默认)|update */
export function importSite(file: File, opts: { dryRun: boolean; strategy: 'skip' | 'update' }): Promise<ImportResult> {
  const form = new FormData()
  form.append('file', file)
  return http.post(`/admin/import?dryRun=${opts.dryRun}&strategy=${opts.strategy}`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}
