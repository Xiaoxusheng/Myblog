# TASK_STATE

任务：D:\myblog 个人博客系统（前台 + 管理端 + Go 后端 + Docker 部署）
最终目标：按 docs/ 契约完成全栈博客，构建/测试通过，可本地运行演示，可 Docker 部署

当前阶段：全部完成（2026-09-19）
整体进度：100%

已完成：
- [x] Phase 0/1：项目初始化、需求清单、架构设计、API 契约、DB 契约、任务板
- [x] Backend：47 项接口全部实现；go fmt/vet/build 通过；go test 33/33 通过（status-backend.md）
- [x] Web：11 类页面 + 全套组件；npm run build（vue-tsc + vite）零错误（status-web.md）
- [x] Admin：13 视图；npm run build 零错误（status-admin.md）
- [x] 联调冒烟：api.md 冒烟清单逐项通过（登录/文章/评论流转/点赞/上传/RSS/鉴权/统计/设置）
- [x] 浏览器视觉验收：前台首页/详情/暗色、管理端登录/仪表盘/文章管理，截图存 .smoke/shots
- [x] 联调期修复：admin 上传解包（media.ts）、admin AntD 全局注册（main.ts，此前 a-* 组件不渲染）、compose 模式 B 的 DSN 改为 .env 提供
- [x] 数据清理：冒烟文章/评论/乱码标签/测试上传已删，站点设置写为正式默认值
- [x] Docker 部署物：server/Dockerfile、deploy/{Dockerfile,nginx.conf}、docker-compose.yml（A/B 两模式）、.env.example、docs/03-部署文档.md

验证结果：
- go test ./... 33/33 ✅（-race 未跑：本机无 gcc）
- web/admin npm run build ✅
- 冒烟 + 页面视觉验收 ✅
- Docker build/compose up 未在本机验证（本机无 docker），需在服务器按部署文档验证

遗留：无阻塞。待服务器实际部署一次以闭环。
