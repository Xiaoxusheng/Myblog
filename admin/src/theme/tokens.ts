/**
 * 管理端设计 token 单一来源(docs/09 §4.2 / docs/prototypes/admin-redesign-v2)
 *
 * v2 换血(2026-09-21)：按设计稿 PDF 实测色值重写,不靠目测。
 * 取色脚本 `.agent/_palette.py`(Pillow 区域主色统计)输出：
 *   主色     #5e6ad2(登录页品牌方块实测 26% + 按钮 40% 覆盖)
 *   亮色     页底 #f5f7f7 / 卡片 #ffffff / 侧栏 #f5f7f7(与页底同色) / 边框 #e3e6e8
 *   暗色     页底 #0f0f12 / 卡片 #131417 / 侧栏 #000003(比页底更黑) / 顶栏 #080a0a
 *
 * AntD 控件层(ConfigProvider theme)与布局层(--admin-* CSS 变量)由此表派生,
 * ECharts 色板也从这里取值;组件内禁止另写颜色字面量。
 */

/** AntD ConfigProvider token(控件层·浅色):值与 main.css 的 --admin-* 保持同源 */
export const antdTokens = {
  /** 卡片圆角 8(设计稿档),控件圆角 6 —— 卡片与控件分档,避免小控件显胖 */
  borderRadius: 6,
  borderRadiusLG: 8,
  /** 品牌靛蓝:低饱和,亮蓝 #1677ff 在浅灰底上过于跳脱,与设计稿气质不符 */
  colorPrimary: '#5e6ad2',
  colorBgLayout: '#f5f7f7',
  colorBgContainer: '#ffffff',
  colorBorder: '#e3e6e8',
  colorText: 'rgba(15, 18, 25, 0.88)',
  colorTextSecondary: 'rgba(15, 18, 25, 0.55)',
  colorTextTertiary: 'rgba(15, 18, 25, 0.38)',
  /** 表头/凹陷面用的极浅灰 */
  colorFillQuaternary: '#fafbfc',
  fontSize: 14,
  controlHeight: 32
} as const

/** AntD ConfigProvider token(控件层·暗色):显式提供色值以配合 darkAlgorithm(docs/09 §8.1) */
export const antdDarkTokens = {
  ...antdTokens,
  colorPrimary: '#6f7ce0',
  colorBgLayout: '#0f0f12',
  colorBgContainer: '#131417',
  colorBorder: '#26282d',
  colorText: 'rgba(255, 255, 255, 0.9)',
  colorTextSecondary: 'rgba(255, 255, 255, 0.55)',
  colorTextTertiary: 'rgba(255, 255, 255, 0.38)',
  colorFillQuaternary: '#141417'
} as const

/** 图表色板(浅色):ECharts 与主题同源,不拥有独立视觉体系 */
export const chartPalette = {
  /** 主序列:与 colorPrimary 同值,图表与控件同一种蓝 */
  primary: '#5e6ad2',
  secondary: '#a8adb5',
  axisLine: '#e3e6e8',
  axisLabel: 'rgba(15, 18, 25, 0.45)',
  splitLine: '#eef0f2',
  legendText: 'rgba(15, 18, 25, 0.65)',
  /** 图例未选中态:与选中态拉开明显差距,让"选中"一眼可辨 */
  legendMuted: 'rgba(15, 18, 25, 0.25)',
  /** 悬浮指示线 */
  axisPointer: 'rgba(15, 18, 25, 0.16)',
  /** 提示框:沿用卡片表面的浮层语言(白底 + 细描边 + 柔影由 extraCssText 提供) */
  tooltipBg: 'rgba(255, 255, 255, 0.98)',
  tooltipBorder: '#e3e6e8',
  tooltipText: 'rgba(15, 18, 25, 0.88)',
  /** 峰值标记的描边环:取卡片底色,让圆点"嵌入"折线而非浮在上面 */
  markerRing: '#ffffff',
  /** 面积填充渐变:极低透明度,只做体积暗示,不喧宾夺主 */
  areaPrimary: ['rgba(94, 106, 210, 0.18)', 'rgba(94, 106, 210, 0)'],
  areaSecondary: ['rgba(168, 173, 181, 0.14)', 'rgba(168, 173, 181, 0)'],
} as const

/** 图表色板(暗色) */
export const chartDarkPalette = {
  primary: '#6f7ce0',
  secondary: 'rgba(255, 255, 255, 0.35)',
  axisLine: '#26282d',
  axisLabel: 'rgba(255, 255, 255, 0.45)',
  splitLine: '#1d1f23',
  legendText: 'rgba(255, 255, 255, 0.65)',
  legendMuted: 'rgba(255, 255, 255, 0.22)',
  axisPointer: 'rgba(255, 255, 255, 0.18)',
  tooltipBg: 'rgba(19, 20, 23, 0.98)',
  tooltipBorder: '#26282d',
  tooltipText: 'rgba(255, 255, 255, 0.88)',
  markerRing: '#131417',
  areaPrimary: ['rgba(111, 124, 224, 0.28)', 'rgba(111, 124, 224, 0)'],
  areaSecondary: ['rgba(255, 255, 255, 0.12)', 'rgba(255, 255, 255, 0)'],
} as const
