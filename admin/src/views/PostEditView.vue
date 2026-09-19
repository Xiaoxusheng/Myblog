<script setup lang="ts">
import { SLUG_PATTERN, SLUG_PATTERN_MESSAGE } from '@/utils/validators'
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
const { message } = useFeedback()
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import {
  ArrowLeftOutlined,
  ClearOutlined,
  HistoryOutlined,
  PictureOutlined,
} from '@ant-design/icons-vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import FormSection from '@/components/FormSection.vue'
import MediaSelectModal from '@/components/MediaSelectModal.vue'
import PostRevisionDrawer from '@/components/PostRevisionDrawer.vue'
import { createPost, getPost, updatePost } from '@/api/posts'
import { getCategories } from '@/api/taxonomy'
import { getSeriesList } from '@/api/series'
import { silentUpdatePost } from '@/utils/autosaveHttp'
import { useAutoSave } from '@/composables/useAutoSave'
import type { AdminPostItem, Category, PostPayload, PostStatus, Series } from '@/types/api'

const route = useRoute()
const router = useRouter()

const postId = computed<number | null>(() => {
  const raw = route.params.id
  return raw ? Number(raw) : null
})

const loading = ref(false)
const saving = ref(false)
const categories = ref<Category[]>([])
const seriesOptions = ref<Series[]>([])
const coverModalOpen = ref(false)
const revisionOpen = ref(false)

/** 定时发布计划时间（status=3 时使用） */
const publishAtValue = ref<Dayjs | null>(null)

const formState = reactive({
  title: '',
  slug: '',
  categoryId: null as number | null,
  tagNames: [] as string[],
  isTop: false,
  status: 0 as PostStatus,
  summary: '',
  cover: '',
  content: '',
  /** 所属专题；0 = 不属于任何专题 */
  seriesId: 0,
  /** 专题内序号；null = 自动排到专题末尾（提交时按契约转为 0） */
  seriesSort: null as number | null,
  seoTitle: '',
  seoDescription: '',
  canonical: '',
  ogImage: '',
})

const formRef = ref()

const rules = {
  categoryId: [{ required: true, message: '请选择分类' }],
  slug: [
    {
      pattern: SLUG_PATTERN,
      message: SLUG_PATTERN_MESSAGE,
    },
  ],
}

// ---------------------------------------------------------------------------
// 自动保存：3s 防抖 → 本地草稿（localStorage 兜底）+ 条件满足时服务器自动保存
// 服务器自动保存仅针对未发布文章（status !== 1），已发布文章的未定稿编辑只留本地
// 机制在 composables/useAutoSave.ts，页面只注入业务条件与快照（docs/09 §7.1）
// ---------------------------------------------------------------------------
/** 定时发布计划时间（ISO 字符串）；旧草稿无此字段 */
type PostDraftExtra = { publishAt?: string | null }

const DRAFT_PREFIX = 'blog_admin_post_draft'

const draftKey = computed(() =>
  postId.value ? `${DRAFT_PREFIX}_${postId.value}` : `${DRAFT_PREFIX}_new`,
)

function currentPublishAtIso(): string | null {
  return publishAtValue.value ? publishAtValue.value.toISOString() : null
}

/** 表单快照（tags 排序后比较，避免服务端顺序差异误报）；publishAt/专题归属参与对比 */
function formSnapshot(form: typeof formState, publishAt?: string | null): string {
  return JSON.stringify({
    title: form.title,
    slug: form.slug,
    categoryId: form.categoryId,
    tagNames: [...form.tagNames].sort(),
    isTop: form.isTop,
    status: form.status,
    summary: form.summary,
    cover: form.cover,
    content: form.content,
    publishAt: publishAt ?? null,
    seriesId: form.seriesId,
    seriesSort: form.seriesSort,
    seoTitle: form.seoTitle,
    seoDescription: form.seoDescription,
    canonical: form.canonical,
    ogImage: form.ogImage,
  })
}

