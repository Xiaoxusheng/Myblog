# Status — Admin（管理端）

负责人：Admin Agent
日期：2026-09-19

## 完成了什么

从零实现 `admin/` 管理后台：Vite 6 + Vue 3.5 + TypeScript（strict）+ Ant Design Vue 4 + Pinia 3 + axios + md-editor-v3 + ECharts 5，全部界面中文。

- **脚手架**：手写全部配置（未用 create-vite 交互命令）；dev 端口 5174；代理 `/api`、`/uploads` → `http://localhost:8080`；别名 `@` → `src`；构建产物对 echarts / md-editor / antd 做 manualChunks 拆分。
- **API 层**（`src/api/http.ts`）：baseURL `/api/v1`；请求拦截器带 `Authorization: Bearer <token>`（localStorage key：`blog_admin_token`）；响应拦截器解包 `{code,message,data}`——code!==0 时 `message.error` 并 reject（ApiError 携带错误码）；HTTP 401 或 code=10002 清 token 跳 `/login`（带 redirect 参数，登录页已跳转时静默）。所有管理端接口均按契约 `docs/contracts/api.md` 封装为独立模块并带 TS 类型。
- **路由与布局**：`/login` 独立页；其余统一侧边菜单布局（可折叠、lg 断点自动收起、面包屑、右上角用户下拉：个人资料 / 修改密码 / 退出登录确认 Modal）。路由守卫：无 token 一律跳 `/login`；已登录访问 `/login` 回仪表盘；`afterEach` 设置中文 document.title。菜单：仪表盘、文章管理、评论管理、分类管理、标签管理、页面管理、友链管理、媒体库、系统设置。
- **页面**（13 个视图，全部实现 Loading / Empty / 错误重试 / 危险操作 Popconfirm / 保存成功 message / 表单校验）：
  1. 登录页：居中卡片、回车提交（a-form @finish）、7 天免登录勾选、错误提示走拦截器（20001 等）。
  2. 仪表盘：7 张统计卡片（文章/草稿/评论/待审评论[>0 橙色警示]/总浏览/总点赞/友链数）+ ECharts 近 7 天发布/评论双折线（自适应 resize、组件卸载 dispose）+ 最近评论列表；加载用 Skeleton，失败给 Result + 重新加载按钮。
  3. 文章列表：关键词/状态/分类筛选 + 分页表格（页码、keyword、status、categoryId 同步 URL query）；状态 Tag 语义色（草稿 default/已发布 success/隐藏 warning）、置顶 Tag、标题+slug 双行、标签折叠（+N）；行操作：编辑、发布/下架快捷切换、删除（Popconfirm，删空自动回退页码）。
  4. 文章编辑（新建/编辑同一组件）：标题、slug（留空自动生成）、分类下拉（必填）、标签 mode="tags"、置顶 Switch、状态 Radio、摘要（500 字计数）、md-editor-v3 编辑器（zh-CN、图片粘贴/插入走 `/admin/uploads` 上传后回调插入）、封面（外链输入 / 从媒体库弹窗选择 / 清除 + 预览）、保存草稿/发布按钮（新建成功后 replace 到编辑路由，后续保存走 PUT）。
  5. 评论管理：Tab 按状态（全部/待审核/已通过/已拒绝）+ 分页表格（文章标题、昵称[管理员/回复标识]、邮箱、内容 Tooltip、IP、状态、时间）；操作：通过/拒绝（按状态显隐）、回复（Modal 展示原评论 + 输入）、删除（Popconfirm）。
  6. 分类管理：表格（名称/slug/描述/文章数）+ 新建/编辑 Modal（name 必填、slug 格式校验）+ 删除（提示文章将变未分类）。
  7. 标签管理：同上模式。
  8. 友链管理：表格含行内可见 Switch（失败自动还原、带行级 loading）、排序、logo 头像；Modal 表单 URL 格式校验。
  9. 页面管理 + 页面编辑：同文章模式（title/slug 必填[页面 slug 契约必填唯一]/content/status，md-editor-v3）。
  10. 媒体库：a-upload customRequest → POST /admin/uploads（前端预校验仅图片、≤10MB）+ 网格展示（a-image 预览、复制链接[clipboard + execCommand 降级]、删除）+ 分页。
  11. 系统设置：结构化 Settings 表单（horizontal labelCol），logo 支持从媒体库选择，commentEnabled Switch、postPageSize InputNumber(1–50)，PUT 保存。
  12. 个人资料：头像（上传取 url + 清除）、昵称必填、邮箱格式校验，成功后同步 Pinia store（下拉头像/昵称实时更新）；修改密码（旧密码 + 新密码≥6 位 + 确认一致校验），从下拉"修改密码"进入时平滑定位到密码卡片。

## 新建文件清单（均在 admin/ 下，共 39 个源文件）

```
admin/
├── index.html  package.json  tsconfig.json  vite.config.ts
└── src/
    ├── main.ts  App.vue
    ├── api/        http.ts auth.ts posts.ts comments.ts taxonomy.ts links.ts pages.ts media.ts stats.ts
    ├── components/ PageHeader.vue MarkdownEditor.vue TrendChart.vue MediaSelectModal.vue
    ├── constants/  status.ts
    ├── layouts/    AdminLayout.vue
    ├── router/     index.ts
    ├── stores/     auth.ts
    ├── styles/     main.css
    ├── types/      api.ts router.d.ts
    ├── utils/      format.ts
    └── views/      LoginView.vue DashboardView.vue PostListView.vue PostEditView.vue CommentView.vue
                    CategoryView.vue TagView.vue PageListView.vue PageEditView.vue LinkView.vue
                    MediaView.vue SettingsView.vue ProfileView.vue
```

## 构建结果

- `npm run build`（vue-tsc --noEmit 类型检查 + vite build）：**通过，零错误**（首次报 4 处 unused import / 类型导出问题已修复后通过）。
- npm install 用默认源一次成功（未启用 npmmirror 回退）。
- 遗留构建警告（非错误）：echarts（约 1MB）、md-editor-v3（约 840KB）单 chunk 超 500KB 体积提示——已拆为独立懒加载 chunk，管理端可接受。

## 契约备注（未改契约）

- 契约未定义"下架"语义，列表快捷操作实现为：已发布(1) → 下架置为隐藏(2)；草稿(0)/隐藏(2) → 发布置为已发布(1)。
- `GET /admin/links` 返回 `{list,total}`，`pageSize` 未在契约注明，默认传 10 并带分页 UI。
- `UploadRequestOption` 类型需从 `ant-design-vue/es/vc-upload/interface` 深度导入（根导出无此类型）。

## 未完成事项 / 未验证

- **浏览器实际运行与后端联调：未验证**（后端 server/ 尚未就绪，属预期；联调由 Orchestrator 按契约冒烟清单执行）。仅验证了类型检查与构建。
- 登录后的 token 过期自动跳转、上传图片回显等运行时行为依赖后端实现与契约一致。
- 无阻塞项，`docs/agent/blocked.md` 未创建。
