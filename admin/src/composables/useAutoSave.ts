import { onScopeDispose, ref, watch, type Ref } from 'vue'
import dayjs from 'dayjs'
import { App } from 'ant-design-vue'

export type DraftSaveStatus = 'idle' | 'saving' | 'saved-server' | 'saved-local' | 'error'

export type LocalDraft<TForm, TExtra extends object = Record<string, never>> = {
  form: TForm
  savedAt: number
} & TExtra

export interface UseAutoSaveOptions<TForm extends object, TExtra extends object> {
  /** 本地草稿 localStorage 键 */
  draftKey: Ref<string>
  /** 表单响应式对象(深度 watch) */
  form: TForm
  /** 变化后防抖毫秒(默认 3000) */
  debounceMs?: number
  /** 草稿完整负载构造(表单之外还有附加字段如定时时间时提供;缺省 form + savedAt) */
  draftPayload?: () => LocalDraft<TForm, TExtra>
  /** true 时本地写入后同时发起服务器保存;false/未提供则仅本地兜底 */
  canServerSave?: () => boolean
  /** 服务器保存执行(静默,失败 reject 不弹全局提示) */
  serverSave?: () => Promise<void>
  /** 服务器保存负载快照(与上次成功快照相同则跳过请求) */
  payloadSnapshot?: () => string
  /** 草稿与当前内容的对比快照(参与差异判断) */
  formSnapshot: () => string
  /** 新建场景(无服务器对照)下草稿是否值得恢复 */
  draftHasContent?: (draft: LocalDraft<TForm, TExtra>) => boolean
  /** 恢复草稿回填(页面把 draft.form 合回自身状态) */
  restore: (draft: LocalDraft<TForm, TExtra>) => void
  /** 手动保存进行中时跳过服务器自动保存 */
  isBusy?: () => boolean
  /** 恢复询问标题主体(如「文章」「页面」) */
  subject?: string
  /**
   * 服务器自动保存失败时的回调(在状态条置 error 之前调用)。
   * 页面据此实现业务级反馈:如 10005 冲突 → 暂停自动保存并提示用户。
   * 返回 true 表示页面已接管(pause 由页面自行决定),内核不再额外处理。
   */
  onServerError?: (error: unknown) => void
}

/**
 * 编辑器自动保存通用内核(docs/09 §7.1)
 * 机制:变更 → 防抖 → 本地草稿(localStorage 兜底)→ 条件满足时服务器静默保存
 * 状态机只驱动底部状态条;业务条件(发布态限制/必填校验)由页面通过回调注入。
 */
