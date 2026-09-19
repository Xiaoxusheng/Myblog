import { api } from './http'
import type {
  CommentCreated,
  CommentPayload,
  CommentPublic,
  Paged,
  PostDetailData,
  PostListParams,
  PostSummary
} from '@/types'

/** GET /posts:文章分页（仅已发布，置顶优先） */
export function fetchPosts(
  params: PostListParams,
  signal?: AbortSignal
): Promise<Paged<PostSummary>> {
  return api.get<Paged<PostSummary>>('/posts', { params, signal })
}

/** GET /posts/:slug:详情 + 上一篇/下一篇 + 相关文章（同时浏览量 +1） */
export function fetchPostDetail(slug: string, signal?: AbortSignal): Promise<PostDetailData> {
  return api.get<PostDetailData>(`/posts/${encodeURIComponent(slug)}`, { signal })
}

/** GET /posts/:slug/comments:已通过评论，两级树 */
export function fetchComments(slug: string, signal?: AbortSignal): Promise<CommentPublic[]> {
  return api
    .get<{ list: CommentPublic[] }>(`/posts/${encodeURIComponent(slug)}/comments`, { signal })
    .then((d) => d?.list ?? [])
}

/** POST /posts/:slug/comments:游客评论（新评论进入待审核），契约返回 {comment:{...}} */
export function createComment(slug: string, payload: CommentPayload): Promise<CommentCreated> {
  return api
    .post<{ comment: CommentCreated }>(`/posts/${encodeURIComponent(slug)}/comments`, payload)
    .then((d) => d.comment)
}

/** POST /posts/:slug/like:点赞，返回最新 likeCount */
export function likePost(slug: string): Promise<number> {
  return api
    .post<{ likeCount: number }>(`/posts/${encodeURIComponent(slug)}/like`)
    .then((d) => d.likeCount)
}
