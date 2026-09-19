<script setup lang="ts">
import { SLUG_REQUIRED_PATTERN, SLUG_PATTERN_MESSAGE } from '@/utils/validators'
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import FormSection from '@/components/FormSection.vue'
import { useFeedback } from '@/composables/useFeedback'
import { useAutoSave } from '@/composables/useAutoSave'
import { createPage, getPage, updatePage } from '@/api/pages'
import { silentUpdatePage } from '@/utils/autosaveHttp'
import type { PageItem, PagePayload, PageStatus } from '@/types/api'

const route = useRoute()
const router = useRouter()
const { message } = useFeedback()

const pageId = computed<number | null>(() => {
  const raw = route.params.id
  return raw ? Number(raw) : null
})

const loading = ref(false)
const saving = ref(false)
const formRef = ref()

const formState = reactive({
  title: '',
  slug: '',
  content: '',
  status: 0 as PageStatus,
})

const rules = {
  slug: [
    { required: true, message: '请输入 Slug，页面必须指定唯一的 slug' },
    {
      pattern: SLUG_REQUIRED_PATTERN,
      message: SLUG_PATTERN_MESSAGE,
    },
  ],
}

// ---------------------------------------------------------------------------
// 自动保存:与文章编辑器同一内核(useAutoSave),体验对齐 docs/09 §7.1
// 已发布页面(status=1)的未定稿编辑不实时上线,只留本地草稿
// ---------------------------------------------------------------------------
const DRAFT_PREFIX = 'blog_admin_page_draft'
const draftKey = computed(() =>
  pageId.value ? `${DRAFT_PREFIX}_${pageId.value}` : `${DRAFT_PREFIX}_new`,
)

function buildPayload(status: PageStatus): PagePayload {
  return {
    title: formState.title.trim(),
    slug: formState.slug.trim(),
    content: formState.content,
    status,
  }
}

function snapshotOf(form: typeof formState): string {
  return JSON.stringify({ ...form, title: form.title.trim(), slug: form.slug.trim() })
}

const wordCount = computed(() => formState.content.length)

const {
  draftStatus,
  savedAtText,
  checkLocalDraft,
  markServerSynced,
  flushDraft,
} = useAutoSave({
  draftKey,
  form: formState,
  formSnapshot: () => snapshotOf(formState),
  draftHasContent: (draft) => Boolean(draft.form.title.trim() || draft.form.content.trim()),
  restore: (draft) => {
    Object.assign(formState, draft.form)
  },
  subject: '页面',
  canServerSave: () =>
    pageId.value !== null &&
    formState.title.trim() !== '' &&
    formState.slug.trim() !== '' &&
    formState.status !== 1,
  serverSave: () => silentUpdatePage(pageId.value as number, buildPayload(formState.status)),
  payloadSnapshot: () => JSON.stringify(buildPayload(formState.status)),
  isBusy: () => saving.value,
})

function resetForm() {
  formState.title = ''
  formState.slug = ''
  formState.content = ''
  formState.status = 0
}

function fillForm(page: PageItem) {
  formState.title = page.title
  formState.slug = page.slug
  formState.content = page.content
  formState.status = page.status
}

async function load() {
  if (!pageId.value) {
    resetForm()
    // 新建:本地草稿有内容才询问恢复
    checkLocalDraft()
    return
  }
  loading.value = true
  try {
    const result = await getPage(pageId.value)
    fillForm(result.page)
    // 编辑:草稿与服务器内容对比,不同才询问恢复
    checkLocalDraft((draft) => snapshotOf(draft.form) !== snapshotOf(formState))
  } finally {
    loading.value = false
  }
}

async function save(nextStatus: PageStatus) {
  if (!formState.title.trim()) {
    message.warning('请输入页面标题')
    return
  }
  await formRef.value?.validate()
  const payload = buildPayload(nextStatus)
  saving.value = true
  try {
    if (pageId.value) {
      await updatePage(pageId.value, payload)
      formState.status = nextStatus
      message.success('保存成功')
    } else {
      const created = await createPage(payload)
      message.success('创建成功')
      await router.replace(`/pages/edit/${created.id}`)
    }
    markServerSynced()
  } finally {
    saving.value = false
  }
}