const {
  draftStatus,
  savedAtText,
  touch: touchAutosave,
  clearLocalDraft,
  checkLocalDraft,
  pause: pauseAutosave,
  resume: resumeAutosave,
  markIdle: markAutosaveIdle,
  setBaseline,
  schedule: scheduleAutosave,
  flushDraft,
} = useAutoSave<typeof formState, PostDraftExtra>({
  draftKey,
  form: formState,
  draftPayload: () => ({
    form: { ...formState, tagNames: [...formState.tagNames] },
    savedAt: Date.now(),
    publishAt: currentPublishAtIso(),
  }),
  formSnapshot: () => formSnapshot(formState, currentPublishAtIso()),
  draftHasContent: (draft) =>
    Boolean(
      draft.form.title.trim() ||
        draft.form.content.trim() ||
        draft.form.summary.trim() ||
        draft.form.tagNames.length > 0,
    ),
  restore: (draft) => {
    Object.assign(formState, draft.form, { tagNames: [...draft.form.tagNames] })
    publishAtValue.value = draft.publishAt ? dayjs(draft.publishAt) : null
  },
  subject: '文章',
  canServerSave: () =>
    postId.value !== null &&
    formState.title.trim() !== '' &&
    formState.categoryId !== null &&
    formState.status !== 1 &&
    (formState.status !== 3 || publishAtValue.value !== null),
  serverSave: () => silentUpdatePost(postId.value as number, buildPayload(formState.status)),
  payloadSnapshot: () => JSON.stringify(buildPayload(formState.status)),
  isBusy: () => saving.value,
})

watch(publishAtValue, () => touchAutosave())

// ---------------------------------------------------------------------------
// 数据加载与保存
// ---------------------------------------------------------------------------

async function load() {
  pauseAutosave()
  markAutosaveIdle()
  if (!postId.value) {
    // 新建：重置表单
    formRef.value?.resetFields()
    formState.title = ''
    formState.slug = ''
    formState.categoryId = null
    formState.tagNames = []
    formState.isTop = false
    formState.status = 0
    formState.summary = ''
    formState.cover = ''
    formState.content = ''
    formState.seriesId = 0
    formState.seriesSort = null
    publishAtValue.value = null
    setBaseline()
    checkLocalDraft()
    return
  }
  loading.value = true
  try {
    const result = await getPost(postId.value)
    fillForm(result.post)
    setBaseline()
    checkLocalDraft((draft) => formSnapshot(draft.form, draft.publishAt) !== formSnapshot(formState, currentPublishAtIso()))
  } finally {
    loading.value = false
  }
}

function fillForm(post: AdminPostItem) {
  formState.title = post.title
  formState.slug = post.slug
  formState.categoryId = post.categoryId
  formState.tagNames = [...post.tagNames]
  formState.isTop = post.isTop
  formState.status = post.status
  formState.summary = post.summary
  formState.cover = post.cover
  formState.content = post.content
  formState.seoTitle = post.seoTitle ?? ''
  formState.seoDescription = post.seoDescription ?? ''
  formState.canonical = post.canonical ?? ''
  formState.ogImage = post.ogImage ?? ''
  formState.seriesId = post.seriesId ?? 0
  formState.seriesSort =
    typeof post.seriesSort === 'number' && post.seriesSort > 0 ? post.seriesSort : null
  publishAtValue.value = post.publishAt ? dayjs(post.publishAt) : null
}

function buildPayload(nextStatus: PostStatus): PostPayload {
  return {
    title: formState.title.trim(),
    slug: formState.slug.trim(),
    summary: formState.summary.trim(),
    content: formState.content,
    cover: formState.cover.trim(),
    categoryId: formState.categoryId as number,
    tags: [...formState.tagNames],
    status: nextStatus,
    isTop: formState.isTop,
    publishAt:
      nextStatus === 3 && publishAtValue.value ? publishAtValue.value.toISOString() : undefined,
    seriesId: formState.seriesId || 0,
    seriesSort: formState.seriesSort ?? 0,
    seoTitle: formState.seoTitle.trim(),
    seoDescription: formState.seoDescription.trim(),
    canonical: formState.canonical.trim(),
    ogImage: formState.ogImage.trim(),
  }
}

async function save(nextStatus: PostStatus) {
  if (!formState.title.trim()) {
    message.error('请输入文章标题')
    return
  }
  if (nextStatus === 3 && !publishAtValue.value) {
    message.error('请选择计划发布时间')
    return
  }
  await formRef.value?.validate()
  const payload = buildPayload(nextStatus)
  saving.value = true
  try {
    if (postId.value) {
      await updatePost(postId.value, payload)
      // 暂停自动保存，避免状态回填触发一次多余的保存
      pauseAutosave()
      formState.status = nextStatus
      setBaseline()
      message.success(nextStatus === 3 ? '已加入定时发布计划' : '保存成功')
      // 保存成功，本地草稿已同步到服务器，清除
      clearLocalDraft()
      await nextTick()
      resumeAutosave()
      // 保存期间又有输入：重新进入自动保存流程，把增量落盘
      scheduleAutosave()
    } else {
      const result = await createPost(payload)
      message.success(nextStatus === 3 ? '已加入定时发布计划' : '创建成功')
      // 在路由切到编辑模式前清除 _new 草稿（draftKey 依赖当前路由）
      clearLocalDraft()
      // 转入编辑模式，后续保存走 PUT
      await router.replace(`/posts/edit/${result.post.id}`)
    }
  } finally {
    saving.value = false
  }
}

