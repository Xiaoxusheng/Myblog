# MyBlog

个人博客系统：Go 后端 + Vue3 前台 + Vue3 管理端，前后端分离。

## 技术栈

- 后端：Go / Gin / GORM / SQLite（可选 MySQL）/ JWT
- 前台：Vue 3 + TypeScript + Vite + Pinia + markdown-it + highlight.js
- 管理端：Vue 3 + TypeScript + Vite + Ant Design Vue 4 + md-editor-v3 + ECharts

## 快速开始（本机开发，零配置，默认 SQLite）

```bash
# 1. 后端（:8080，首次自动建库 + 写入种子数据）
cd server && go mod tidy && go run .

# 2. 前台（:5173）
cd web && npm install && npm run dev

# 3. 管理端（:5174）
cd admin && npm install && npm run dev
```

管理端初始账号：`admin / admin123`（请登录后立即在"个人资料"中修改密码）。

> 若本机 5173/5174 端口被占用，可换端口启动：`npm run dev -- --port 5200`（后端不变，Vite 代理目标始终是 :8080）。

## Docker 部署（服务器）

支持两种模式：**A. Compose 全套（自带 MySQL）**；**B. 复用服务器已有 MySQL**（只起 server + nginx）。
完整步骤见 [docs/03-部署文档.md](docs/03-部署文档.md)，最简流程：

```bash
cp .env.example .env && vim .env   # 修改密码/密钥/DSN
docker compose --profile bundled up -d -build   # 模式 A；模式 B 去掉 --profile bundled
```

启动后：80=前台，8081=管理端，8080=API。

## 文档

- 需求与功能清单：`docs/01-需求与功能清单.md`
- 架构设计：`docs/02-架构设计.md`
- API 契约（唯一真相来源）：`docs/contracts/api.md`
- 数据库契约：`docs/contracts/database.md`
- Docker 部署：`docs/03-部署文档.md`
