/** localStorage 封装：点赞记录、评论者信息记忆、阅读位置、主题 key 常量 */

export const THEME_STORAGE_KEY = 'myblog-theme'

const LIKED_KEY = 'myblog-liked-posts'
const PROFILE_KEY = 'myblog-comment-profile'
const READING_KEY = 'myblog-reading-positions'

/** 阅读位置保留条数与时效：防止无限增长 */
const READING_MAX_ENTRIES = 100
const READING_TTL_MS = 90 * 24 * 60 * 60 * 1000

export interface CommentProfile {
  nickname: string
  email: string
  website: string
}

export function getLikedPostIds(): number[] {
  try {
    const raw = localStorage.getItem(LIKED_KEY)
    const parsed: unknown = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed.filter((x): x is number => typeof x === 'number') : []
  } catch {
    return []
  }
}

export function isPostLiked(postId: number): boolean {
  return getLikedPostIds().includes(postId)
}

export function markPostLiked(postId: number): void {
  const ids = getLikedPostIds()
  if (!ids.includes(postId)) {
    ids.push(postId)
    try {
      localStorage.setItem(LIKED_KEY, JSON.stringify(ids))
    } catch {
      /* 存储不可用时静默降级：仅当前会话生效 */
    }
  }
}

export function loadCommentProfile(): CommentProfile {
  try {
    const raw = localStorage.getItem(PROFILE_KEY)
    const parsed: unknown = raw ? JSON.parse(raw) : {}
    if (parsed && typeof parsed === 'object') {
      const p = parsed as Record<string, unknown>
      return {
        nickname: typeof p.nickname === 'string' ? p.nickname : '',
        email: typeof p.email === 'string' ? p.email : '',
        website: typeof p.website === 'string' ? p.website : ''
      }
    }
  } catch {
    /* ignore */
  }
  return { nickname: '', email: '', website: '' }
}

export function saveCommentProfile(profile: CommentProfile): void {
  try {
    localStorage.setItem(PROFILE_KEY, JSON.stringify(profile))
  } catch {
    /* ignore */
  }
}

/** 单篇文章的阅读位置：p 为整页滚动比例（0-1，相对比例可容内容与视口变化），d 表示已读完 */
export interface ReadingPosition {
  p: number
  d: boolean
  t: number
}

function readAllPositions(): Record<string, ReadingPosition> {
  try {
    const raw = localStorage.getItem(READING_KEY)
    const parsed: unknown = raw ? JSON.parse(raw) : {}
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}
    const out: Record<string, ReadingPosition> = {}
    for (const [slug, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (!value || typeof value !== 'object') continue
      const v = value as Record<string, unknown>
      if (typeof v.p !== 'number' || typeof v.t !== 'number') continue
      out[slug] = { p: v.p, d: v.d === true, t: v.t }
    }
    return out
  } catch {
    return {}
  }
}

export function getReadingPosition(slug: string): ReadingPosition | null {
  if (!slug) return null
  const all = readAllPositions()
  const hit = all[slug]
  return hit ?? null
}

/** 记录阅读位置。写入失败（隐私模式/配额满）静默降级，不影响阅读 */
export function saveReadingPosition(slug: string, progress: number, done: boolean): void {
  if (!slug) return
  const all = readAllPositions()
  const now = Date.now()
  all[slug] = { p: Math.min(1, Math.max(0, progress)), d: done, t: now }

  // 清理策略：过期条目剔除；仍超上限时按最后阅读时间淘汰最旧的
  let entries = Object.entries(all).filter(([, v]) => now - v.t < READING_TTL_MS)
  if (entries.length > READING_MAX_ENTRIES) {
    entries = entries.sort((a, b) => b[1].t - a[1].t).slice(0, READING_MAX_ENTRIES)
  }
  try {
    localStorage.setItem(READING_KEY, JSON.stringify(Object.fromEntries(entries)))
  } catch {
    /* ignore */
  }
}
