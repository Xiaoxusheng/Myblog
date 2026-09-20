package handler

import (
	"strconv"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// settings 默认值（契约：key 缺省给默认值；seed 亦使用）
// 新增设置项时：此处加默认值 + getSettingsDTO 加读取分支 + AdminUpdateSettings 加写入分支，
// 三处必须同步，否则读写会不同步。
var settingsDefaults = model.SettingsDTO{
	SiteName:                 "My Blog",
	CommentEnabled:           true,
	PostPageSize:             10,
	AutoRedirectOnSlugChange: true,
}

// getSettingsDTO 读取全部设置行并映射为契约 Settings 对象
func getSettingsDTO(db *gorm.DB) model.SettingsDTO {
	s := settingsDefaults
	var rows []model.Setting
	if err := db.Find(&rows).Error; err != nil {
		return s
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Key] = r.Value
	}
	if v, ok := m["siteName"]; ok {
		s.SiteName = v
	}
	if v, ok := m["siteDescription"]; ok {
		s.SiteDescription = v
	}
	if v, ok := m["siteKeywords"]; ok {
		s.SiteKeywords = v
	}
	if v, ok := m["siteUrl"]; ok {
		s.SiteURL = v
	}
	if v, ok := m["logo"]; ok {
		s.Logo = v
	}
	if v, ok := m["notice"]; ok {
		s.Notice = v
	}
	if v, ok := m["icp"]; ok {
		s.ICP = v
	}
	if v, ok := m["footerText"]; ok {
		s.FooterText = v
	}
	if v, ok := m["commentEnabled"]; ok {
		s.CommentEnabled = v == "true"
	}
	if v, ok := m["postPageSize"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= common.MaxPageSize {
			s.PostPageSize = n
		}
	}
	if v, ok := m["autoRedirectOnSlugChange"]; ok {
		s.AutoRedirectOnSlugChange = v == "true"
	}
	return s
}

// autoRedirectOnSlugChangeEnabled 读取 slug 变更自动 301 开关（缺省开启）
func autoRedirectOnSlugChangeEnabled(db *gorm.DB) bool {
	var setting model.Setting
	// 反引号包住 key：MySQL 8 保留字，裸写会触发 Error 1064，
	// 而这里错误被吞掉会静默回退默认值 —— 属于「不报错但行为错」的隐蔽 bug
	if err := db.Where("`key` = ?", "autoRedirectOnSlugChange").First(&setting).Error; err != nil {
		return settingsDefaults.AutoRedirectOnSlugChange
	}
	return setting.Value == "true"
}

// upsertSetting 按 key 写入（存在则更新）
func upsertSetting(db *gorm.DB, key, value string) {
	var setting model.Setting
	err := db.Where(model.Setting{Key: key}).First(&setting).Error
	if err != nil {
		setting = model.Setting{Key: key, Value: value}
		_ = db.Create(&setting).Error
		return
	}
	_ = db.Model(&setting).Update("value", value).Error
}

// AdminGetSettings GET /api/v1/admin/settings
func AdminGetSettings(c *gin.Context) {
	common.OK(c, gin.H{"settings": getSettingsDTO(model.DB)})
}

// AdminUpdateSettings PUT /api/v1/admin/settings —— body 为结构化 Settings 对象
func AdminUpdateSettings(c *gin.Context) {
	var req model.SettingsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	// 夹取合法范围
	if req.PostPageSize < 1 {
		req.PostPageSize = 1
	}
	if req.PostPageSize > common.MaxPageSize {
		req.PostPageSize = common.MaxPageSize
	}

	db := model.DB
	upsertSetting(db, "siteName", req.SiteName)
	upsertSetting(db, "siteDescription", req.SiteDescription)
	upsertSetting(db, "siteKeywords", req.SiteKeywords)
	upsertSetting(db, "siteUrl", req.SiteURL)
	upsertSetting(db, "logo", req.Logo)
	upsertSetting(db, "notice", req.Notice)
	upsertSetting(db, "icp", req.ICP)
	upsertSetting(db, "footerText", req.FooterText)
	if req.CommentEnabled {
		upsertSetting(db, "commentEnabled", "true")
	} else {
		upsertSetting(db, "commentEnabled", "false")
	}
	upsertSetting(db, "postPageSize", strconv.Itoa(req.PostPageSize))
	if req.AutoRedirectOnSlugChange {
		upsertSetting(db, "autoRedirectOnSlugChange", "true")
	} else {
		upsertSetting(db, "autoRedirectOnSlugChange", "false")
	}

	writeAudit(c, "setting.update", "setting", "", "更新系统设置")
	common.OK(c, nil)
}