/** 版本恢复成功：用返回的 post 回填表单，清空本地草稿；暂停自动保存避免恢复内容再触发保存 */
async function onRestored(post: AdminPostItem) {
  pauseAutosave()
  fillForm(post)
  setBaseline()
  clearLocalDraft()
  await nextTick()
  resumeAutosave()
}

/** 定时发布日期禁选今天之前 */
function disabledDate(current: Dayjs): boolean {
  return current.isBefore(dayjs().startOf('day'))
}

/** a-date-picker 的 update:value 可能是 string/null，统一收敛为 Dayjs | null */
function onPublishAtChange(value: Dayjs | string | null) {
  publishAtValue.value = dayjs.isDayjs(value) ? value : null
}

function goBack() {
  void router.push('/posts')
}

watch(postId, () => void load(), { immediate: true })

async function loadCategories() {
  const result = await getCategories({ page: 1, pageSize: 100 })
  categories.value = result.list
}
void loadCategories()

async function loadSeriesOptions() {
  const result = await getSeriesList({ page: 1, pageSize: 50 })
  seriesOptions.value = result.list
}
void loadSeriesOptions()

/** 专题 select：清空（allowClear）时归 0，表示移出专题 */
function onSeriesChange(value: unknown) {
  formState.seriesId = typeof value === 'number' ? value : 0
  if (!formState.seriesId) {
    formState.seriesSort = null
  }
}

/** 序号输入：清空或非法值归 null，表示自动排到专题末尾 */
function onSeriesSortChange(value: number | string | null) {
  formState.seriesSort = typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : null
}

// ---------------------------------------------------------------------------
// 底部状态栏：字数 / 预计阅读时长（中文 400 字/分钟）
// ---------------------------------------------------------------------------
const wordCount = computed(() => formState.content.length)
const readingMinutes = computed(() =>
  wordCount.value === 0 ? 0 : Math.max(1, Math.ceil(wordCount.value / 400)),
)

