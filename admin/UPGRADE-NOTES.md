# Admin 后台产品级升级说明（2026-09-19）

风格目标：Linear/Vercel 式克制后台。API 契约（docs/contracts/api.md）未改动，路由与既有功能全部保留。

## 改动文件清单

- `src/styles/main.css`：建立 `--admin-*` Design Token（bg/surface/surface-2/border/text/muted/brand/warning/danger/sidebar、圆角 6/8、两档低强度阴影）；全局卡片=白底+细描边+极轻投影，区块标题统一 14px/600；间距只用 8/12/16/20/24
- `src/layouts/AdminLayout.vue`：侧边菜单按「内容/互动/资源/系统」a-menu-item-group 分组（小灰字组标题、折叠态隐藏组标题）；Header 改 sticky top0 + 1px 底边框 + surface 背景，高度 56 保留；小屏抽屉逻辑保留
- `src/components/StatCard.vue`（新增）：图标浅底块 + 标题 + 大数字(tabular-nums) + 真实字段辅助信息，支持 warning 态
- `src/components/FormSection.vue`（新增）：13px/600/muted 小标题 + 16 内边距的分组卡片
- `src/views/DashboardView.vue`：改为 StatCard 六卡（文章/草稿/评论/待审[>0 warning 色]/浏览/点赞，辅助信息全部来自 /admin/stats 真实字段，无虚构数据）；新增快捷操作四格（新建文章/审核评论/媒体库/系统设置）；趋势图降饱和（蓝+中性灰）、无数据显示 Empty；最近评论重排：首字母头像、状态 Tag、两行截断、待审行 warning 竖线+浅底、点击跳 /comments
- `src/components/TrendChart.vue`：配色降饱和 + 轴/分割线/图例细节，resize/dispose 逻辑保留
- `src/views/PostEditView.vue`：升级 CMS 布局——顶部工具条（返回 + 无框化大标题 + 保存草稿/发布）；≥1200 左 1fr/右 320px，<1200 右栏折到正文下；右栏 FormSection 分组「发布/分类与标签(含 Slug)/封面/摘要」；本地草稿自动保存（3s 防抖写 localStorage，key 按新建/编辑+id 区分，进入页面草稿与服务器不同则 Modal 询问恢复，保存/发布成功即清除，新建成功后同步 id 到路由）；编辑器下沿 32px 状态栏：字数、预计阅读时长（400 字/分钟，computed 实时）、已保存至本地 HH:mm / 正在保存… / 保存失败（无 Toast）；标题必填改为保存前校验（无框化输入不再挂 form-item），粘贴上传等原有行为保留
- `src/views/MediaView.vue`：右上角 Segmented 网格/列表切换；网格 4:3 cover 缩略图 + 文件名一行 + 大小/时间，hover 显示 预览/复制 URL/删除；列表 60px 紧凑行；点击图片 Modal 放大；复制成功 message.success('链接已复制')；修复 ReloadOutlined 缺失 import
- `src/views/CommentView.vue`：状态 Tab 保留；支持 `/comments?status=0` 直达（联动仪表盘快捷入口并同步 query）；行点击/「详情」打开 a-drawer（完整内容 + 邮箱/网站/IP + 通过/拒绝/回复/删除 + 「查看该文章的其他评论」按 postId 过滤）；回复 Modal 保留
- 各列表页（Category/Tag/Link/PageList/PostList）：删除确认文案统一为「删除后不可恢复，…确定删除吗？」风格；PostList 标题色改 token；仅统一视觉与结构，逻辑未动
- `src/components/MediaSelectModal.vue`：删除文案对齐 + 硬编码色值换 token

## 说明与未尽事项

- 媒体库「尺寸」未展示：契约 UploadItem 无宽高字段，列表模式以 mime 代替，未伪造数据
- 暗色模式按需求未做（后台保持亮色）；ECharts 内为字面量色值（canvas 不支持 CSS 变量）
- 已验证：`npm run build`（vue-tsc --noEmit + vite build）零错误；路由无增删；5201 已有 dev 服务未动
- 未验证：浏览器端逐页交互（恢复草稿弹窗、Drawer、hover 操作等）未实机点验，建议联调时过一遍
