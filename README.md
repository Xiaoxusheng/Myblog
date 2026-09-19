# MyBlog

基于 **Go + Vue 3** 的个人博客系统，前后端分离：前台展示 + 访客互动 + 管理后台，单管理员模式，开箱即用。

🔗 **在线示例**：[xyx.homes](https://xyx.homes)

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white)
![Vite](https://img.shields.io/badge/Vite-7-646CFF?logo=vite&logoColor=white)
![Ant Design Vue](https://img.shields.io/badge/Ant_Design_Vue-4-1677FF?logo=antdesign&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)

## 功能特性

### 前台（访客端）

- **文章**：Markdown 渲染、代码高亮与一键复制、图片点击预览、目录 TOC、阅读时长、上一篇/下一篇、相关文章、点赞、阅读量、置顶
- **浏览**：首页分页列表（封面/摘要/分类/标签）、归档时间线（按年月）、分类页、标签页、全文搜索
- **互动**：游客评论（昵称/邮箱/网站选填）、楼中楼二级回复、审核通过后展示
- **页面**：自定义页面（如"关于"）、友链、公告栏、热门文章、标签云
- **其他**：RSS 订阅（`/rss`）、sitemap / robots.txt、SEO meta、暗色模式（跟随系统 + 手动切换）、移动端响应式

### 管理后台

- **仪表盘**：文章/草稿/计划发布/评论/浏览/点赞统计卡片、最近 7 天发布与评论趋势图、最近评论、计划发布
- **文章管理**：Markdown 编辑器（md-editor-v3）、封面（媒体库或外链）、多标签、草稿/发布/隐藏/定时发布、置顶、自动保存（本地兜底 + 服务器草稿）、版本历史（对比/一键恢复）、删除
- **内容管理**：分类、标签、**专题**（教程系列：成员文章排序、前台专题页与"第 X/N 篇"导航）、友链、自定义页面（均含文章数/排序/显隐）
- **重定向**：旧地址 301/302 管理、环检测；修改文章 slug 时可自动创建旧→新 301
- **评论管理**：按状态筛选、通过/拒绝、管理员回复、删除（可见 IP/邮箱）
- **媒体库**：图片上传、列表、删除
- **系统设置**：站点名称/描述/关键词/URL、Logo、公告、ICP 备案、页脚、评论开关、每页文章数
- **账号**：登录/登出、修改密码、个人资料（昵称/邮箱/头像）

## 技术栈

| 端 | 技术 |
|---|---|
| 后端 `server/` | Go 1.27 · Gin · GORM · SQLite（默认，纯 Go 驱动，零 CGO）/ MySQL（可选）· JWT · bcrypt |
| 前台 `web/` | Vue 3 · TypeScript · Vite · Pinia · Vue Router · markdown-it · highlight.js |
| 管理端 `admin/` | Vue 3 · TypeScript · Vite · Ant Design Vue 4 · md-editor-v3 · ECharts |
| 部署 | Docker Compose · nginx · 自动备份脚本 |

## 项目结构

```
myblog/
├── server/               # Go 后端 API（handler / model / middleware / router）
├── web/                  # 博客前台
├── admin/                # 管理后台
├── deploy/               # Dockerfile / nginx 配置 / 备份脚本
├── docs/                 # 需求清单、架构设计、部署文档、API 与数据库契约
├── docker-compose.yml
└── .env.example
```

## 快速开始（本机开发，零配置）

默认使用 SQLite，首次启动自动建库并写入种子数据：

```bash
# 1. 后端（:8080）
cd server && go mod tidy && go run .

# 2. 前台（:5173）
cd web && npm install && npm run dev

# 3. 管理端（:5174）
cd admin && npm install && npm run dev
```

管理端初始账号：`admin / admin123`（请登录后立即在"个人资料"中修改密码）。

> 若本机 5173/5174 端口被占用，可换端口启动：`npm run dev -- --port 5200`（后端不变，Vite 代理目标始终是 :8080）。

运行后端测试：

```bash
cd server && go test ./...
```

## Docker 部署（服务器）

支持两种模式：**A. Compose 全套（自带 MySQL）**；**B. 复用服务器已有 MySQL**（只起 server + nginx）。
完整步骤见 [docs/03-部署文档.md](docs/03-部署文档.md)，最简流程：

```bash
cp .env.example .env && vim .env   # 修改密码/密钥/DSN
docker compose --profile bundled up -d --build   # 模式 A；模式 B 去掉 --profile bundled
```

启动后：80 = 前台，8081 = 管理端，8080 = API。

## 文档

- [需求与功能清单](docs/01-需求与功能清单.md)
- [架构设计](docs/02-架构设计.md)
- [Docker 部署](docs/03-部署文档.md)
- [API 契约（唯一真相来源）](docs/contracts/api.md)
- [数据库契约](docs/contracts/database.md)
