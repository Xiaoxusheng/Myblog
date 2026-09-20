<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import dayjs, { type Dayjs } from 'dayjs'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import {
  ArrowLeftOutlined,
  EyeOutlined,
  HistoryOutlined,
  LinkOutlined,
} from '@ant-design/icons-vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import FormSection from '@/components/FormSection.vue'
import PageRevisionDrawer from '@/components/PageRevisionDrawer.vue'
import MediaSelectModal from '@/components/MediaSelectModal.vue'
import { useFeedback } from '@/composables/useFeedback'
import { useAutoSave } from '@/composables/useAutoSave'
import { adminTheme } from '@/theme'
import { SLUG_REQUIRED_PATTERN, SLUG_PATTERN_MESSAGE } from '@/utils/validators'
import { countWords, estimateReadingMinutes } from '@/utils/postCheck'
import { formatTime } from '@/utils/format'
import { createPage, getPage, getPagePreview, updatePage } from '@/api/pages'
import { silentUpdatePage, SilentRequestError, CODE_CONFLICT } from '@/utils/autosaveHttp'
import { getSettings } from '@/api/media'
import { PAGE_STATUS_MAP, PAGE_TYPE_LABELS } from '@/constants/status'
import type { PageItem, PagePayload, PageStatus, PageType } from '@/types/api'

const route = useRoute()
const router = useRouter()
const { message, modal } = useFeedback()

const pageId = computed<number | null>(() => {
  const raw = route.params.id
  return raw ? Number(raw) : null
})

const loading = ref(false)
/** 详情加载失败标记：驱动错误面板 + 重试，避免无数据时静默进入空表单 */
const loadFailed = ref(false)
const saving = ref(false)
const formRef = ref()

const formState = reactive({
  title: '',
  slug: '',
  content: '',
  status: 0 as PageStatus,
  pageType: 'default' as PageType,
  publishAt: null as Dayjs | null,
  seoTitle: '',
  seoDescription: '',
  canonical: '',
  ogImage: '',
})

/** 服务器已确认的并发基线（保存/自动保存成功后回填） */
const syncedUpdatedAt = ref<string | null>(null)
/** 服务器已知的 slug，用于判断「URL 变化」并提供 301 创建入口 */
const serverSlug = ref('')
/** slug 变更确认：是否已就「创建 301」做出选择 */
const slugChangeAcknowledged = ref(false)

const rules = {
  slug: [{ pattern: SLUG_REQUIRED_PATTERN, message: SLUG_PATTERN_MESSAGE }],
}

/* ---------------- 自动保存（复用 useAutoSave 内核） ---------------- */

const DRAFT_PREFIX = 'blog_admin_page_draft'
const draftKey = computed(() =>
  pageId.value ? `${DRAFT_PREFIX}_${pageId.value}` : `${DRAFT_PREFIX}_new`,
)

function buildPayload(status: PageStatus, withBaseline = true): PagePayload {
  const payload: PagePayload = {
    title: formState.title.trim(),
    slug: formState.slug.trim(),
    content: formState.content,
    status,
    pageType: formState.pageType,
    publishAt: formState.publishAt ? formState.publishAt.toISOString() : null,
    seoTitle: formState.seoTitle.trim(),
    seoDescription: formState.seoDescription.trim(),
    canonical: formState.canonical.trim(),
    ogImage: formState.ogImage.trim(),
  }
  if (withBaseline) payload.baseUpdatedAt = syncedUpdatedAt.value
  return payload
}

function snapshotOf(form: typeof formState): string {
  return JSON.stringify({
    ...form,
    title: form.title.trim(),
    slug: form.slug.trim(),
    publishAt: form.publishAt ? form.publishAt.toISOString() : null,
  })
}

