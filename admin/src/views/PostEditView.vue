<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import dayjs from 'dayjs'
import { ArrowLeftOutlined, ClearOutlined, PictureOutlined } from '@ant-design/icons-vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import FormSection from '@/components/FormSection.vue'
import MediaSelectModal from '@/components/MediaSelectModal.vue'
import { createPost, getPost, updatePost } from '@/api/posts'
import { getCategories } from '@/api/taxonomy'
import type { AdminPostItem, Category, PostStatus } from '@/types/api'

const route = useRoute()
const router = useRouter()

const postId = computed<number | null>(() => {
  const raw = route.params.id
  return raw ? Number(raw) : null
})

const loading = ref(false)
const saving = ref(false)
const categories = ref<Category[]>([])
const coverModalOpen = ref(false)

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
})

const formRef = ref()

const rules = {
  categoryId: [{ required: true, message: '请选择分类' }],
  slug: [
    {
      pattern: /^[a-zA-Z0-9_-]*$/,
      message: '仅支持字母、数字、短横线和下划线',
    },
  ],
}

// ---------------------------------------------------------------------------
// 本地草稿自动保存（3s 防抖写 localStorage，key 按 新建/编辑+id 区分）
// ---------------------------------------------------------------------------
type DraftSaveStatus = 'idle' | 'saving' | 'saved' | 'error'

interface LocalDraft {
  form: typeof formState
  savedAt: number
}

const DRAFT_PREFIX = 'blog_admin_post_draft'

const draftKey = computed(() =>
  postId.value ? `${DRAFT_PREFIX}_${postId.value}` : `${DRAFT_PREFIX}_new`,
)

const autosaveReady = ref(false)
const draftStatus = ref<DraftSaveStatus>('idle')
const savedAtText = ref('')
let draftTimer: ReturnType<typeof setTimeout> | null = null

function clearDraftTimer() {
  if (draftTimer) {
    clearTimeout(draftTimer)
    draftTimer = null
  }
}

function readDraft(): LocalDraft | null {
  try {
    const raw = localStorage.getItem(draftKey.value)
    if (!raw) return null
    const parsed = JSON.parse(raw) as LocalDraft
    if (!parsed || typeof parsed !== 'object' || !parsed.form) return null
    return parsed
  } catch {
    return null
  }
}

function writeDraft() {
  draftTimer = null
  try {
    const draft: LocalDraft = { form: { ...formState, tagNames: [...formState.tagNames] }, savedAt: Date.now() }
    localStorage.setItem(draftKey.value, JSON.stringify(draft))
    savedAtText.value = dayjs().format('HH:mm')
    draftStatus.value = 'saved'
  } catch {
    draftStatus.value = 'error'
  }
}

function clearLocalDraft() {
  clearDraftTimer()
  localStorage.removeItem(draftKey.value)
  draftStatus.value = 'idle'
}

function onFormChange() {
  if (!autosaveReady.value) return
  draftStatus.value = 'saving'
  clearDraftTimer()
  draftTimer = setTimeout(writeDraft, 3000)
}

watch(formState, onFormChange, { deep: true })

/** 表单快照对比（tags 排序后比较，避免服务端顺序差异误报） */
function snapshot(form: typeof formState): string {
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
  })
}

/** 载入后检查本地草稿：与服务器内容不同则询问恢复 */
function checkLocalDraft(serverPost: AdminPostItem | null) {
  const draft = readDraft()
  if (!draft) {
    autosaveReady.value = true
    return
  }
  const differs = serverPost
    ? snapshot(draft.form) !== snapshot(formState)
    : Boolean(
        draft.form.title.trim() ||
          draft.form.content.trim() ||
          draft.form.summary.trim() ||
          draft.form.tagNames.length > 0,
      )
  if (!differs) {
    // 与服务器一致，草稿无价值，直接清除
    clearLocalDraft()
    autosaveReady.value = true
    return
  }
  const savedAt = dayjs(draft.savedAt).format('HH:mm')
  Modal.confirm({
    title: '发现未保存的本地草稿',
    content: `本地保存于 ${savedAt}，与当前内容不同，是否恢复？`,
    okText: '恢复',
    cancelText: '不恢复',
    onOk: () => {
      Object.assign(formState, draft.form, { tagNames: [...draft.form.tagNames] })
      savedAtText.value = savedAt
      draftStatus.value = 'saved'
      autosaveReady.value = true
    },
    onCancel: () => {
      // 放弃恢复则丢弃本地草稿
      clearLocalDraft()
      autosaveReady.value = true
    },
  })
}

// ---------------------------------------------------------------------------
// 数据加载与保存
// ---------------------------------------------------------------------------

async function load() {
  autosaveReady.value = false
  clearDraftTimer()
  draftStatus.value = 'idle'
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
    checkLocalDraft(null)
    return
  }
  loading.value = true
  try {
    const result = await getPost(postId.value)
    fillForm(result.post)
    checkLocalDraft(result.post)
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
}

async function save(nextStatus: PostStatus) {
  if (!formState.title.trim()) {
    message.error('请输入文章标题')
    return
  }
  await formRef.value?.validate()
  const payload = {
    title: formState.title.trim(),
    slug: formState.slug.trim(),
    summary: formState.summary.trim(),
    content: formState.content,
    cover: formState.cover.trim(),
    categoryId: formState.categoryId as number,
    tags: formState.tagNames,
    status: nextStatus,
    isTop: formState.isTop,
  }
  saving.value = true
  try {
    if (postId.value) {
      await updatePost(postId.value, payload)
      formState.status = nextStatus
      message.success('保存成功')
      // 保存成功，本地草稿已同步到服务器，清除
      clearLocalDraft()
    } else {
      const result = await createPost(payload)
      message.success('创建成功')
      // 在路由切到编辑模式前清除 _new 草稿（draftKey 依赖当前路由）
      clearLocalDraft()
      // 转入编辑模式，后续保存走 PUT
      await router.replace(`/posts/edit/${result.post.id}`)
    }
  } finally {
    saving.value = false
  }
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

// ---------------------------------------------------------------------------
// 底部状态栏：字数 / 预计阅读时长（中文 400 字/分钟）
// ---------------------------------------------------------------------------
const wordCount = computed(() => formState.content.length)
const readingMinutes = computed(() =>
  wordCount.value === 0 ? 0 : Math.max(1, Math.ceil(wordCount.value / 400)),
)

// 离开页面前把未落盘的变更立即写入，避免丢失
onBeforeUnmount(() => {
  if (draftTimer) {
    writeDraft()
  }
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
        <a-button :loading="saving" @click="save(0)">保存草稿</a-button>
        <a-button type="primary" :loading="saving" @click="save(1)">发布</a-button>
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
                <template v-else-if="draftStatus === 'saved'">已保存至本地 {{ savedAtText }}</template>
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
                </a-radio-group>
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
          </aside>
        </div>
      </a-form>
    </a-spin>

    <MediaSelectModal v-model:open="coverModalOpen" @select="formState.cover = $event" />
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
</style>
