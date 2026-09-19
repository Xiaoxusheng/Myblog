package model

// SearchLog 站内搜索统计（契约 #85）：公开搜索接口按关键词记录，含无结果次数。

import "time"

type SearchLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Keyword     string    `gorm:"index;type:varchar(200)" json:"keyword"`
	ResultCount int       `json:"resultCount"` // 0=无结果
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
}
