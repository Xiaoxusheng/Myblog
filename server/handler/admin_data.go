package handler

// 全站导出/导入（契约 #81/#82）。
// 安全：zip 炸弹防护（条目数/解压总量上限）、媒体按包内相对路径落盘且拒绝穿越、导入不覆盖默认策略。

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

const (
	importMaxFileBytes = 50 << 20  // 上传 zip 上限 50MB
	importMaxEntries   = 5000      // zip 条目上限
	importMaxUnpack    = 200 << 20 // 解压总量上限 200MB
)

// ---------- 导出（#81） ----------

// AdminExport GET /api/v1/admin/export —— 全站 zip（JSON 数据 + media/）
func AdminExport(c *gin.Context) {
	posts := exportPosts()
	pages := exportPages()
	categories := exportCategories()
	tags := exportTags()
	links := exportLinks()
	comments := exportComments()
	settings := getSettingsDTO(model.DB)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=myblog-export-%s.zip", time.Now().Format("20060102-150405")))
	zw := zip.NewWriter(c.Writer)
	writeJSONEntry := func(name string, v any) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(v)
	}
	for _, e := range []struct {
		name string
		v    any
	}{
		{"posts.json", posts}, {"pages.json", pages}, {"categories.json", categories},
		{"tags.json", tags}, {"links.json", links}, {"comments.json", comments},
		{"settings.json", settings},
	} {
		if err := writeJSONEntry(e.name, e.v); err != nil {
			abortExport(zw, c, err)
			return
		}
	}
	// media/：uploads 递归打包（含 YYYYMM/ 子目录，保留相对路径）
	if err := addUploadsToZip(zw); err != nil {
		abortExport(zw, c, err)
		return
	}
	if err := zw.Close(); err != nil {
		abortExport(zw, c, err)
		return
	}
	writeAudit(c, "system.export", "system", "", "全站导出")
}

func abortExport(zw *zip.Writer, c *gin.Context, err error) {
	_ = zw.Close()
	if !c.Writer.Written() {
		common.ServerError(c, err)
	}
}

