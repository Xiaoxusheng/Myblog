/**
 * 纯文本行级 Diff（LCS 动态规划，无第三方依赖）
 *
 * 统一视图语义（diffLines(a, b)）：
 * - del：基准 a 有、目标 b 无（浅红）
 * - add：目标 b 有、基准 a 无（浅绿）
 * - same：两者相同（无底色）
 *
 * 性能保护：
 * 1. 先裁掉公共前缀 / 公共后缀，LCS 只计算中间差异区域（常规编辑场景下极小）；
 * 2. 中间区域格数超过 MAX_LCS_CELLS 时退化为「整段删除 + 整段新增」——仍是合法差异
 *    （非最小对齐），保证统计可用且不会分配巨型矩阵；
 * 3. 渲染层另有 DIFF_RENDER_LINE_LIMIT：任一侧超过该行数时只展示统计与提示，不逐行渲染。
 */

export interface DiffRow {
  type: 'same' | 'add' | 'del'
  text: string
}

/** 渲染保护上限：任一侧超过该行数时，调用方只显示统计与提示，不逐行渲染 */
export const DIFF_RENDER_LINE_LIMIT = 2000

/** LCS 矩阵最大格数（Int32Array 约 48MB），超过则退化为整段 del + add */
const MAX_LCS_CELLS = 12_000_000

/** 拆行：统一 \r\n；末尾单个换行视为行结束符而非空行；空串返回 [] */
function splitLines(text: string): string[] {
  if (text.length === 0) return []
  const normalized = text.replace(/\r\n/g, '\n')
  const body = normalized.endsWith('\n') ? normalized.slice(0, -1) : normalized
  return body.split('\n')
}

/** 行数（供渲染保护判断使用） */
export function countLines(text: string): number {
  return splitLines(text).length
}

function delRow(text: string): DiffRow {
  return { type: 'del', text }
}

function addRow(text: string): DiffRow {
  return { type: 'add', text }
}

function sameRow(text: string): DiffRow {
  return { type: 'same', text }
}

/** 中间差异区域的 LCS 对齐 */
function diffMiddle(base: string[], target: string[]): DiffRow[] {
  const m = base.length
  const n = target.length
  if (m === 0 && n === 0) return []
  if (m === 0) return target.map(addRow)
  if (n === 0) return base.map(delRow)

  // 中间区域过大：退化为整段删除 + 整段新增（合法差异，非最小对齐）
  if ((m + 1) * (n + 1) > MAX_LCS_CELLS) {
    return base.map(delRow).concat(target.map(addRow))
  }

  const width = n + 1
  let dp: Int32Array
  try {
    dp = new Int32Array((m + 1) * width)
  } catch {
    return base.map(delRow).concat(target.map(addRow))
  }

  // dp[i][j] = base[i..] 与 target[j..] 的 LCS 长度
  for (let i = m - 1; i >= 0; i -= 1) {
    for (let j = n - 1; j >= 0; j -= 1) {
      dp[i * width + j] =
        base[i] === target[j]
          ? dp[(i + 1) * width + (j + 1)] + 1
          : Math.max(dp[(i + 1) * width + j], dp[i * width + (j + 1)])
    }
  }

  const rows: DiffRow[] = []
  let i = 0
  let j = 0
  while (i < m && j < n) {
    if (base[i] === target[j]) {
      rows.push(sameRow(base[i]))
      i += 1
      j += 1
    } else if (dp[(i + 1) * width + j] >= dp[i * width + (j + 1)]) {
      rows.push(delRow(base[i]))
      i += 1
    } else {
      rows.push(addRow(target[j]))
      j += 1
    }
  }
  while (i < m) {
    rows.push(delRow(base[i]))
    i += 1
  }
  while (j < n) {
    rows.push(addRow(target[j]))
    j += 1
  }
  return rows
}

/**
 * 行级 Diff（统一视图）：a 为基准（旧），b 为目标（新）。
 * 返回的行序为「公共前缀 → 中间差异 → 公共后缀」。
 */
export function diffLines(a: string, b: string): DiffRow[] {
  const base = splitLines(a)
  const target = splitLines(b)

  const prefix: DiffRow[] = []
  let start = 0
  while (start < base.length && start < target.length && base[start] === target[start]) {
    prefix.push(sameRow(base[start]))
    start += 1
  }

  // 公共后缀（不与前缀重叠），收集后反转避免 unshift 的 O(n²)
  let endBase = base.length
  let endTarget = target.length
  const suffix: DiffRow[] = []
  while (endBase > start && endTarget > start && base[endBase - 1] === target[endTarget - 1]) {
    suffix.push(sameRow(base[endBase - 1]))
    endBase -= 1
    endTarget -= 1
  }
  suffix.reverse()

  const middle = diffMiddle(base.slice(start, endBase), target.slice(start, endTarget))
  return prefix.concat(middle, suffix)
}
