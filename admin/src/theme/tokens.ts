/**
 * 管理端设计 token 单一来源(docs/09 §4.2)
 * AntD 控件层(ConfigProvider theme)与布局层(--admin-* CSS 变量)由此表派生,
 * ECharts 色板也从这里取值;组件内禁止另写颜色字面量。
 * 暗色模式(M5)时 dark 表也从本模块导出。
 */

/** AntD ConfigProvider token(控件层·浅色):值与 main.css 的 --admin-* 保持同源 */
export const antdTokens = {
  borderRadius: 6,
  colorPrimary: '#1677ff',
  colorBgLayout: '#f5f6f8',
  colorBgContainer: '#ffffff',
  colorBorder: '#e6e8eb',
  colorText: 'rgba(0, 0, 0, 0.88)',
  colorTextSecondary: 'rgba(0, 0, 0, 0.45)',
  fontSize: 14,
  controlHeight: 32
} as const

/** AntD ConfigProvider token(控件层·暗色):显式提供色值以配合 darkAlgorithm(docs/09 §8.1) */
export const antdDarkTokens = {
  ...antdTokens,
  colorBgLayout: '#141414',
  colorBgContainer: '#1f1f1f',
  colorBorder: '#333333',
  colorText: 'rgba(255, 255, 255, 0.88)',
  colorTextSecondary: 'rgba(255, 255, 255, 0.45)'
} as const

/** 图表色板(浅色):ECharts 与主题同源,不拥有独立视觉体系 */
export const chartPalette = {
  primary: '#1677ff',
  secondary: '#a6adb8',
  axisLine: '#e6e8eb',
  axisLabel: 'rgba(0, 0, 0, 0.45)',
  splitLine: '#f0f1f3',
  legendText: 'rgba(0, 0, 0, 0.65)'
} as const

/** 图表色板(暗色) */
export const chartDarkPalette = {
  primary: '#1668dc',
  secondary: 'rgba(255, 255, 255, 0.35)',
  axisLine: '#333333',
  axisLabel: 'rgba(255, 255, 255, 0.45)',
  splitLine: '#262626',
  legendText: 'rgba(255, 255, 255, 0.65)'
} as const
