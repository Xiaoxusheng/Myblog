<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import type { FormInstance } from 'ant-design-vue'
import { PictureOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import MediaSelectModal from '@/components/MediaSelectModal.vue'
import { getSettings, updateSettings } from '@/api/media'
import type { Settings } from '@/types/api'

const loading = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const logoModalOpen = ref(false)

const formState = reactive<Settings>({
  siteName: '',
  siteDescription: '',
  siteKeywords: '',
  siteUrl: '',
  logo: '',
  notice: '',
  icp: '',
  footerText: '',
  commentEnabled: true,
  postPageSize: 10,
  autoRedirectOnSlugChange: true,
})

const rules = {
  siteName: [{ required: true, message: '请输入站点名称' }],
  siteUrl: [
    {
      pattern: /^https?:\/\/\S*$/,
      message: '地址需以 http:// 或 https:// 开头',
    },
  ],
  postPageSize: [{ required: true, message: '请设置每页文章数' }],
}

async function load() {
  loading.value = true
  try {
    const result = await getSettings()
    Object.assign(formState, result.settings)
  } finally {
    loading.value = false
  }
}

async function save() {
  await formRef.value?.validate()
  saving.value = true
  try {
    await updateSettings({ ...formState, postPageSize: Number(formState.postPageSize) })
    message.success('设置已保存')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="系统设置" description="站点信息、评论开关与分页设置" />

    <a-spin :spinning="loading">
      <a-card :bordered="false">
        <a-form
          ref="formRef"
          :model="formState"
          :rules="rules"
          :label-col="{ style: { width: '140px' } }"
          :wrapper-col="{ flex: 'auto' }"
        >
          <a-form-item label="站点名称" name="siteName">
            <a-input v-model:value="formState.siteName" placeholder="如：MyBlog" :maxlength="50" />
          </a-form-item>
          <a-form-item label="站点描述" name="siteDescription">
            <a-textarea v-model:value="formState.siteDescription" placeholder="用于 SEO 的站点描述" :rows="2" :maxlength="300" />
          </a-form-item>
          <a-form-item label="站点关键词" name="siteKeywords">
            <a-input v-model:value="formState.siteKeywords" placeholder="多个关键词用英文逗号分隔" :maxlength="200" />
          </a-form-item>
          <a-form-item label="站点 URL" name="siteUrl">
            <a-input v-model:value="formState.siteUrl" placeholder="https://example.com（用于 RSS 等绝对链接）" />
          </a-form-item>
          <a-form-item label="站点 Logo" name="logo">
            <a-space direction="vertical" :size="8" style="width: 100%">
              <a-space wrap>
                <a-input v-model:value="formState.logo" placeholder="Logo 图片地址" style="width: 320px" allow-clear />
                <a-button @click="logoModalOpen = true">
                  <template #icon><PictureOutlined /></template>
                  从媒体库选择
                </a-button>
              </a-space>
              <a-image v-if="formState.logo" :src="formState.logo" :height="80" />
            </a-space>
          </a-form-item>
          <a-form-item label="公告" name="notice">
            <a-textarea v-model:value="formState.notice" placeholder="显示在前台的公告栏" :rows="2" :maxlength="300" />
          </a-form-item>
          <a-form-item label="ICP 备案号" name="icp">
            <a-input v-model:value="formState.icp" placeholder="如：京ICP备xxxxxxxx号" :maxlength="60" />
          </a-form-item>
          <a-form-item label="页脚文字" name="footerText">
            <a-input v-model:value="formState.footerText" placeholder="页脚版权说明" :maxlength="100" />
          </a-form-item>
          <a-form-item label="开启评论" name="commentEnabled">
            <a-switch v-model:checked="formState.commentEnabled" checked-children="开" un-checked-children="关" />
          </a-form-item>
          <a-form-item label="每页文章数" name="postPageSize">
            <a-input-number v-model:value="formState.postPageSize" :min="1" :max="50" style="width: 160px" />
          </a-form-item>
          <a-form-item label="Slug 重定向" name="autoRedirectOnSlugChange">
            <a-space direction="vertical" :size="4">
              <a-switch
                v-model:checked="formState.autoRedirectOnSlugChange"
                checked-children="开"
                un-checked-children="关"
              />
              <span class="settings-hint">修改文章 slug 时自动创建旧地址到新地址的 301 重定向，避免外链失效</span>
            </a-space>
          </a-form-item>
          <a-form-item :wrapper-col="{ offset: 0 }">
            <a-button type="primary" :loading="saving" @click="save">保存设置</a-button>
          </a-form-item>
        </a-form>
      </a-card>
    </a-spin>

    <MediaSelectModal v-model:open="logoModalOpen" @select="formState.logo = $event" />
  </div>
</template>

<style scoped>
.settings-hint {
  font-size: 12px;
  color: var(--admin-muted);
}
</style>
