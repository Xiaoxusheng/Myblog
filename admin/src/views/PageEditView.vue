<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import { createPage, getPage, updatePage } from '@/api/pages'
import type { PageItem, PageStatus } from '@/types/api'

const route = useRoute()
const router = useRouter()

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
  title: [{ required: true, message: '请输入页面标题' }],
  slug: [
    { required: true, message: '请输入 Slug，页面必须指定唯一的 slug' },
    {
      pattern: /^[a-zA-Z0-9_-]+$/,
      message: '仅支持字母、数字、短横线和下划线',
    },
  ],
}

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
    formRef.value?.resetFields()
    resetForm()
    return
  }
  loading.value = true
  try {
    const result = await getPage(pageId.value)
    fillForm(result.page)
  } finally {
    loading.value = false
  }
}

async function save(nextStatus: PageStatus) {
  await formRef.value?.validate()
  const payload = {
    title: formState.title.trim(),
    slug: formState.slug.trim(),
    content: formState.content,
    status: nextStatus,
  }
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
  } finally {
    saving.value = false
  }
}

function goBack() {
  void router.push('/pages')
}

watch(pageId, () => void load(), { immediate: true })
</script>

<template>
  <div class="page">
    <PageHeader :title="pageId ? '编辑页面' : '新建页面'" description="使用 Markdown 编写自定义页面">
      <template #actions>
        <a-button @click="goBack">
          <template #icon><ArrowLeftOutlined /></template>
          返回列表
        </a-button>
        <a-button :loading="saving" @click="save(0)">保存草稿</a-button>
        <a-button type="primary" :loading="saving" @click="save(1)">发布</a-button>
      </template>
    </PageHeader>

    <a-spin :spinning="loading">
      <a-form ref="formRef" :model="formState" :rules="rules" layout="vertical">
        <a-card :bordered="false">
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="标题" name="title">
                <a-input v-model:value="formState.title" placeholder="请输入页面标题" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :md="12">
              <a-form-item label="Slug" name="slug">
                <a-input v-model:value="formState.slug" placeholder="如 about，作为访问路径" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :md="12">
              <a-form-item label="状态" name="status">
                <a-radio-group v-model:value="formState.status">
                  <a-radio :value="0">草稿</a-radio>
                  <a-radio :value="1">已发布</a-radio>
                </a-radio-group>
              </a-form-item>
            </a-col>
          </a-row>
        </a-card>

        <a-card :bordered="false" title="正文内容" class="editor-card">
          <MarkdownEditor v-model="formState.content" placeholder="请输入 Markdown 内容" />
        </a-card>
      </a-form>
    </a-spin>
  </div>
</template>

<style scoped>
.editor-card {
  margin-top: 16px;
}
</style>