function goBack() {
  void router.push('/pages')
}

watch(pageId, () => void load(), { immediate: true })

onBeforeUnmount(() => {
  flushDraft()
})
</script>

<template>
  <div class="page page-edit">
    <!-- 顶部工具条:返回 + 无框化标题 + 保存动作(与文章编辑器同构,docs/09 §7.1) -->
    <div class="page-edit__topbar">
      <a-button type="text" class="page-edit__back" @click="goBack">
        <template #icon><ArrowLeftOutlined /></template>
      </a-button>
      <div class="page-edit__title-box">
        <a-input v-model:value="formState.title" class="page-edit__title" placeholder="请输入页面标题" />
      </div>
      <a-space class="page-edit__actions">
        <a-button :loading="saving" @click="save(0)">保存草稿</a-button>
        <a-button type="primary" :loading="saving" @click="save(1)">发布</a-button>
      </a-space>
    </div>

    <a-spin :spinning="loading">
      <a-form ref="formRef" :model="formState" :rules="rules" layout="vertical">
        <div class="page-edit__body">
          <div class="page-edit__main">
            <div class="page-edit__editor">
              <MarkdownEditor
                v-model="formState.content"
                placeholder="请输入 Markdown 内容，支持粘贴图片自动上传"
              />
            </div>
            <div class="page-edit__statusbar">
              <span class="page-edit__status-metrics tabular-nums">字数 {{ wordCount }}</span>
              <span
                class="page-edit__status-save"
                :class="{ 'page-edit__status-save--error': draftStatus === 'error' }"
              >
                <template v-if="draftStatus === 'saving'">正在保存…</template>
                <template v-else-if="draftStatus === 'saved-server'">已保存 {{ savedAtText }}</template>
                <template v-else-if="draftStatus === 'saved-local'">已保存至本地 {{ savedAtText }}</template>
                <template v-else-if="draftStatus === 'error'">保存失败</template>
                <template v-else>自动保存已开启</template>
              </span>
            </div>
          </div>

          <!-- 右栏:分组卡片 -->
          <aside class="page-edit__aside">
            <a-card :bordered="false">
              <FormSection title="发布">
                <a-form-item label="状态" name="status">
                  <a-radio-group v-model:value="formState.status">
                    <a-radio :value="0">草稿</a-radio>
                    <a-radio :value="1">已发布</a-radio>
                  </a-radio-group>
                </a-form-item>
              </FormSection>
            </a-card>

            <a-card :bordered="false">
              <FormSection title="路径">
                <a-form-item label="Slug" name="slug">
                  <a-input v-model:value="formState.slug" placeholder="如 about，作为访问路径" />
                </a-form-item>
                <p class="page-edit__hint">作为页面访问路径，发布后通过 /page/&lt;slug&gt; 访问。</p>
              </FormSection>
            </a-card>
          </aside>
        </div>
      </a-form>
    </a-spin>
  </div>
</template>

<style scoped>
.page-edit {
  max-width: 1600px;
}

/* 顶部工具条(与 PostEditView 同构) */
.page-edit__topbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.page-edit__back {
  flex: none;
}

.page-edit__title-box {
  flex: 1;
  min-width: 200px;
  border-bottom: 1px solid transparent;
  transition: border-color 0.2s ease;
}

.page-edit__title-box:focus-within {
  border-bottom-color: var(--admin-border);
}

.page-edit__title.ant-input,
.page-edit__title.ant-input:hover,
.page-edit__title.ant-input:focus,
.page-edit__title.ant-input-focused {
  border: none;
  box-shadow: none;
  padding: 0 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--admin-text);
}

.page-edit__title::placeholder {
  font-weight: 400;
  color: var(--admin-muted);
}

/* 左右两栏:≥1200 左 1fr / 右 320px;<1200 右栏折到正文下方 */
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
  margin: 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

/* 编辑器外壳 + 底部状态栏(32px) */
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

.page-edit__statusbar {
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

.page-edit__status-save--error {
  color: var(--admin-danger);
}

@media (max-width: 1199px) {
  .page-edit__body {
    grid-template-columns: 1fr;
  }
}
</style>
