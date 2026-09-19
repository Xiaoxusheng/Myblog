# DECISIONS

1. **技术栈**：Go 1.27 + Gin + GORM + SQLite(glebarez 纯 Go 驱动)；前端 Vue3+TS+Vite；管理端 Ant Design Vue 4。依据：本机 Go 生态与既有项目习惯（EchoIM 同为 Vue+AntD 模式），SQLite 免安装零配置，Windows 免 CGO。
2. **后端不分 service 层**：博客业务简单，handler → GORM 两层即可，避免过度设计。
3. **单管理员**：不做注册/RBAC，users 表仅一条管理员记录。
4. **评论免注册**：昵称+邮箱（学 WordPress/Typecho），审核制 + IP 限流。
5. **API 响应统一 body envelope**，业务错误 HTTP 200 + code，仅 401/500 用 HTTP 状态码。
6. **契约唯一来源**：docs/contracts/api.md，任何 API 变更必须先改契约。
7. **前端直接写脚手架文件**（不用 create-vite 交互命令），保证依赖版本可控。
8. **Markdown 存原文、前端渲染**：服务端不存 HTML。
9. **点赞不鉴权**，前端 localStorage 防重复，简单博客可接受。
10. **部署（用户新增需求）**：Docker Compose。server 镜像多阶段构建（CGO_ENABLED=0）；前台/管理端由一个"门户"nginx 镜像承载（80=前台、8081=管理端、反向代理 /api /uploads /rss → server:8080）；MySQL 两种模式：compose 自带 mysql:8 服务，或复用服务器已有 MySQL（DSN 指过去，只起 server+nginx）。默认部署仍以 BLOG_DB_TYPE=mysql + utf8mb4 为准。
11. **本地开发默认 SQLite**（零配置），MySQL 主要用于部署环境；MySQL 驱动代码保留但本地不强制测试（服务器上验证）。
