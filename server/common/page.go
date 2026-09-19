package common

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// MaxPageSize 分页上限（契约：pageSize ≤ 50）
const MaxPageSize = 50

// PageQuery 分页参数
type PageQuery struct {
	Page     int
	PageSize int
}

// ParsePage 解析 query 中的 page / pageSize。
// defaultSize 为未传 pageSize 时的默认值；无论默认还是显式传入，均被夹取到 [1, MaxPageSize]。
func ParsePage(c *gin.Context, defaultSize int) PageQuery {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.Query("pageSize"))
	if size <= 0 {
		size = defaultSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return PageQuery{Page: page, PageSize: size}
}

// Offset 计算 SQL 偏移量
func (p PageQuery) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Data 组装统一分页响应 data：{"list":...,"total":...,"page":...,"pageSize":...}
func (p PageQuery) Data(list any, total int64) gin.H {
	return gin.H{
		"list":     list,
		"total":    total,
		"page":     p.Page,
		"pageSize": p.PageSize,
	}
}
