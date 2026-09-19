<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined, ClearOutlined, PictureOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
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
  title: [{ required: true, message: '请输入文章标题' }],
  categoryId: [{ required: true, message: '请选择分类' }],
  slug: [
    {
      pattern: /^[a-zA-Z0-9_-]*$/,
      message: '仅支持字母、数字、短横线和下划线',
    },
  ],
}

async function load() {
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
    return
  }
  loading.value = true
  try {
    const result = await getPost(postId.value)
    fillForm(result.post)
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
    } else {
      const result = await createPost(payload)
      message.success('创建成功')
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
</script>

<template>
  <div class="page">
    <PageHeader :title="postId ? '编辑文章' : '新建文章'" description="支持 Markdown 编辑、图片上传与封面设置">
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
                <a-input v-model:value="formState.title" placeholder="请输入文章标题" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :md="8">
              <a-form-item label="Slug" name="slug">
                <a-input v-model:value="formState.slug" placeholder="留空自动生成" />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :md="8">
              <a-form-item label="分类" name="categoryId">
                <a-select
                  v-model:value="formState.categoryId"
                  placeholder="请选择分类"
                  :options="categories.map((item) => ({ label: item.name, value: item.id }))"
                />
              </a-form-item>
            </a-col>
            <a-col :xs="24" :md="8">
              <a-form-item label="标签" name="tagNames">
                <a-select
                  v-model:value="formState.tagNames"
                  mode="tags"
                  placeholder="输入后回车添加标签"
                  :token-separators="[',']"
                />
              </a-form-item>
            </a-col>
            <a-col :xs="12" :md="8">
              <a-form-item label="置顶" name="isTop">
                <a-switch v-model:checked="formState.isTop" />
              </a-form-item>
            </a-col>
            <a-col :xs="12" :md="16">
              <a-form-item label="状态" name="status">
                <a-radio-group v-model:value="formState.status">
                  <a-radio :value="0">草稿</a-radio>
                  <a-radio :value="1">已发布</a-radio>
                  <a-radio :value="2">隐藏</a-radio>
                </a-radio-group>
              </a-form-item>
            </a-col>
            <a-col :span="24">
              <a-form-item label="摘要" name="summary">
                <a-textarea
                  v-model:value="formState.summary"
                  placeholder="留空则前台可能截取正文开头"
                  :rows="3"
                  :maxlength="500"
                  show-count
                />
              </a-form-item>
            </a-col>
            <a-col :span="24">
              <a-form-item label="封面" name="cover">
                <a-space direction="vertical" :size="8" style="width: 100%">
                  <a-space wrap>
                    <a-input
                      v-model:value="formState.cover"
                      placeholder="输入图片外链地址"
                      style="width: 320px"
                      allow-clear
                    />
                    <a-button @click="coverModalOpen = true">
                      <template #icon><PictureOutlined /></template>
                      从媒体库选择
                    </a-button>
                    <a-button v-if="formState.cover" @click="formState.cover = ''">
                      <template #icon><ClearOutlined /></template>
                      清除
                    </a-button>
                  </a-space>
                  <a-image v-if="formState.cover" :src="formState.cover" :height="120" />
                </a-space>
              </a-form-item>
            </a-col>
          </a-row>
        </a-card>

        <a-card :bordered="false" title="正文内容" class="editor-card">
          <MarkdownEditor v-model="formState.content" placeholder="请输入 Markdown 内容，支持粘贴图片自动上传" />
        </a-card>
      </a-form>
    </a-spin>

    <MediaSelectModal v-model:open="coverModalOpen" @select="formState.cover = $event" />
  </div>
</template>

<style scoped>
.editor-card {
  margin-top: 16px;
}
</style>