export function useAutoSave<TForm extends object, TExtra extends object = Record<string, never>>(
  options: UseAutoSaveOptions<TForm, TExtra>,
) {
  const { modal } = App.useApp()

  const autosaveReady = ref(false)
  const draftStatus = ref<DraftSaveStatus>('idle')
  const savedAtText = ref('')
  let draftTimer: ReturnType<typeof setTimeout> | null = null
  /** 服务器已同步的最新负载快照(跳过无变化的自动保存请求) */
  let lastServerSnapshot = ''
  /** 服务器自动保存序号:仅最新一次请求的结果可以更新状态条 */
  let autosaveSeq = 0

  function clearDraftTimer() {
    if (draftTimer) {
      clearTimeout(draftTimer)
      draftTimer = null
    }
  }

  function readDraft(): LocalDraft<TForm, TExtra> | null {
    try {
      const raw = localStorage.getItem(options.draftKey.value)
      if (!raw) return null
      const parsed = JSON.parse(raw) as LocalDraft<TForm, TExtra>
      if (!parsed || typeof parsed !== 'object' || !parsed.form) return null
      return parsed
    } catch {
      return null
    }
  }

  function writeDraft() {
    draftTimer = null
    // 1) 本地兜底始终写入:服务器自动保存失败时草稿仍在
    try {
      const draft: LocalDraft<TForm, TExtra> = options.draftPayload
        ? options.draftPayload()
        : ({ form: options.form, savedAt: Date.now() } as LocalDraft<TForm, TExtra>)
      localStorage.setItem(options.draftKey.value, JSON.stringify(draft))
    } catch {
      if (!canServer()) {
        draftStatus.value = 'error'
        return
      }
    }
    // 2) 条件满足时同时发起服务器自动保存
    if (canServer()) {
      void serverAutosave()
    } else {
      savedAtText.value = dayjs().format('HH:mm')
      draftStatus.value = 'saved-local'
    }
  }

  function canServer(): boolean {
    return Boolean(
      options.canServerSave &&
      options.serverSave &&
      options.payloadSnapshot &&
      autosaveReady.value &&
      !(options.isBusy?.() ?? false) &&
      options.canServerSave() &&
      options.payloadSnapshot() !== lastServerSnapshot,
    )
  }

  /** 服务器端自动保存:静默请求,结果只更新底部状态条,不回填表单(避免覆盖正在输入的内容) */
  async function serverAutosave() {
    if (!options.serverSave || !options.payloadSnapshot) return
    const seq = ++autosaveSeq
    const requestSnapshot = options.payloadSnapshot()
    try {
      await options.serverSave()
      if (seq !== autosaveSeq) return
      lastServerSnapshot = requestSnapshot
      savedAtText.value = dayjs().format('HH:mm')
      draftStatus.value = 'saved-server'
      // 服务器已接住,清理本地兜底草稿(下次变更会重新写入)
      localStorage.removeItem(options.draftKey.value)
    } catch (error) {
      // 静默失败:不弹错误提示,本地草稿已兜底;业务级处理交给页面回调
      if (seq !== autosaveSeq) return
      options.onServerError?.(error)
      draftStatus.value = 'error'
    }
  }

  function clearLocalDraft() {
    clearDraftTimer()
    localStorage.removeItem(options.draftKey.value)
    draftStatus.value = 'idle'
  }

  function onFormChange() {
    if (!autosaveReady.value) return
    draftStatus.value = 'saving'
    clearDraftTimer()
    draftTimer = setTimeout(writeDraft, options.debounceMs ?? 3000)
  }

  watch(() => options.form, onFormChange, { deep: true })

  /** 表单之外的字段(如定时时间)变化时由页面手动调用 */
  function touch() {
    onFormChange()
  }

  /** 载入后检查本地草稿:与当前内容不同则询问恢复
   *  differsFn 由页面提供(编辑场景拿草稿与服务器数据对比);缺省用 draftHasContent(新建场景) */
  function checkLocalDraft(differsFn?: (draft: LocalDraft<TForm, TExtra>) => boolean) {
    const draft = readDraft()
    if (!draft) {
      autosaveReady.value = true
      return
    }
    const differs = differsFn ? differsFn(draft) : (options.draftHasContent?.(draft) ?? true)
    if (!differs) {
      // 与当前内容一致,草稿无价值,直接清除
      clearLocalDraft()
      autosaveReady.value = true
      return
    }
    const savedAt = dayjs(draft.savedAt).format('HH:mm')
    modal.confirm({
      title: '发现未保存的本地草稿',
      content: `本地保存于 ${savedAt}，与当前内容不同，是否恢复？`,
      okText: '恢复',
      cancelText: '不恢复',
      onOk: () => {
        options.restore(draft)
        savedAtText.value = savedAt
        draftStatus.value = 'saved-local'
        autosaveReady.value = true
      },
      onCancel: () => {
        // 放弃恢复则丢弃本地草稿
        clearLocalDraft()
        autosaveReady.value = true
      },
    })
  }

  /** 手动保存成功后调用:更新服务器快照基线,避免下次自动保存重复请求 */
  function markServerSynced() {
    if (options.payloadSnapshot) {
      lastServerSnapshot = options.payloadSnapshot()
    }
  }

  /** 暂停自动保存(数据加载/手动保存/版本恢复期间),同时丢弃未落盘的防抖计时 */
  function pause() {
    autosaveReady.value = false
    clearDraftTimer()
  }

  /** 恢复自动保存 */
  function resume() {
    autosaveReady.value = true
  }

  /** 状态条回到初始态(加载中) */
  function markIdle() {
    draftStatus.value = 'idle'
  }

  /** 重置服务器快照基线(表单被服务器数据回填后调用) */
  function setBaseline() {
    if (options.payloadSnapshot) {
      lastServerSnapshot = options.payloadSnapshot()
    }
  }

  /** 立即进入自动保存流程(保存期间又有输入时把增量落盘) */
  function schedule() {
    onFormChange()
  }

  /** 离开页面前把未落盘的变更立即写入(在 onBeforeUnmount 调用) */
  function flushDraft() {
    if (draftTimer) {
      writeDraft()
    }
  }

  /** 立即把当前内容写入本地草稿(不等防抖)。用于冲突处理「载入最新」前暂存用户输入 */
  function writeNow() {
    clearDraftTimer()
    try {
      const draft: LocalDraft<TForm, TExtra> = options.draftPayload
        ? options.draftPayload()
        : ({ form: options.form, savedAt: Date.now() } as LocalDraft<TForm, TExtra>)
      localStorage.setItem(options.draftKey.value, JSON.stringify(draft))
    } catch {
      // 本地写入失败不阻断流程(如配额满/隐私模式)
    }
  }

  /** 是否有尚未落盘的防抖变更(供 beforeunload 判断是否需要拦截) */
  function hasPending(): boolean {
    return draftTimer !== null
  }

  // 页面隐藏/卸载兜底:防抖窗口内直接关页会丢内容,这里同步落盘。
  // pagehide 覆盖移动端切后台与 bfcache;beforeunload 仅在真有未落盘变更时拦截。
  function onPageHide() {
    flushDraft()
  }
  function onBeforeUnload(event: BeforeUnloadEvent) {
    if (!hasPending()) return
    flushDraft()
    event.preventDefault()
    // 部分浏览器需设置 returnValue 才展示离开确认
    event.returnValue = ''
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('pagehide', onPageHide)
    window.addEventListener('beforeunload', onBeforeUnload)
  }
  onScopeDispose(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('pagehide', onPageHide)
      window.removeEventListener('beforeunload', onBeforeUnload)
    }
    clearDraftTimer()
  })

  return {
    autosaveReady,
    draftStatus,
    savedAtText,
    touch,
    clearLocalDraft,
    checkLocalDraft,
    markServerSynced,
    pause,
    resume,
    markIdle,
    setBaseline,
    schedule,
    flushDraft,
    writeNow,
    hasPending,
  }
}