const {
  draftStatus,
  savedAtText,
  touch: scheduleAutosave,
  clearLocalDraft,
  checkLocalDraft,
  pause: pauseAutosave,
  resume: resumeAutosave,
  markIdle: markAutosaveIdle,
  setBaseline,
  flushDraft,
  writeNow,
} = useAutoSave({
  draftKey,
  form: formState,
  formSnapshot: () => snapshotOf(formState),
  draftHasContent: (draft) => Boolean(draft.form.title.trim() || draft.form.content.trim()),
  restore: (draft) => {
    Object.assign(formState, draft.form)
  },
  subject: '页面',
  // 定时发布（status=3）也可静默保存；已发布（1）不实时改写线上内容
  canServerSave: () =>
    pageId.value !== null && formState.title.trim() !== '' && formState.status !== 1,
  serverSave: async () => {
    const result = await silentUpdatePage(pageId.value as number, buildPayload(formState.status))
    syncedUpdatedAt.value = result.updatedAt ?? syncedUpdatedAt.value
  },
  payloadSnapshot: () => JSON.stringify(buildPayload(formState.status)),
  isBusy: () => saving.value,
})

/* ---------------- 载入 / 回填 ---------------- */

function resetForm() {
  formState.title = ''
  formState.slug = ''
  formState.content = ''
  formState.status = 0
  formState.pageType = 'default'
  formState.publishAt = null
  formState.seoTitle = ''
  formState.seoDescription = ''
  formState.canonical = ''
  formState.ogImage = ''
  syncedUpdatedAt.value = null
  serverSlug.value = ''
}

function fillForm(page: PageItem) {
  formState.title = page.title
  formState.slug = page.slug
  formState.content = page.content
  formState.status = page.status
  formState.pageType = page.pageType ?? 'default'
  formState.publishAt = page.publishAt ? dayjs(page.publishAt) : null
  formState.seoTitle = page.seoTitle ?? ''
  formState.seoDescription = page.seoDescription ?? ''
  formState.canonical = page.canonical ?? ''
  formState.ogImage = page.ogImage ?? ''
  syncedUpdatedAt.value = page.updatedAt
  serverSlug.value = page.slug
}

/** 本地草稿时间晚于服务器内容时间才提示恢复（绝不无提示覆盖服务器内容） */
function draftNewerThanServer(draft: { savedAt?: number }): boolean {
  if (!syncedUpdatedAt.value) return true
  const serverTime = new Date(syncedUpdatedAt.value).getTime()
  const localTime = draft.savedAt ?? 0
  return localTime > serverTime
}

