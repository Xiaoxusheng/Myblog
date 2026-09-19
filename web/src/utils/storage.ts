/** localStorage 封装：点赞记录、评论者信息记忆、主题 key 常量 */

export const THEME_STORAGE_KEY = 'myblog-theme'

const LIKED_KEY = 'myblog-liked-posts'
const PROFILE_KEY = 'myblog-comment-profile'

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
