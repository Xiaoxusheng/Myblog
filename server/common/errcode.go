package common

// 业务错误码（唯一来源：docs/contracts/api.md）
const (
	CodeOK            = 0
	CodeParamError    = 10001 // 参数错误
	CodeUnauthorized  = 10002 // 未认证
	CodeForbidden     = 10003 // 无权限
	CodeNotFound      = 10004 // 资源不存在
	CodeConflict      = 10005 // 内容冲突（并发编辑保护）
	CodeLoginFailed   = 20001 // 用户名或密码错误
	CodeCommentClosed = 20002 // 评论已关闭
	CodeTooFrequent   = 20003 // 操作过于频繁
)