// ---------- SEO 检查项（明确可验证项，不做虚假评分） ----------
const seoChecks = computed(() => {
  const seoTitleLen = formState.seoTitle.trim().length
  const descLen = (formState.seoDescription.trim() || formState.summary.trim()).length
  return [
    { label: `标题存在（${formState.title.trim().length} 字）`, ok: formState.title.trim() !== '' },
    {
      label: seoTitleLen > 0 ? `SEO 标题已设置（${seoTitleLen} 字）` : 'SEO 标题未设置（将用文章标题）',
      ok: seoTitleLen > 0 && seoTitleLen <= 60,
    },
    {
      label: descLen > 0 ? `描述已设置（${descLen} 字）` : '描述未设置（将用摘要或正文开头）',
      ok: descLen > 0 && descLen <= 160,
    },
    { label: '封面已设置', ok: formState.cover.trim() !== '' },
    { label: '摘要已设置', ok: formState.summary.trim() !== '' },
    { label: '正文含 H2 小节', ok: /^##\s/m.test(formState.content) },
    {
      label: formState.canonical.trim() !== '' ? 'Canonical 已设置' : 'Canonical 未设置（将用默认规则）',
      ok: formState.canonical.trim() !== '',
    },
  ]
})


// 离开页面前把未落盘的变更立即写入，避免丢失
onBeforeUnmount(() => {
  flushDraft()
})
</script>

<template>
  <div class="page post-edit">
    <!-- 顶部工具条：返回 + 无框化标题 + 保存动作 -->
    <div class="post-edit__topbar">
      <a-button type="text" class="post-edit__back" @click="goBack">
        <template #icon><ArrowLeftOutlined /></template>
      </a-button>
      <div class="post-edit__title-box">
        <a-input
          v-model:value="formState.title"
          class="post-edit__title"
          placeholder="请输入文章标题"
        />
      </div>
      <a-space class="post-edit__actions">
        <a-button v-if="postId" @click="revisionOpen = true">
          <template #icon><HistoryOutlined /></template>
          版本历史
        </a-button>
        <a-button :loading="saving" @click="save(0)">保存草稿</a-button>
        <a-button
          type="primary"
          :loading="saving"
          @click="formState.status === 3 ? save(3) : save(1)"
        >
          {{ formState.status === 3 ? '定时发布' : '发布' }}
        </a-button>
      </a-space>
    </div>

    <a-spin :spinning="loading">
      <a-form ref="formRef" :model="formState" :rules="rules" layout="vertical">
        <div class="post-edit__body">
          <!-- 左栏：编辑器 + 底部状态栏 -->
          <div class="post-edit__main">
            <div class="post-edit__editor">
              <MarkdownEditor
                v-model="formState.content"
                placeholder="请输入 Markdown 内容，支持粘贴图片自动上传"
              />
            </div>
            <div class="post-edit__statusbar">
              <span class="post-edit__status-metrics tabular-nums">
                字数 {{ wordCount }} · 预计阅读约 {{ readingMinutes }} 分钟
              </span>
              <span class="post-edit__status-save" :class="{ 'post-edit__status-save--error': draftStatus === 'error' }">
                <template v-if="draftStatus === 'saving'">正在保存…</template>
                <template v-else-if="draftStatus === 'saved-server'">已保存 {{ savedAtText }}</template>
                <template v-else-if="draftStatus === 'saved-local'">已保存至本地 {{ savedAtText }}</template>
                <template v-else-if="draftStatus === 'error'">保存失败</template>
                <template v-else>自动保存已开启</template>
              </span>
            </div>
          </div>

          <!-- 右栏：分组卡片 -->
          <aside class="post-edit__aside">
            <FormSection title="发布">
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
                  :value="publishAtValue ?? undefined"
                  show-time
                  format="YYYY-MM-DD HH:mm"
                  placeholder="选择到点自动发布的时间"
                  style="width: 100%"
                  :disabled-date="disabledDate"
                  @update:value="onPublishAtChange"
                />
                <p class="post-edit__hint">到点由服务器自动上线，到点前可在文章列表查看。</p>
              </a-form-item>
              <a-form-item label="置顶" name="isTop">
                <a-switch v-model:checked="formState.isTop" />
              </a-form-item>
              <p class="post-edit__hint">
                右上角「发布」将文章公开；「保存草稿」仅保存内容不发布。
              </p>
            </FormSection>

            <FormSection title="分类与标签">
              <a-form-item label="分类" name="categoryId" required>
                <a-select
                  v-model:value="formState.categoryId"
                  placeholder="请选择分类"
                  :options="categories.map((item) => ({ label: item.name, value: item.id }))"
                />
              </a-form-item>
              <a-form-item label="标签" name="tagNames">
                <a-select
                  v-model:value="formState.tagNames"
                  mode="tags"
                  placeholder="输入后回车添加标签"
                  :token-separators="[',']"
                />
              </a-form-item>
              <a-form-item label="Slug" name="slug">
                <a-input v-model:value="formState.slug" placeholder="留空自动生成" />
              </a-form-item>
            </FormSection>

            <FormSection title="专题">
              <a-form-item label="所属专题" name="seriesId">
                <a-select
                  :value="formState.seriesId || undefined"
                  :options="seriesOptions.map((item) => ({ label: item.name, value: item.id }))"
                  placeholder="不属于任何专题"
                  allow-clear
                  show-search
                  option-filter-prop="label"
                  @change="onSeriesChange"
                />
              </a-form-item>
              <a-form-item label="专题内序号" name="seriesSort">
                <a-input-number
                  :value="formState.seriesSort ?? undefined"
                  :min="1"
                  :max="9999"
                  :disabled="!formState.seriesId"
                  placeholder="自动"
                  style="width: 100%"
                  @change="onSeriesSortChange"
                />
              </a-form-item>
              <p class="post-edit__hint">留空自动排到专题末尾；清空专题即移出。</p>
            </FormSection>

            <FormSection title="封面">
              <a-space direction="vertical" :size="8" style="width: 100%">
                <a-input
                  v-model:value="formState.cover"
                  placeholder="输入图片外链地址"
                  allow-clear
                />
                <a-space wrap>
                  <a-button size="small" @click="coverModalOpen = true">
                    <template #icon><PictureOutlined /></template>
                    从媒体库选择
                  </a-button>
                  <a-button v-if="formState.cover" size="small" @click="formState.cover = ''">
                    <template #icon><ClearOutlined /></template>
                    清除
                  </a-button>
                </a-space>
                <a-image v-if="formState.cover" :src="formState.cover" :width="160" />
              </a-space>
            </FormSection>

            <FormSection title="摘要">
              <a-textarea
                v-model:value="formState.summary"
                placeholder="留空则前台可能截取正文开头"
                :rows="3"
                :maxlength="500"
                show-count
              />
            </FormSection>

            <FormSection title="SEO">
              <a-form-item label="SEO 标题" :extra="`当前 ${formState.seoTitle.length}/60 字（建议 ≤60）`">
                <a-input
                  v-model:value="formState.seoTitle"
                  placeholder="留空则使用文章标题"
                  :maxlength="200"
                />
              </a-form-item>
              <a-form-item
                label="SEO 描述"
                :extra="`当前 ${formState.seoDescription.length}/160 字（建议 ≤160）`"
              >
                <a-textarea
                  v-model:value="formState.seoDescription"
                  placeholder="留空则使用摘要"
                  :rows="2"
                  :maxlength="300"
                />
              </a-form-item>
              <a-form-item label="Canonical URL">
                <a-input v-model:value="formState.canonical" placeholder="留空使用默认规则" allow-clear />
              </a-form-item>
              <a-form-item label="OG 图">
                <a-input v-model:value="formState.ogImage" placeholder="留空使用封面" allow-clear />
              </a-form-item>
              <ul class="seo-checklist">
                <li v-for="check in seoChecks" :key="check.label" :class="{ 'seo-check--warn': !check.ok }">
                  <span>{{ check.ok ? '✓' : '⚠' }}</span>
                  {{ check.label }}
                </li>
              </ul>
            </FormSection>
          </aside>
        </div>
      </a-form>
    </a-spin>

    <MediaSelectModal v-model:open="coverModalOpen" @select="formState.cover = $event" />

    <PostRevisionDrawer
      v-model:open="revisionOpen"
      :post-id="postId"
      :current-content="formState.content"
      @restored="onRestored"
    />
  </div>
</template>

<style scoped>
.post-edit {
  max-width: 1600px;
}

/* 顶部工具条 */
.post-edit__topbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.post-edit__back {
  flex: none;
}

.post-edit__title-box {
  flex: 1;
  min-width: 200px;
  border-bottom: 1px solid transparent;
  transition: border-color 0.2s ease;
}

.post-edit__title-box:focus-within {
  border-bottom-color: var(--admin-border);
}

.post-edit__title.ant-input,
.post-edit__title.ant-input:hover,
.post-edit__title.ant-input:focus,
.post-edit__title.ant-input-focused {
  border: none;
  box-shadow: none;
  padding: 0 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--admin-text);
}

