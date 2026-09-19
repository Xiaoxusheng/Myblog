# MyBlog

个人博客系统：Go 后端 + Vue3 前台 + Vue3 管理端，前后端分离。

## 技术栈

- 后端：Go / Gin / GORM / SQLite（可选 MySQL）/ JWT
- 前台：Vue 3 + TypeScript + Vite + Pinia + markdown-it + highlight.js
- 管理端：Vue 3 + TypeScript + Vite + Ant Design Vue 4 + md-editor-v3 + ECharts

## 快速开始

```bash
# 1. 后端（:8080，首次自动建库 + 写入种子数据）
cd server && go mod tidy && go run .

# 2. 前台（:5173）
cd web && npm install && npm run dev

# 3. 管理端（:5174）
cd admin && npm install && npm run dev
```

管理端初始账号：`admin / admin123`（请登录后立即在"个人资料"中修改密码）。

## 文档

- 需求与功能清单：`docs/01-需求与功能清单.md`
- 架构设计：`docs/02-架构设计.md`
- API 契约（唯一真相来源）：`docs/contracts/api.md`
- 数据库契约：`docs/contracts/database.md`