async function load() {
  // 载入期间必须先暂停自动保存并重置快照基线，否则 fillForm 触发的深度 watch
  // 会把「服务器数据回填」误判成用户编辑，debounce 后立刻发起一次无意义的静默保存
  // （表现为刚打开页面状态条就显示「正在保存…」，且每次打开都写库）。
  pauseAutosave()
  markAutosaveIdle()

  // 预览模式：未发布页面走管理端预览接口（公开接口只认已发布）
  const wantPreview = route.query.preview === '1'

  if (!pageId.value) {
    resetForm()
    setBaseline()
    resumeAutosave()
    checkLocalDraft()
    return
  }
  loading.value = true
  loadFailed.value = false
  try {
    const result = wantPreview ? await getPagePreview(pageId.value) : await getPage(pageId.value)
    fillForm(result.page)
    conflictDetected.value = false
    setBaseline()
    // 等编辑器挂载完成再恢复自动保存：md-editor-v3 挂载时会 emit 一次
    // update:modelValue，若此时 autosaveReady 已为 true，会被当成用户编辑而写入
    // 本地草稿（表现为刚打开页面状态条就显示「已保存至本地」）。
    await nextTick()
    resumeAutosave()
    // 仅当本地草稿比服务器更新、且内容确有差异时才询问
    checkLocalDraft(
      (draft) => draftNewerThanServer(draft) && snapshotOf(draft.form) !== snapshotOf(formState),
    )
    if (wantPreview) {
      previewMode.value = true
      await router.replace(`/pages/edit/${pageId.value}`)
    }
  } catch {
    // 详情拉取失败：展示错误面板 + 重试，而非静默停留在空白表单
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

/* ---------------- 保存 ---------------- */

async function save(nextStatus: PageStatus) {
  if (!formState.title.trim()) {
    message.error('请输入页面标题')
    return
  }
  if (nextStatus === 3 && !formState.publishAt) {
    message.error('请选择计划发布时间')
    return
  }
  await formRef.value?.validate()
  const payload = buildPayload(nextStatus)
  saving.value = true
  try {
    if (pageId.value) {
      const result = await updatePage(pageId.value, payload)
      syncedUpdatedAt.value = result.page.updatedAt ?? syncedUpdatedAt.value
      const oldServerSlug = serverSlug.value
      serverSlug.value = result.page.slug
      conflictDetected.value = false
      pauseAutosave()
      formState.status = nextStatus
      formState.slug = result.page.slug
      setBaseline()
      message.success(
        nextStatus === 3 ? '已加入定时发布计划' : nextStatus === 2 ? '已隐藏' : '保存成功',
      )
      clearLocalDraft()
      await nextTick()
      resumeAutosave()
      scheduleAutosave()
      // slug 真被改写（含后端追加后缀）→ 提示创建 301
      if (oldServerSlug && result.page.slug !== oldServerSlug) {
        promptSlugRedirect(oldServerSlug, result.page.slug)
      }
    } else {
      const result = await createPage(payload)
      message.success(nextStatus === 3 ? '已加入定时发布计划' : '创建成功')
      clearLocalDraft()
      await router.replace(`/pages/edit/${result.page.id}`)
    }
  } catch (error) {
    if (error instanceof SilentRequestError && error.code === CODE_CONFLICT) {
      conflictDetected.value = true
      pauseAutosave()
      void promptManualConflict()
      return
    }
    throw error
  } finally {
    saving.value = false
  }
}

/* ---------------- 并发冲突处理 ---------------- */

const conflictDetected = ref(false)

function promptManualConflict() {
  modal.confirm({
    title: '页面已在其他窗口被修改',
    content:
      '服务器上的内容比你打开时更新。选择「载入最新」会用服务器版本替换当前编辑内容（你的改动会暂存为本地草稿）；「强制覆盖」则以你当前的内容为准写入。',
    okText: '载入最新',
    cancelText: '强制覆盖（用我的内容）',
    okButtonProps: { type: 'primary' },
    onOk: () => resolveConflictReload(),
    onCancel: () => resolveConflictOverwrite(),
  })
}

async function resolveConflictReload() {
  if (!pageId.value) return
  writeNow()
  const result = await getPage(pageId.value)
  pauseAutosave()
  fillForm(result.page)
  setBaseline()
  clearLocalDraft()
  conflictDetected.value = false
  await nextTick()
  resumeAutosave()
  message.success('已载入服务器最新版本（你的改动已暂存为本地草稿）')
}

async function resolveConflictOverwrite() {
  if (!pageId.value) return
  saving.value = true
  try {
    const result = await updatePage(pageId.value, buildPayload(formState.status, false))
    syncedUpdatedAt.value = result.page.updatedAt ?? null
    serverSlug.value = result.page.slug
    setBaseline()
    clearLocalDraft()
    conflictDetected.value = false
    await nextTick()
    resumeAutosave()
    message.success('已用当前内容覆盖服务器版本')
  } finally {
    saving.value = false
  }
}

/* ---------------- 版本恢复 ---------------- */

async function onRestored(page: PageItem) {
  pauseAutosave()
  fillForm(page)
  setBaseline()
  clearLocalDraft()
  await nextTick()
  resumeAutosave()
}

/** slug 变化 → 301 重定向 */
/** 本地感知：slug 输入与服务器不一致 → 顶部提示条（发布前提醒） */
const slugChanged = computed(
  () =>
    pageId.value !== null &&
    formState.slug.trim() !== '' &&
    formState.slug.trim() !== serverSlug.value,
)

/** 设置里的自动重定向开关（缺省开启）。仅当开启时后端才会自动写 301。 */
const autoRedirectEnabled = ref(true)

async function loadSettings() {
  try {
    const result = await getSettings()
    autoRedirectEnabled.value = result.settings.autoRedirectOnSlugChange ?? true
  } catch {
    // 读不到设置时保持默认（后端缺省为开启），不阻塞编辑
  }
}

/**
 * 保存后 slug 确实变化（含后端追加后缀）时的提示。
 * 后端在 autoRedirectOnSlugChange 开启时会自动创建 /page/旧 → /page/新 301
 * （契约 #41），此处只做告知与直达入口，不重复创建。
 */
function promptSlugRedirect(oldSlug: string, newSlug: string) {
  const auto = autoRedirectEnabled.value
  modal.info({
    title: 'URL 已变化',
    content: auto
      ? `页面地址由 /page/${oldSlug} 变为 /page/${newSlug}，已自动创建 301 重定向，旧链接会跳转到新地址。`
      : `页面地址由 /page/${oldSlug} 变为 /page/${newSlug}。「slug 变更自动重定向」已关闭，如需保留旧链接请手动添加 301 重定向。`,
    okText: '去重定向管理',
    cancelText: '关闭',
    onOk: () => goRedirects(),
  })
}

/* ---------------- 预览 ---------------- */

const previewMode = ref(false)

/* ---------------- 版本历史 ---------------- */

const revisionOpen = ref(false)

/* ---------------- 底部状态栏 ---------------- */

const wordCount = computed(() => countWords(formState.content))
const readingMinutes = computed(() => estimateReadingMinutes(wordCount.value))

/* ---------------- SEO / AEO 检查（只显示真实检查结果，不伪造分数） ---------------- */

const seoChecks = computed(() => {
  const seoTitleLen = formState.seoTitle.trim().length
  const descLen = formState.seoDescription.trim().length
  return [
    { key: 'title', label: formState.title.trim() ? '标题存在' : '标题为空', ok: formState.title.trim() !== '' },
    {
      key: 'seoTitle',
      label: seoTitleLen > 0 ? `SEO 标题已设置（${seoTitleLen} 字）` : 'SEO 标题未设置（将用页面标题）',
      ok: seoTitleLen > 0 && seoTitleLen <= 60,
    },
    {
      key: 'seoDescription',
      label: descLen > 0 ? `SEO 描述已设置（${descLen} 字）` : 'SEO 描述为空',
      ok: descLen > 0 && descLen <= 160,
    },
    {
      key: 'canonical',
      label: formState.canonical.trim() ? 'Canonical 正常' : 'Canonical 未设置（将用默认规则）',
      ok: formState.canonical.trim() !== '',
    },
    { key: 'ogImage', label: formState.ogImage.trim() ? 'OG Image 已设置' : 'OG Image 未设置', ok: formState.ogImage.trim() !== '' },
    {
      key: 'published',
      label: formState.status === 1 ? '页面已发布' : '页面未发布（前台不可见）',
      ok: formState.status === 1,
    },
  ]
})

/** AEO：结构化标题 / 清晰段落 / FAQ —— 仅做可验证的文本检查 */
const aeoChecks = computed(() => {
  const hasH2 = /^##\s/m.test(formState.content)
  const paragraphs = formState.content.split(/\n{2,}/).filter((p) => p.trim().length > 40).length
  const hasFaq = /^##+\s*(FAQ|常见问题)/im.test(formState.content)
  return [
    { key: 'h2', label: hasH2 ? '正文含结构化小标题' : '缺少 ## 小标题', ok: hasH2 },
    { key: 'paragraph', label: paragraphs > 0 ? `有 ${paragraphs} 段清晰段落` : '缺少成形段落', ok: paragraphs > 0 },
    { key: 'faq', label: hasFaq ? '检测到 FAQ 小节' : '未检测到 FAQ 小节（可选）', ok: hasFaq },
  ]
})

/* ---------------- 页面类型 ---------------- */

const pageTypeOptions = Object.entries(PAGE_TYPE_LABELS).map(([value, label]) => ({ value, label }))

/* ---------------- 发布前检查：未发布时点「发布」先提示未完成项 ---------------- */
const publishHintOpen = ref(false)

function onPublishClick(nextStatus: PageStatus) {
  const blocking = seoChecks.value.filter((item) => !item.ok)
  if (nextStatus === 1 && blocking.length > 0) {
    publishHintOpen.value = true
    return
  }
  void save(nextStatus)
}

async function confirmPublishAnyway() {
  publishHintOpen.value = false
  await save(1)
}

/* ---------------- 媒体选择 ---------------- */

const ogModalOpen = ref(false)

/* ---------------- 生命周期 ---------------- */

function goBack() {
  void router.push('/pages')
}

function disabledDate(current: Dayjs): boolean {
  return current.isBefore(dayjs().startOf('day'))
}

function onPublishAtChange(value: Dayjs | string | null) {
  formState.publishAt = dayjs.isDayjs(value) ? value : null
}

function goRedirects() {
  void router.push('/redirects')
}

watch(pageId, () => void load(), { immediate: true })

onMounted(() => {
  void loadSettings()
})

onBeforeUnmount(() => {
  flushDraft()
})
</script>

<template>
  <div class="page page-edit">
    <!-- 顶部固定操作栏：返回 / 标题 / 当前状态 / 预览·保存草稿·发布 -->
    <div class="page-edit__topbar">
      <a-button type="text" class="page-edit__back" @click="goBack">
        <template #icon><ArrowLeftOutlined /></template>
      </a-button>
      <div class="page-edit__title-box">
        <a-input v-model:value="formState.title" class="page-edit__title" placeholder="请输入页面标题" />
      </div>
      <a-tag class="page-edit__status" :color="PAGE_STATUS_MAP[formState.status].color">
        {{ PAGE_STATUS_MAP[formState.status].text }}
      </a-tag>
      <a-space class="page-edit__actions">
        <a-button @click="previewMode = !previewMode">
          <template #icon><EyeOutlined /></template>
          {{ previewMode ? '编辑' : '预览' }}
        </a-button>
        <a-button v-if="pageId" @click="revisionOpen = true">
          <template #icon><HistoryOutlined /></template>
          版本历史
        </a-button>
        <a-button :loading="saving" @click="save(0)">保存草稿</a-button>
        <a-button
          type="primary"
          :loading="saving"
          @click="
            formState.status === 3 || formState.publishAt ? onPublishClick(3) : onPublishClick(1)
          "
        >
          {{ formState.status === 3 ? '保存计划' : '发布' }}
        </a-button>
      </a-space>
    </div>

    <!-- slug 变化提醒（发布前告知，不阻断） -->
    <a-alert
      v-if="slugChanged && !slugChangeAcknowledged"
      class="page-edit__alert"
      type="warning"
      show-icon
      message="检测到 URL 变化"
      :description="`页面地址将由 /page/${serverSlug} 变为 /page/${formState.slug.trim()}。保存后旧链接会失效，建议创建 301 重定向。`"
    >
      <template #action>
        <a-space>
          <a-button size="small" @click="goRedirects">
            <template #icon><LinkOutlined /></template>
            重定向管理
          </a-button>
          <a-button size="small" type="text" @click="slugChangeAcknowledged = true">知道了</a-button>
        </a-space>
      </template>
    </a-alert>

    <!-- 并发冲突提示 -->
    <a-alert
      v-if="conflictDetected"
      class="page-edit__alert"
      type="warning"
      show-icon
      message="页面已在其他窗口被修改"
      description="自动保存已暂停。请选择「载入最新」获取服务器版本，或用「强制覆盖」保留你的内容。"
    >
      <template #action>
        <a-space>
          <a-button size="small" @click="resolveConflictReload">载入最新</a-button>
          <a-button size="small" danger @click="resolveConflictOverwrite">强制覆盖</a-button>
        </a-space>
      </template>
    </a-alert>

    <!-- 未完成项提示（不阻断发布） -->
    <a-alert
      v-if="publishHintOpen"
      class="page-edit__alert"
      type="info"
      show-icon
      closable
      message="以下 SEO 项目尚未完成"
      :description="seoChecks.filter((i) => !i.ok).map((i) => i.label).join('、')"
      @close="publishHintOpen = false"
    >
      <template #action>
        <a-button size="small" @click="confirmPublishAnyway">仍然发布</a-button>
      </template>
    </a-alert>

    <!-- 详情加载失败：给明确重试入口，避免在空表单上误编辑 -->
    <a-card v-if="loadFailed" class="page-edit__alert">
      <a-result
        status="warning"
        title="页面内容加载失败"
        sub-title="无法获取该页面数据，请确认服务已启动后重试"
      >
        <template #extra>
          <a-button type="primary" @click="load">重新加载</a-button>
          <a-button @click="goBack">返回列表</a-button>
        </template>
      </a-result>
    </a-card>

    <a-spin v-else :spinning="loading">
      <a-form ref="formRef" :model="formState" :rules="rules" layout="vertical">
        <div class="page-edit__body">
          <!-- 主区 70%：正文编辑器 / 预览 -->
          <div class="page-edit__main">
            <div v-if="previewMode" class="page-edit__preview">
              <div class="page-edit__preview-head">
                <h1 class="page-edit__preview-title">{{ formState.title || '未命名页面' }}</h1>
                <span class="page-edit__preview-path">/page/{{ formState.slug || '—' }}</span>
              </div>
              <MdPreview
                :model-value="formState.content"
                :theme="adminTheme === 'dark' ? 'dark' : 'light'"
                language="zh-CN"
                no-katex
                no-mermaid
              />
            </div>
            <div v-else class="page-edit__editor">
              <MarkdownEditor
                v-model="formState.content"
                placeholder="请输入 Markdown 内容，支持粘贴图片自动上传"
              />
            </div>

            <!-- 底部状态栏：字数 / 阅读时长 / 保存状态 -->
            <div class="page-edit__statusbar">
              <span class="page-edit__status-metrics tabular-nums">
                {{ wordCount }} 字 · 约 {{ readingMinutes }} 分钟
              </span>
              <span
                class="page-edit__status-save"
                :class="{ 'page-edit__status-save--error': draftStatus === 'error' }"
              >
                <template v-if="draftStatus === 'saving'">正在保存…</template>
                <template v-else-if="draftStatus === 'saved-server'">已保存于 {{ savedAtText }}</template>
                <template v-else-if="draftStatus === 'saved-local'">
                  已保存至本地 {{ savedAtText }}
                </template>
                <template v-else-if="draftStatus === 'error'">保存失败（已存本地草稿）</template>
                <template v-else>自动保存已开启</template>
              </span>
            </div>
          </div>

          <!-- 右栏 30%（320px）：页面设置 -->
          <aside class="page-edit__aside">
            <a-card :bordered="false">
              <FormSection title="基础信息" anchor="slug">
                <a-form-item label="Slug" name="slug">
                  <a-input v-model:value="formState.slug" placeholder="如 about，作为访问路径" />
                </a-form-item>
                <p class="page-edit__hint">
                  发布后通过 /page/{{ formState.slug || '&lt;slug&gt;' }} 访问；修改会提示创建 301。
                </p>
                <a-form-item label="页面类型">
                  <a-select v-model:value="formState.pageType" :options="pageTypeOptions" />
                </a-form-item>
              </FormSection>
            </a-card>

            <a-card :bordered="false">
              <FormSection title="发布设置">
                <a-form-item label="状态" name="status">
                  <a-radio-group v-model:value="formState.status">
                    <a-radio :value="0">草稿</a-radio>
                    <a-radio :value="1">已发布</a-radio>
                    <a-radio :value="2">隐藏</a-radio>
                    <a-radio :value="3">定时发布</a-radio>
                  </a-radio-group>
                </a-form-item>
                <a-form-item v-if="formState.status === 3" label="计划发布时间" required>
                  <a-date-picker
                    :value="formState.publishAt ?? undefined"
                    show-time
                    format="YYYY-MM-DD HH:mm"
                    placeholder="选择到点自动发布的时间"
                    style="width: 100%"
                    :disabled-date="disabledDate"
                    @update:value="onPublishAtChange"
                  />
                  <p class="page-edit__hint">到点由服务器自动上线，无需保持浏览器打开。</p>
                </a-form-item>
              </FormSection>
            </a-card>

            <a-card :bordered="false">
              <FormSection title="内容" anchor="pageType">
                <a-descriptions :column="1" size="small" class="page-edit__stats">
                  <a-descriptions-item label="正文字数">
                    <span class="tabular-nums">{{ wordCount }}</span>
                  </a-descriptions-item>
                  <a-descriptions-item label="预计阅读">
                    <span class="tabular-nums">约 {{ readingMinutes }} 分钟</span>
                  </a-descriptions-item>
                  <a-descriptions-item label="最近保存">
                    {{ pageId && syncedUpdatedAt ? formatTime(syncedUpdatedAt) : '—' }}
                  </a-descriptions-item>
                </a-descriptions>
                <p class="page-edit__hint">
                  正文即主区 Markdown 内容；SEO 描述留空时前台会截取正文开头。
                </p>
              </FormSection>
            </a-card>

            <a-card :bordered="false">
              <FormSection title="SEO" anchor="seoTitle">
                <a-form-item label="SEO 标题" :extra="`${formState.seoTitle.length}/60 字（建议 ≤60）`">
                  <a-input
                    v-model:value="formState.seoTitle"
                    placeholder="留空则使用页面标题"
                    :maxlength="200"
                  />
                </a-form-item>
                <a-form-item
                  label="SEO 描述"
                  :extra="`${formState.seoDescription.length}/160 字（建议 ≤160）`"
                >
                  <a-textarea
                    v-model:value="formState.seoDescription"
                    placeholder="留空则截取正文开头"
                    :rows="3"
                    :maxlength="300"
                  />
                </a-form-item>
                <a-form-item label="Canonical URL">
                  <a-input v-model:value="formState.canonical" placeholder="留空使用默认规则" allow-clear />
                </a-form-item>
                <a-form-item label="OG Image">
                  <a-input
                    v-model:value="formState.ogImage"
                    placeholder="留空则不输出 OG 图"
                    allow-clear
                  >
                    <template #addonAfter>
                      <a class="page-edit__media-link" @click="ogModalOpen = true">选择</a>
                    </template>
                  </a-input>
                </a-form-item>

                <!-- SEO 检查：只显示真实检查结果，不给分数 -->
                <div class="page-edit__checks">
                  <div class="page-edit__checks-title">SEO 检查</div>
                  <ul class="checklist">
                    <li v-for="item in seoChecks" :key="item.key" :class="{ 'check-blocked': !item.ok }">
                      <span class="check-mark">{{ item.ok ? '✓' : '⚠' }}</span>
                      {{ item.label }}
                    </li>
                  </ul>
                </div>

                <!-- AEO 检查 -->
                <div class="page-edit__checks">
                  <div class="page-edit__checks-title">AEO / 结构化数据</div>
                  <ul class="checklist">
                    <li v-for="item in aeoChecks" :key="item.key" :class="{ 'check-blocked': !item.ok }">
                      <span class="check-mark">{{ item.ok ? '✓' : '⚠' }}</span>
                      {{ item.label }}
                    </li>
                  </ul>
                  <p class="page-edit__hint">
                    结构化数据按页面类型生成（WebPage / AboutPage / ContactPage）；检测到 FAQ 小节时输出 FAQPage。
                  </p>
                </div>
              </FormSection>
            </a-card>

            <a-card :bordered="false">
              <FormSection title="版本">
                <a-button block :disabled="!pageId" @click="revisionOpen = true">
                  <template #icon><HistoryOutlined /></template>
                  版本历史
                </a-button>
                <p class="page-edit__hint">每次保存内容变化自动生成版本，可查看、对比与恢复。</p>
              </FormSection>
            </a-card>
          </aside>
        </div>
      </a-form>
    </a-spin>

    <MediaSelectModal v-model:open="ogModalOpen" @select="formState.ogImage = $event" />

    <PageRevisionDrawer
      v-model:open="revisionOpen"
      :page-id="pageId"
      :current-content="formState.content"
      @restored="onRestored"
    />
  </div>
</template>

<style scoped>
.page-edit {
  max-width: 1600px;
}

/* 顶部固定操作栏 */
.page-edit__topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  margin-bottom: 8px;
  background: var(--admin-bg);
}

.page-edit__back {
  flex: none;
}

.page-edit__alert {
  margin-bottom: 12px;
}

.page-edit__alert :deep(.ant-alert-action) {
  margin-inline-start: 12px;
}

.page-edit__title-box {
  flex: 1;
  min-width: 200px;
  border-bottom: 1px solid transparent;
  transition: border-color 0.18s ease;
}

.page-edit__title-box:focus-within {
  border-bottom-color: var(--admin-border);
}

.page-edit__title.ant-input,
.page-edit__title.ant-input:hover,
.page-edit__title.ant-input:focus,
.page-edit__title.ant-input-focused {
  padding: 0 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--admin-text);
  border: none;
  box-shadow: none;
}

