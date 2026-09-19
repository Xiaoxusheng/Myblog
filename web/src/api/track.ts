/**
 * 原生访问埋点上报：POST /api/v1/track（无鉴权，恒 200，body {ok}）
 * 刻意不走 axios 实例 —— /track 返回 {ok} 而非契约 {code,message,data}，
 * 且埋点只发不收，失败必须静默，不影响任何用户可见行为。
 */

export interface TrackPayload {
  path: string
  postId?: number
  referer?: string
}

/**
 * 应用加载后只上报一次外部来源（document.referrer）：
 * 第一次 sendTrack 时附加，之后不再携带，避免站内跳转被计为来源。
 */
let referrerSent = false

/** 发送埋点：fire-and-forget，不等待响应，任何失败静默吞掉 */
export function sendTrack(payload: TrackPayload): void {
  const body: TrackPayload = { ...payload }
  if (!referrerSent) {
    referrerSent = true
    if (document.referrer) {
      body.referer = document.referrer
    }
  }
  try {
    // keepalive 保证页面即将卸载时请求仍可发出
    void fetch('/api/v1/track', {
      method: 'POST',
      keepalive: true,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }).catch(() => {
      /* 静默：埋点失败不影响浏览 */
    })
  } catch {
    /* 静默 */
  }
}
