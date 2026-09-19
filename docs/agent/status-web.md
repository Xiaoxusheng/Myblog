# Web 前台状态报告（Frontend Agent）

日期：2026-09-19
范围：`D:\myblog\web\`（Vite + Vue 3 + TypeScript，无 UI 组件库，纯手写样式）

## 完成了什么

- 全部前台页面与组件（在本会话前已由前序 Web 会话搭建骨架，本会话完成审查、修复与验证）：
  - 首页 `/`：文章卡片列表（封面/标题/摘要/分类/标签/时间/阅读量/置顶标记）+ 分页（pageSize 取 `/site` 的 `postPageSize`，页码同步到 URL）
  - 文章详情 `/post/:slug`：markdown-it 渲染（`html:false` 防 XSS）+ highlight.js 代码高亮（浅色/暗色两套 token 配色，随 `html.dark` 切换，按需注册 13 种语言）+ 自动生成 TOC（桌面右侧吸附、滚动高亮当前节，≤1199px 隐藏）+ 上一篇/下一篇 + 相关文章 + 点赞（localStorage 防重复）+ 阅读量展示 + 评论区
  - 评论区：两级树展示、游客表单（昵称/邮箱必填、网站选填，前端校验 + 提交 Loading + 错误码 20002/20003 专门提示），提交成功提示「已提交，等待审核」，评论者信息 localStorage 记忆；`commentEnabled=false` 时显示已关闭
  - 归档 `/archives`：按年分组时间线（按契约 `GET /archive` 结构）
  - 分类汇总 `/categories`、分类文章 `/category/:slug`；标签云汇总 `/tags`、标签文章 `/tag/:slug`
  - 搜索 `/search?keyword=`：复用 ArticleCard 列表组件，关键词变化（含前进/后退）重新搜索
  - 友链卡片墙 `/links`（logo 加载失败自动回退首字占位）
  - 自定义页面 `/page/:slug`（Markdown 渲染）、404 页
- 布局：Header（站点名/Logo、导航、搜索入口、三态暗色切换、移动端汉堡菜单）、Footer（版权 + ICP + RSS + 页脚文字，取自 `/site`）、侧栏（公告、热门文章 top5、标签云）
- 暗色模式：CSS 变量双主题，亮/暗/跟随系统三态，`html.dark` 切换，localStorage 记忆，index.html 内联脚本首帧防闪烁
- 状态完备：所有列表/详情均有骨架屏（shimmer）、空态（EmptyState）、错误态（ErrorState 带重试）；异步请求统一 AbortController 取消防竞态
- 路由守卫：切换页面回到顶部（scrollBehavior，前进/后退保留位置）、`afterEach` 设置 `document.title`（页面名 · 站点名）
- 本会话修复的问题：
  1. **契约 Bug**：`createComment` 此前直接把响应 data 类型化为评论对象，契约 #5 实际返回 `{comment:{...}}`，已改为正确解包（`src/api/post.ts`）
  2. **健壮性**：`CommentPublic.parentId` 类型改为 `number | null`（顶级评论兼容 0/null），回复二级评论时挂顶级父 id 的判断做判空（`src/types/index.ts`、`CommentForm.vue`）
  3. **错误态漏洞**：分类/标签文章页在 `/site` 加载失败时误显示「该分类/标签下还没有文章」，已补充「站点信息加载失败」+ 重试入口（`CategoryPostsView.vue`、`TagPostsView.vue`）
  4. **中文排版**：全站用户可见文案标点全角化（，：！？～（）），与「已提交，等待审核」等要求文案精确一致

## 新建/修改文件清单（web/，共 51 个源文件）

- 脚手架：`package.json`、`vite.config.ts`、`tsconfig.json`、`index.html`、`public/favicon.svg`、`src/vite-env.d.ts`、`src/main.ts`、`src/App.vue`
- API 层：`src/api/http.ts`（axios 封装、code!==0 拒绝并携带 message、ApiError、取消识别）、`src/api/site.ts`、`src/api/post.ts`、`src/api/content.ts`
- 类型：`src/types/index.ts`（与契约对象一一对应）
- 路由/状态：`src/router/index.ts`、`src/stores/site.ts`、`src/stores/theme.ts`
- 组合式：`src/composables/usePagedList.ts`（分页状态机 + 竞态取消）、`src/composables/useToast.ts`
- 工具：`src/utils/format.ts`、`src/utils/markdown.ts`、`src/utils/storage.ts`、`src/utils/title.ts`
- 布局组件：`SiteHeader.vue`、`SiteFooter.vue`、`SiteSidebar.vue`
- 通用组件：`ArticleCard.vue`、`EmptyState.vue`、`ErrorState.vue`、`ListSkeleton.vue`、`Pagination.vue`、`SearchBox.vue`、`ThemeToggle.vue`、`ToastHost.vue`
- 文章组件：`PostToc.vue`、`CommentSection.vue`、`CommentForm.vue`、`CommentItem.vue`
- 视图：`HomeView.vue`、`PostDetailView.vue`、`ArchivesView.vue`、`CategoriesView.vue`、`CategoryPostsView.vue`、`TagsView.vue`、`TagPostsView.vue`、`SearchView.vue`、`LinksView.vue`、`PageView.vue`、`NotFoundView.vue`
- 样式：`src/styles/main.css`（设计令牌双主题/Reset/布局/通用类）、`src/styles/markdown.css`（Markdown 排版 + 高亮配色）

## 构建结果

- `npm run build`（vue-tsc --noEmit + vite build）：**通过，exit 0，零错误**（本会话修复后复验两次）
- 独立 `npx vue-tsc --noEmit`：exit 0
- `npm install`：依赖与 package-lock 同步（up to date）
- 产物：`web/dist/`，主包约 172KB（gzip 66KB），markdown+highlight 懒加载分包约 179KB（gzip 70KB）

## 未完成事项 / 说明

- 后端尚未就绪，页面仅按契约封装、未做真实联调（属计划内，由 Orchestrator 联调）
- 浏览器端视觉/交互验收（亮暗双主题、移动端真机）未执行——dev server 未长期运行，留待联调阶段
- `web/Dockerfile` 按分工由 Backend Agent 负责，未创建
- 契约疑问（未改动契约，仅代码容错）：契约未写明顶级评论 `parentId` 具体取值（前端已按 0/null 双兼容处理）