.page-edit__title::placeholder {
  font-weight: 400;
  color: var(--admin-muted);
}

.page-edit__status {
  flex: none;
}

.page-edit__actions {
  flex: none;
}

/* 左右两栏：主区 70% / 右栏 320px；<1200 右栏折到正文下方 */
.page-edit__body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 16px;
  align-items: start;
}

.page-edit__aside {
  display: grid;
  gap: 16px;
}

.page-edit__hint {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

.page-edit__media-link {
  cursor: pointer;
}

/* 编辑器外壳 + 底部状态栏（32px） */
.page-edit__editor {
  border: 1px solid var(--admin-border);
  border-bottom: none;
  border-radius: var(--admin-radius-md) var(--admin-radius-md) 0 0;
  overflow: hidden;
}

.page-edit__editor :deep(.md-editor) {
  border: none;
  box-shadow: none;
  border-radius: 0;
}

/* 预览：复用前台页面观感 */
.page-edit__preview {
  min-height: 520px;
  padding: 24px 32px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-bottom: none;
  border-radius: var(--admin-radius-md) var(--admin-radius-md) 0 0;
}

.page-edit__preview-head {
  padding-bottom: 12px;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--admin-border);
}

.page-edit__preview-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: var(--admin-text);
}

.page-edit__preview-path {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--admin-muted);
}

.page-edit__statusbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  height: 32px;
  padding: 0 12px;
  font-size: 12px;
  color: var(--admin-muted);
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: 0 0 var(--admin-radius-md) var(--admin-radius-md);
}

.page-edit__status-save--error {
  color: var(--admin-danger);
}

/* 检查清单 */
.page-edit__checks {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--admin-border);
}

.page-edit__checks-title {
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-muted);
}

.checklist {
  margin: 0;
  padding: 0;
  list-style: none;
}

.checklist li {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 2px 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-text);
}

.checklist li.check-blocked {
  color: var(--admin-warning);
}

.check-mark {
  flex: none;
}

@media (max-width: 1199px) {
  .page-edit__body {
    grid-template-columns: 1fr;
  }
}
</style>
