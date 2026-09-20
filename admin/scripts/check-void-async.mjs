/**
 * 静态检查：fire-and-forget 调用必须被捕获。
 *
 * 背景（2026-09-20 线上事故）：
 *   AdminLayout.loadNotifications() 与 AnalyticsView.loadSearchStats() 都是
 *   `try { ... } finally { ... }` 形式 —— 缺 catch，且调用方用 `void fn()` 触发。
 *   后端 500 时 rejection 逃逸为 unhandledrejection，并沿用了接口层的全局错误 toast，
 *   于是「后台 60s 轮询失败」被渲染成整页「网络连接失败」，每分钟弹一次。
 *
 * 判定规则（保守，目标是零误报）：
 *   对每个 `void <f>(` 调用点：
 *     1. <f> 必须是**本文件内定义**的 async 函数；否则跳过（可能来自 composable/导入）
 *     2. <f> 函数体里必须有 `catch` 或 `allSettled`；否则报错
 *     3. 调用点后紧跟 `).catch(` 视为已处理
 *
 * 用法：node scripts/check-void-async.mjs
 */
import { readFileSync } from 'node:fs'
import { readdir } from 'node:fs/promises'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const ROOT = fileURLToPath(new URL('..', import.meta.url))
const SRC = join(ROOT, 'src')

async function walk(dir, out = []) {
  for (const e of await readdir(dir, { withFileTypes: true })) {
    const p = join(dir, e.name)
    if (e.isDirectory()) await walk(p, out)
    else if (/\.(vue|ts)$/.test(e.name)) out.push(p)
  }
  return out
}

/** 只收集**本文件内**的 async 函数定义（含 `const x = async () =>`） */
function localAsyncFns(src) {
  const map = new Map()
  for (const m of src.matchAll(/async\s+function\s+([A-Za-z_$][\w$]*)\s*\(/g)) {
    map.set(m[1], m.index)
  }
  for (const m of src.matchAll(/(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*async\s*[\s\S]{0,20}?\(/g)) {
    if (!map.has(m[1])) map.set(m[1], m.index)
  }
  return map
}

/** 取函数体（大括号配对） */
function bodyOf(src, startIdx) {
  const i = src.indexOf('{', startIdx)
  if (i < 0) return null
  let depth = 0
  for (let j = i; j < src.length; j++) {
    if (src[j] === '{') depth++
    else if (src[j] === '}') {
      depth--
      if (depth === 0) return src.slice(i, j + 1)
    }
  }
  return null
}

const problems = []
for (const file of await walk(SRC)) {
  const src = readFileSync(file, 'utf8')
  const fns = localAsyncFns(src)
  if (fns.size === 0) continue

  for (const m of src.matchAll(/void\s+([A-Za-z_$][\w$]*)\s*\(/g)) {
    const name = m[1]
    if (!fns.has(name)) continue // 非本文件定义 → 跳过（避免 composable/导入误报）

    // 调用点后 60 字符内 .catch( 视为已处理
    const after = src.slice(m.index, m.index + 300)
    if(/\)\s*\.\s*catch\s*\(/.test(after.slice(0, 80))) continue

    const body = bodyOf(src, fns.get(name))
    if (!body) continue
    const guarded = /\bcatch\s*\(|\bcatch\s*\{|allSettled/.test(body)
    if (!guarded) {
      problems.push({ file: relative(ROOT, file), line: src.slice(0, m.index).split('\n').length, name })
    }
  }
}

if (problems.length) {
  console.error('[check-void-async] 以下 fire-and-forget 调用的目标 async 函数没有 catch：')
  for (const p of problems) {
    console.error(`  ✗ ${p.file}:${p.line}  void ${p.name}(...)  —— ${p.name}() 体内缺 catch/allSettled`)
  }
  console.error('\n修复：在该 async 函数中补 catch（后台轮询/次级面板应吞掉异常并局部降级），')
  console.error('或改为 `void fn().catch(() => {})`。接口层可用 { silent: true } 抑制全局 toast。')
  process.exit(1)
}
console.log('[check-void-async] OK：所有 fire-and-forget 的本地 async 调用都有 catch 兜底')