.post-edit__title::placeholder {
  font-weight: 400;
  color: var(--admin-muted);
}

/* 左右两栏：≥1200 左 1fr / 右 320px；<1200 右栏折到正文下方 */
.post-edit__body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 16px;
  align-items: start;
}

.post-edit__aside {
  display: grid;
  gap: 16px;
}

.post-edit__hint {
  margin: 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

/* 编辑器外壳 + 底部状态栏（32px） */
.post-edit__editor {
  border: 1px solid var(--admin-border);
  border-bottom: none;
  border-radius: var(--admin-radius-md) var(--admin-radius-md) 0 0;
  overflow: hidden;
}

.post-edit__editor :deep(.md-editor) {
  border: none;
  box-shadow: none;
  border-radius: 0;
}

.post-edit__statusbar {
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 12px;
  font-size: 12px;
  color: var(--admin-muted);
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: 0 0 var(--admin-radius-md) var(--admin-radius-md);
}

.post-edit__status-save--error {
  color: var(--admin-danger);
}

@media (max-width: 1199px) {
  .post-edit__body {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  /* 工具条换行：标题独占一行，动作靠右 */
  .post-edit__topbar {
    gap: 4px;
  }

  .post-edit__title-box {
    flex-basis: 100%;
    order: 3;
  }

  .post-edit__actions {
    margin-left: auto;
  }
}

/* SEO 检查清单 */
.seo-checklist {
  list-style: none;
  margin: 12px 0 0;
  padding: 0;
  display: grid;
  gap: 4px;
}

.seo-checklist li {
  font-size: 12px;
  color: var(--admin-muted);
}

.seo-checklist li.seo-check--warn {
  color: var(--admin-warning);
}
</style>
