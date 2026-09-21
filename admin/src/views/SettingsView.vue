<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import type { FormInstance } from 'ant-design-vue'
import { PictureOutlined, SaveOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import FormSection from '@/components/FormSection.vue'
import MediaSelectModal from '@/components/MediaSelectModal.vue'
import { useFeedback } from '@/composables/useFeedback'
import { getSettings, updateSettings } from '@/api/media'
import { SITE_URL_PATTERN } from '@/utils/validators'
import type { Settings } from '@/types/api'

/**
 * 系统设置（v2 设计稿第 8 页）。
 *
 * 形态：左侧分类导航（锚点）+ 右侧单列大区块表单 + 底部保存状态条。
 * 与文章编辑器右栏 FormSection 同一套表单语言，不再用多列卡片网格
 * （多列会把"站点信息"和"品牌页脚"并排放，视线要横跳；单列顺读更符合配置流程）。
 */

const { message } = useFeedback()

const loading = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const logoModalOpen = ref(false)

const DEFAULT_FORM: Settings = {
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
}

const formState = reactive<Settings>({ ...DEFAULT_FORM })

/** 保存基线：用于"有 N 项未保存更改"的真实比对，不靠按钮状态猜 */
const baseline = ref<string>('')
/** 上次保存时间（仅本次会话内记录；页面重载后显示"未知"而不是编一个时间） */
const lastSavedAt = ref<string>('')

function snapshot(): string {
  return JSON.stringify(formState)
}

const dirtyCount = computed(() => {
  if (!baseline.value) return 0
  const before = JSON.parse(baseline.value) as Settings
  return (Object.keys(formState) as (keyof Settings)[]).filter(
    (key) => formState[key] !== before[key],
  ).length
})

/** 分区导航：锚点 + 标题，顺序即表单顺序（单一来源，不另维护一份目录） */
const sections = [
  { key: 'site', title: '站点信息' },
  { key: 'brand', title: '品牌与页脚' },
  { key: 'comment', title: '评论与阅读' },
] as const

const activeSection = ref<string>(sections[0].key)

const rules = {
  siteName: [{ required: true, message: '请输入站点名称' }],
  siteUrl: [
    {
      pattern: SITE_URL_PATTERN,
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
    baseline.value = snapshot()
  } finally {
    loading.value = false
  }
}

async function save() {
  await formRef.value?.validate()
  saving.value = true
  try {
    await updateSettings({ ...formState, postPageSize: Number(formState.postPageSize) })
    baseline.value = snapshot()
    lastSavedAt.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
    message.success('设置已保存')
  } finally {
    saving.value = false
  }
}

/** 恢复默认：把表单退回打开时的状态（不是"恢复出厂设置"——那会静默改线上配置） */
function restore() {
  if (!baseline.value) return
  Object.assign(formState, JSON.parse(baseline.value) as Settings)
  message.info('已恢复为打开页面时的内容')
}

/** 锚点滚动 */
function goSection(key: string) {
  activeSection.value = key
  document.getElementById(`settings-${key}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="系统设置" description="站点参数、内容策略与安全偏好">
      <template #actions>
        <a-button :disabled="saving" @click="restore">恢复默认</a-button>
      </template>
    </PageHeader>

    <a-spin :spinning="loading">
      <a-form ref="formRef" :model="formState" :rules="rules" layout="vertical">
        <div class="settings-layout">
          <!-- 左侧分类导航（v2 设计稿左侧栏） -->
          <nav class="settings-nav" aria-label="设置分类">
            <button
              v-for="section in sections"
              :key="section.key"
              type="button"
              class="settings-nav__item"
              :class="{ 'settings-nav__item--active': activeSection === section.key }"
              @click="goSection(section.key)"
            >
              {{ section.title }}
            </button>
          </nav>

          <!-- 右侧单列区块表单 -->
          <div class="settings-body">
            <FormSection id="settings-site" title="站点信息" description="显示在前后台导航、浏览器标题与分享卡片中">
              <a-form-item label="站点名称" name="siteName">
                <a-input v-model:value="formState.siteName" placeholder="如：MyBlog" :maxlength="50" />
              </a-form-item>
              <a-form-item label="站点描述" name="siteDescription">
                <a-textarea
                  v-model:value="formState.siteDescription"
                  placeholder="用于 SEO 的站点描述"
                  :rows="2"
                  :maxlength="300"
                />
              </a-form-item>
              <a-form-item label="站点关键词" name="siteKeywords">
                <a-input
                  v-model:value="formState.siteKeywords"
                  placeholder="多个关键词用英文逗号分隔"
                  :maxlength="200"
                />
              </a-form-item>
              <a-form-item label="站点 URL" name="siteUrl">
                <a-input
                  v-model:value="formState.siteUrl"
                  placeholder="https://example.com（用于 RSS 等绝对链接）"
                />
              </a-form-item>
            </FormSection>

            <FormSection id="settings-brand" title="品牌与页脚" description="品牌露出与页脚署名信息">
              <a-form-item label="站点 Logo" name="logo">
                <a-space direction="vertical" :size="8" style="width: 100%">
                  <a-space wrap>
                    <a-input
                      v-model:value="formState.logo"
                      placeholder="Logo 图片地址"
                      style="width: 320px"
                      allow-clear
                    />
                    <a-button @click="logoModalOpen = true">
                      <template #icon><PictureOutlined /></template>
                      从媒体库选择
                    </a-button>
                  </a-space>
                  <a-image v-if="formState.logo" :src="formState.logo" :height="80" />
                </a-space>
              </a-form-item>
              <a-form-item label="公告" name="notice">
                <a-textarea
                  v-model:value="formState.notice"
                  placeholder="显示在前台的公告栏"
                  :rows="2"
                  :maxlength="300"
                />
              </a-form-item>
              <a-form-item label="ICP 备案号" name="icp">
                <a-input
                  v-model:value="formState.icp"
                  placeholder="如：京ICP备xxxxxxxx号"
                  :maxlength="60"
                />
              </a-form-item>
              <a-form-item label="页脚文字" name="footerText">
                <a-input
                  v-model:value="formState.footerText"
                  placeholder="页脚版权说明"
                  :maxlength="100"
                />
              </a-form-item>
            </FormSection>

            <FormSection id="settings-comment" title="评论与阅读" description="互动开关与列表分页策略">
              <a-form-item label="开启评论" name="commentEnabled">
                <a-switch
                  v-model:checked="formState.commentEnabled"
                  checked-children="开"
                  un-checked-children="关"
                />
              </a-form-item>
              <a-form-item label="每页文章数" name="postPageSize">
                <a-input-number
                  v-model:value="formState.postPageSize"
                  :min="1"
                  :max="50"
                  style="width: 160px"
                />
              </a-form-item>
              <a-form-item label="Slug 重定向" name="autoRedirectOnSlugChange">
                <a-space direction="vertical" :size="4">
                  <a-switch
                    v-model:checked="formState.autoRedirectOnSlugChange"
                    checked-children="开"
                    un-checked-children="关"
                  />
                  <span class="settings-hint">
                    修改文章 slug 时自动创建旧地址到新地址的 301 重定向，避免外链失效
                  </span>
                </a-space>
              </a-form-item>
            </FormSection>
          </div>
        </div>

        <!-- 底部保存状态条（v2 设计稿）：如实反映未保存项数 -->
        <div class="settings-footer">
          <span class="settings-footer__status">
            <template v-if="lastSavedAt">上次保存于 {{ lastSavedAt }} · </template>
            <template v-if="dirtyCount > 0">
              <b>{{ dirtyCount }}</b> 项未保存更改
            </template>
            <template v-else>没有未保存的更改</template>
          </span>
          <a-button type="primary" :loading="saving" :disabled="dirtyCount === 0" @click="save">
            <template #icon><SaveOutlined /></template>
            保存更改
          </a-button>
        </div>
      </a-form>
    </a-spin>

    <MediaSelectModal v-model:open="logoModalOpen" @select="formState.logo = $event" />
  </div>
</template>

<style scoped>
/* 左导航 200px + 右表单：设计稿为固定左导航，窄屏折叠为顶部横排 */
.settings-layout {
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr);
  gap: 24px;
  align-items: start;
}

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  position: sticky;
  top: 80px;
}

.settings-nav__item {
  padding: 8px 12px;
  font-family: inherit;
  font-size: 13px;
  color: var(--admin-muted);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--admin-radius-sm);
  cursor: pointer;
  transition:
    color var(--admin-dur) var(--admin-ease),
    background-color var(--admin-dur) var(--admin-ease);
}

.settings-nav__item:hover {
  color: var(--admin-text);
  background: var(--admin-surface-2);
}

.settings-nav__item--active {
  font-weight: 600;
  color: var(--admin-brand-active);
  background: var(--admin-brand-bg);
}

.settings-nav__item:focus-visible {
  outline: 2px solid var(--admin-brand);
  outline-offset: 1px;
}

.settings-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

/* 底部状态条：与表单同宽，粘在视口底部方便随时保存 */
.settings-footer {
  position: sticky;
  bottom: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px;
  margin-top: 16px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-md);
}

.settings-footer__status {
  font-size: 13px;
  color: var(--admin-muted);
}

.settings-footer__status b {
  font-weight: 600;
  color: var(--admin-warning-text);
}

.settings-hint {
  font-size: 12px;
  color: var(--admin-muted);
}

@media (max-width: 900px) {
  .settings-layout {
    grid-template-columns: minmax(0, 1fr);
    gap: 16px;
  }

  .settings-nav {
    flex-direction: row;
    flex-wrap: wrap;
    position: static;
  }
}
</style>