func copyInto(w io.Writer, src string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// exportPosts 导出文章（含标签名，供导入重建关联）
func exportPosts() []gin.H {
	var posts []model.Post
	model.DB.Preload("Tags").Preload("Category").Order("id ASC").Find(&posts)
	out := make([]gin.H, 0, len(posts))
	for _, p := range posts {
		tagNames := make([]string, 0, len(p.Tags))
		for _, t := range p.Tags {
			tagNames = append(tagNames, t.Name)
		}
		item := gin.H{
			"id": p.ID, "title": p.Title, "slug": p.Slug, "summary": p.Summary,
			"content": p.Content, "cover": p.Cover, "categoryId": p.CategoryID,
			"seriesId": p.SeriesID, "seriesSort": p.SeriesSort, "tags": tagNames,
			"status": p.Status, "isTop": p.IsTop, "createdAt": p.CreatedAt,
		}
		if p.Category != nil {
			item["categoryName"] = p.Category.Name
		}
		if p.PublishedAt != nil {
			item["publishedAt"] = p.PublishedAt
		}
		out = append(out, item)
	}
	return out
}

func exportPages() []model.Page {
	var pages []model.Page
	model.DB.Order("id ASC").Find(&pages)
	return pages
}

func exportCategories() []model.Category {
	var list []model.Category
	model.DB.Order("id ASC").Find(&list)
	return list
}

func exportTags() []model.Tag {
	var list []model.Tag
	model.DB.Order("id ASC").Find(&list)
	return list
}

func exportLinks() []model.Link {
	var list []model.Link
	model.DB.Order("id ASC").Find(&list)
	return list
}

func exportComments() []gin.H {
	var comments []model.Comment
	model.DB.Order("id ASC").Find(&comments)
	// 关联文章 slug，导入时按 slug 回链
	var posts []model.Post
	model.DB.Select("id", "slug").Find(&posts)
	slugOf := map[uint]string{}
	for _, p := range posts {
		slugOf[p.ID] = p.Slug
	}
	out := make([]gin.H, 0, len(comments))
	for _, cm := range comments {
		out = append(out, gin.H{
			"postSlug": slugOf[cm.PostID], "parentId": cm.ParentID,
			"nickname": cm.Nickname, "email": cm.Email, "website": cm.Website,
			"content": cm.Content, "status": cm.Status, "isAdmin": cm.IsAdmin,
			"createdAt": cm.CreatedAt,
		})
	}
	return out
}

// ---------- 导入（#82） ----------

// AdminImport POST /api/v1/admin/import?dryRun=true&strategy=skip|update
func AdminImport(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.Fail(c, common.CodeParamError, "缺少 file 字段（zip 包）")
		return
	}
	if fileHeader.Size > importMaxFileBytes {
		common.Fail(c, common.CodeParamError, "包大小超过 50MB 上限")
		return
	}
	dryRun := c.Query("dryRun") == "true"
	strategy := c.DefaultQuery("strategy", "skip")
	if strategy != "skip" && strategy != "update" {
		common.Fail(c, common.CodeParamError, "strategy 仅支持 skip / update")
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		common.ServerError(c, err)
		return
	}
	defer f.Close()
	zr, err := zip.NewReader(f, fileHeader.Size)
	if err != nil {
		common.Fail(c, common.CodeParamError, "不是合法的 zip 包")
		return
	}
	if len(zr.File) > importMaxEntries {
		common.Fail(c, common.CodeParamError, "zip 条目数超上限")
		return
	}
	// zip 炸弹防护：解压总量
	var totalUnpack uint64
	for _, zf := range zr.File {
		totalUnpack += zf.UncompressedSize64
		if totalUnpack > importMaxUnpack {
			common.Fail(c, common.CodeParamError, "解压总量超过 200MB 上限")
			return
		}
	}

	var importBundle struct {
		Posts      []importPost     `json:"posts"`
		Pages      []model.Page     `json:"pages"`
		Categories []model.Category `json:"categories"`
		Tags       []model.Tag      `json:"tags"`
		Links      []model.Link     `json:"links"`
		Comments   []importComment  `json:"comments"`
	}
	summary := map[string]int{"posts": 0, "pages": 0, "categories": 0, "tags": 0, "links": 0, "comments": 0, "media": 0}
	conflicts := make([]gin.H, 0)

	for _, zf := range zr.File {
		switch {
		case strings.HasSuffix(zf.Name, ".json") && !strings.Contains(zf.Name, "media/"):
			var target any
			switch zf.Name {
			case "posts.json":
				target = &importBundle.Posts
			case "pages.json":
				target = &importBundle.Pages
			case "categories.json":
				target = &importBundle.Categories
			case "tags.json":
				target = &importBundle.Tags
			case "links.json":
				target = &importBundle.Links
			case "comments.json":
				target = &importBundle.Comments
			default:
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				common.ServerError(c, err)
				return
			}
			err = json.NewDecoder(rc).Decode(target)
			rc.Close()
			if err != nil {
				common.Fail(c, common.CodeParamError, fmt.Sprintf("%s 解析失败：%v", zf.Name, err))
				return
			}
		case strings.HasPrefix(zf.Name, "media/"):
			// 媒体按包内相对路径落 uploads（含子目录，兼容旧包根下文件）；目录条目与穿越路径跳过
			if strings.HasSuffix(zf.Name, "/") {
				continue
			}
			rel, ok := importMediaRelPath(zf.Name)
			if !ok {
				continue
			}
			if dryRun {
				summary["media"]++
				continue
			}
			dest := filepath.Join(appCfg.UploadDir, filepath.FromSlash(rel))
			if _, err := os.Stat(dest); err == nil {
				continue // 同名媒体跳过
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				common.ServerError(c, err)
				return
			}
			if err := extractFile(zf, dest); err != nil {
				common.ServerError(c, err)
				return
			}
			summary["media"]++
		}
	}

	// 预统计冲突
	for _, ip := range importBundle.Posts {
		if slugTaken(model.DB, ip.Slug, 0) {
			conflicts = append(conflicts, gin.H{"type": "post.slug", "value": ip.Slug})
		}
	}
	summary["posts"] = len(importBundle.Posts)
	summary["pages"] = len(importBundle.Pages)
	summary["categories"] = len(importBundle.Categories)
	summary["tags"] = len(importBundle.Tags)
	summary["links"] = len(importBundle.Links)
	summary["comments"] = len(importBundle.Comments)

	if dryRun {
		common.OK(c, gin.H{"summary": summary, "conflicts": conflicts})
		return
	}

	updated := int64(0)
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		// 分类/标签按名称 upsert（tag 复用既有 upsertTagsByName）
		var n int64
		for _, cat := range importBundle.Categories {
			tx.Model(&model.Category{}).Where("name = ?", cat.Name).Count(&n)
			if n > 0 {
				if strategy == "update" {
					tx.Model(&model.Category{}).Where("name = ?", cat.Name).
						Updates(map[string]any{"slug": cat.Slug, "description": cat.Description})
				}
				continue
			}
			if err := tx.Create(&model.Category{Name: cat.Name, Slug: cat.Slug, Description: cat.Description}).Error; err != nil {
				return err
			}
		}
		for _, tag := range importBundle.Tags {
			if _, err := upsertOneTag(tx, tag.Name); err != nil {
				return err
			}
		}
		for _, p := range importBundle.Posts {
			var existing model.Post
			found := tx.Where("slug = ?", p.Slug).First(&existing).Error == nil
			if found && strategy == "skip" {
				continue
			}
			categoryID := resolveCategoryID(tx, p.CategoryName)
			if found && strategy == "update" {
				updates := map[string]any{
					"title": p.Title, "summary": p.Summary, "content": p.Content,
					"cover": p.Cover, "status": p.Status, "is_top": p.IsTop,
				}
				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
				updated++
				continue
			}
			np := model.Post{
				Title: p.Title, Slug: p.Slug, Summary: p.Summary, Content: p.Content,
				Cover: p.Cover, CategoryID: categoryID, SeriesID: p.SeriesID,
				SeriesSort: p.SeriesSort, Status: p.Status, IsTop: p.IsTop,
			}
			if p.PublishedAt != nil {
				np.PublishedAt = p.PublishedAt
			}
			if err := tx.Create(&np).Error; err != nil {
				return err
			}
			tags, err := upsertTagsByName(tx, p.Tags)
			if err != nil {
				return err
			}
			if err := tx.Model(&np).Association("Tags").Replace(tags); err != nil {
				return err
			}
		}
		for _, pg := range importBundle.Pages {
			var n int64
			tx.Model(&model.Page{}).Where("slug = ?", pg.Slug).Count(&n)
			if n > 0 {
				if strategy != "update" {
					continue
				}
				tx.Model(&model.Page{}).Where("slug = ?", pg.Slug).
					Updates(map[string]any{"title": pg.Title, "content": pg.Content, "status": pg.Status})
				updated++
				continue
			}
			if err := tx.Create(&model.Page{Title: pg.Title, Slug: pg.Slug, Content: pg.Content, Status: pg.Status}).Error; err != nil {
				return err
			}
		}
		for _, l := range importBundle.Links {
			var n int64
			tx.Model(&model.Link{}).Where("name = ?", l.Name).Count(&n)
			if n > 0 && strategy != "update" {
				continue
			}
			if n > 0 {
				tx.Model(&model.Link{}).Where("name = ?", l.Name).
					Updates(map[string]any{"url": l.URL, "logo": l.Logo, "description": l.Description, "visible": l.Visible, "sort": l.Sort})
				updated++
				continue
			}
			if err := tx.Create(&model.Link{Name: l.Name, URL: l.URL, Logo: l.Logo, Description: l.Description, Visible: l.Visible, Sort: l.Sort}).Error; err != nil {
				return err
			}
		}
		for _, cm := range importBundle.Comments {
			if cm.PostSlug == "" {
				continue
			}
			var post model.Post
			if err := tx.Select("id").Where("slug = ?", cm.PostSlug).First(&post).Error; err != nil {
				continue
			}
			var n int64
			tx.Model(&model.Comment{}).Where("post_id = ? AND content = ? AND nickname = ?", post.ID, cm.Content, cm.Nickname).Count(&n)
			if n > 0 {
				continue
			}
			if err := tx.Create(&model.Comment{
				PostID: post.ID, Nickname: cm.Nickname, Email: cm.Email, Website: cm.Website,
				Content: cm.Content, Status: cm.Status, IsAdmin: cm.IsAdmin,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "system.import", "system", "", fmt.Sprintf("导入 strategy=%s posts=%d", strategy, len(importBundle.Posts)))
	model.CreateNotification(model.DB, "backup", "导入完成", fmt.Sprintf("strategy=%s，posts=%d", strategy, len(importBundle.Posts)), "/posts")
	common.OK(c, gin.H{"updated": updated, "summary": summary, "conflicts": conflicts})
}

func resolveCategoryID(tx *gorm.DB, name string) uint {
	if name == "" {
		return 0
	}
	var cat model.Category
	if err := tx.Where("name = ?", name).First(&cat).Error; err != nil {
		return 0
	}
	return cat.ID
}

// importMediaRelPath 解析 zip 内媒体条目的安全相对路径（去掉 media/ 前缀）。
// 返回 false 表示非法条目：空、穿越（含 .. 段）、绝对路径或盘符前缀，一律拒绝落盘。
func importMediaRelPath(name string) (string, bool) {
	norm := strings.ReplaceAll(name, "\\", "/")
	if path.IsAbs(norm) || strings.Contains(norm, "..") || strings.Contains(norm, ":") {
		return "", false
	}
	rel := path.Clean(strings.TrimPrefix(norm, "media/"))
	if rel == "" || rel == "." || rel == ".." ||
		strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
		return "", false
	}
	return rel, true
}

// extractFile 解压单个 zip 条目（调用方已校验目标路径，防穿越）
func extractFile(zf *zip.File, dest string) error {
	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

type importPost struct {
	ID           uint       `json:"id"`
	Title        string     `json:"title"`
	Slug         string     `json:"slug"`
	Summary      string     `json:"summary"`
	Content      string     `json:"content"`
	Cover        string     `json:"cover"`
	CategoryID   uint       `json:"categoryId"`
	CategoryName string     `json:"categoryName"`
	SeriesID     uint       `json:"seriesId"`
	SeriesSort   int        `json:"seriesSort"`
	Tags         []string   `json:"tags"`
	Status       int8       `json:"status"`
	IsTop        bool       `json:"isTop"`
	PublishedAt  *time.Time `json:"publishedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type importComment struct {
	PostSlug  string    `json:"postSlug"`
	ParentID  uint      `json:"parentId"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Website   string    `json:"website"`
	Content   string    `json:"content"`
	Status    int8      `json:"status"`
	IsAdmin   bool      `json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
}
