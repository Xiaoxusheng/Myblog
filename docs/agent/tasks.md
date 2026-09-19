# Tasks — Task Board

负责人：Backend / Web(前台) / Admin(管理端) / Orchestrator(联调+QA)

## TODO

- [x] [Orchestrator] Contract：api.md + database.md
- [x] [Orchestrator] 需求与架构文档
- [ ] [Backend] server/ 全部实现（模型/种子/公开接口/管理接口/上传/RSS/限流/JWT）
- [ ] [Backend] 后端测试（登录、文章分页、评论流转、点赞）+ go build/vet/test 通过
- [ ] [Backend] Docker 部署物：server/Dockerfile、deploy/（门户镜像+nginx.conf）、docker-compose.yml、.env.example、docs/03-部署文档.md
- [ ] [Web] web/ 全部页面 + 组件 + API 层 + 暗色模式 + 响应式 + npm run build 通过
- [x] [Admin] admin/ 全部页面 + AntD + md-editor-v3 + ECharts + npm run build 通过 ✅
- [ ] [Orchestrator] 联调：起后端，冒烟清单（api.md 末尾）逐项过
- [ ] [Orchestrator] 前台/管理端页面视觉验收
- [ ] [Orchestrator] Docker 部署验证（本地可 build 则验证；含 MySQL 模式说明）
- [ ] [Orchestrator] README 定稿、最终提交

## DOING

- [ ] [Backend] 并行开发中（含 Docker 部署物）
- [ ] [Web] 并行开发中

## DONE

- [x] [Orchestrator] 项目初始化、git init、文档与契约
- [x] [Admin] 管理端全部页面，npm run build（vue-tsc + vite）零错误，status-admin.md 已写
