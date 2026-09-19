<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
import {
  BarChartOutlined,
  HistoryOutlined,
  LockOutlined,
  ScheduleOutlined,
  SyncOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import { getCaptcha } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const { message } = useFeedback()
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const formRef = ref()
const submitting = ref(false)
const loginError = ref('')

const captchaId = ref('')
const captchaImage = ref('')
const captchaLoading = ref(false)

const formState = reactive({
  username: '',
  password: '',
  captcha: '',
  remember: true,
})

const rules = {
  username: [{ required: true, message: '请输入用户名' }],
  password: [{ required: true, message: '请输入密码' }],
  captcha: [{ required: true, message: '请输入验证码' }],
}

const features = [
  { icon: HistoryOutlined, title: '版本历史', desc: '每次保存自动存档，可对比、可回滚' },
  { icon: ScheduleOutlined, title: '定时发布', desc: '到点自动上线，重启不丢计划' },
  { icon: BarChartOutlined, title: '访问分析', desc: 'PV/UV 趋势与读者来源一目了然' },
]

/** 拉取图形验证码（进入页面 / 点击图片 / 登录失败后刷新） */
async function refreshCaptcha() {
  captchaLoading.value = true
  try {
    const data = await getCaptcha()
    captchaId.value = data.captchaId
    captchaImage.value = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(data.image)}`
  } finally {
    captchaLoading.value = false
  }
}

async function handleSubmit() {
  loginError.value = ''
  await formRef.value?.validate()
  submitting.value = true
  try {
    await auth.login({
      username: formState.username.trim(),
      password: formState.password,
      remember: formState.remember,
      captchaId: captchaId.value,
      captchaCode: formState.captcha.trim(),
    })
    message.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redirect)
  } catch (e) {
    // 拦截器已 toast;行内 Alert 提供持续可见的错误态(docs/09 §9.1)
    loginError.value = e instanceof Error && e.message ? e.message : '登录失败，请稍后重试'
    // 验证码单次有效：任何失败后都换新题
    formState.captcha = ''
    void refreshCaptcha()
  } finally {
    submitting.value = false
  }
}

onMounted(refreshCaptcha)
</script>

<template>
  <div class="login-view">
    <!-- 背景层：深色向浅色水平过渡 + 细网格，整页融合 -->
    <div class="login-bg" aria-hidden="true"></div>

    <div class="login-content">
      <!-- 左侧品牌区 -->
      <aside class="login-hero">
        <div class="login-hero__brand">
          <span class="login-hero__logo">M</span>
          <span class="login-hero__name">MyBlog</span>
        </div>

        <h1 class="login-hero__title">写作、发布、分析<br />一站式的个人内容管理平台</h1>
        <p class="login-hero__sub">从一篇 Markdown 到完整的 Mini CMS——版本、专题、数据尽在掌握。</p>

        <ul class="login-hero__features">
          <li v-for="f in features" :key="f.title" class="login-hero__feature">
            <span class="login-hero__feature-icon"><component :is="f.icon" /></span>
            <span>
              <strong>{{ f.title }}</strong>
              <em>{{ f.desc }}</em>
            </span>
          </li>
        </ul>

        <p class="login-hero__foot">© 2026 MyBlog · 个人内容管理平台</p>
      </aside>

      <!-- 右侧表单 -->
      <main class="login-main">
        <div class="login-panel">
          <div class="login-panel__head">
            <h2 class="login-panel__title">欢迎回来</h2>
            <p class="login-panel__sub">登录以管理你的内容</p>
          </div>

          <a-alert
            v-if="loginError"
            type="error"
            :message="loginError"
            show-icon
            style="margin-bottom: 16px"
          />

          <a-form
            ref="formRef"
            :model="formState"
            :rules="rules"
            layout="vertical"
            hide-required-mark
            @finish="handleSubmit"
          >
            <a-form-item label="用户名" name="username">
              <a-input
                v-model:value="formState.username"
                size="large"
                placeholder="用户名"
                autocomplete="username"
              >
                <template #prefix><UserOutlined /></template>
              </a-input>
            </a-form-item>

            <a-form-item label="密码" name="password">
              <a-input-password
                v-model:value="formState.password"
                size="large"
                placeholder="密码"
                autocomplete="current-password"
              >
                <template #prefix><LockOutlined /></template>
              </a-input-password>
            </a-form-item>

            <a-form-item label="验证码" name="captcha">
              <div class="captcha-row">
                <a-input
                  v-model:value="formState.captcha"
                  size="large"
                  placeholder="不区分大小写"
                  maxlength="4"
                  autocomplete="off"
                />
                <button
                  type="button"
                  class="captcha-img"
                  :class="{ 'captcha-img--loading': captchaLoading }"
                  title="看不清？点击刷新"
                  @click="refreshCaptcha"
                >
                  <img v-if="captchaImage" :src="captchaImage" alt="验证码" />
                  <SyncOutlined v-else spin />
                </button>
              </div>
            </a-form-item>

            <a-form-item>
              <a-checkbox v-model:checked="formState.remember">7 天内免登录</a-checkbox>
            </a-form-item>

            <a-form-item class="login-panel__submit">
              <a-button type="primary" size="large" block html-type="submit" :loading="submitting">
                登录
              </a-button>
            </a-form-item>
          </a-form>

          <p class="login-panel__foot">登录即代表同意合理的站点使用约定</p>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
/* 整页背景：深海军蓝向浅灰水平过渡，两侧融合 */
.login-view {
  position: relative;
  min-height: 100vh;
  background: #001529;
}

/* 整页一体深色：品牌色光斑 + 细网格，无分界 */
.login-bg {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 55% 65% at 18% 8%, rgba(22, 119, 255, 0.22), transparent 70%),
    radial-gradient(ellipse 45% 55% at 88% 92%, rgba(64, 150, 255, 0.14), transparent 70%);
  pointer-events: none;
}

.login-bg::after {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: radial-gradient(ellipse 90% 90% at 40% 40%, #000 30%, transparent 100%);
  -webkit-mask-image: radial-gradient(ellipse 90% 90% at 40% 40%, #000 30%, transparent 100%);
}

.login-content {
  position: relative;
  z-index: 1;
  display: flex;
  min-height: 100vh;
}

/* ---------- 左侧品牌区（文字压在深色段上） ---------- */
.login-hero {
  flex: 1.15;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 18px;
  max-width: 540px;
  padding: 48px 32px 48px 64px;
  color: #fff;
}

.login-hero__brand {
  display: flex;
  align-items: center;
  gap: 10px;
}

.login-hero__logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: linear-gradient(135deg, #1677ff, #4096ff);
  font-size: 18px;
  font-weight: 700;
  color: #fff;
}

.login-hero__name {
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: rgba(255, 255, 255, 0.92);
}

.login-hero__title {
  margin: 8px 0 0;
  font-size: 28px;
  line-height: 1.4;
  font-weight: 600;
  color: #fff;
}

.login-hero__sub {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: rgba(255, 255, 255, 0.55);
}

.login-hero__features {
  list-style: none;
  margin: 20px 0 0;
  padding: 0;
  display: grid;
  gap: 18px;
}

.login-hero__feature {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.login-hero__feature-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex: none;
  border-radius: 8px;
  background: rgba(22, 119, 255, 0.18);
  border: 1px solid rgba(64, 150, 255, 0.35);
  font-size: 15px;
  color: #69b1ff;
}

.login-hero__feature strong {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.88);
}

.login-hero__feature em {
  display: block;
  margin-top: 2px;
  font-style: normal;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
}

.login-hero__foot {
  margin: 28px 0 0;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.28);
}

/* ---------- 右侧表单（磨砂浮层压在过渡带上） ---------- */
.login-main {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
}

.login-panel {
  width: 100%;
  max-width: 320px;
  padding: 24px 24px 18px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.login-panel__head {
  margin-bottom: 18px;
}

.login-panel__title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.94);
}

.login-panel__sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
}

/* 深色玻璃卡内的表单控件适配 */
.login-panel :deep(.ant-form-item-label > label) {
  color: rgba(255, 255, 255, 0.62);
}

.login-panel :deep(.ant-input),
.login-panel :deep(.ant-input-affix-wrapper) {
  background: rgba(255, 255, 255, 0.07);
  border-color: rgba(255, 255, 255, 0.16);
  color: rgba(255, 255, 255, 0.92);
}

.login-panel :deep(.ant-input::placeholder) {
  color: rgba(255, 255, 255, 0.32);
}

.login-panel :deep(.ant-input-affix-wrapper > .ant-input) {
  background: transparent;
}

.login-panel :deep(.ant-input-password-icon) {
  color: rgba(255, 255, 255, 0.4);
}

.login-panel :deep(.ant-input-affix-wrapper-focused),
.login-panel :deep(.ant-input-affix-wrapper:focus-within) {
  border-color: var(--admin-brand, #1677ff);
  box-shadow: none;
}

.login-panel :deep(.ant-checkbox + span) {
  color: rgba(255, 255, 255, 0.68);
}

.captcha-row {
  display: flex;
  gap: 10px;
}

.captcha-row .ant-input-affinity-wrapper,
.captcha-row .ant-input {
  flex: 1;
}

.captcha-img {
  flex: none;
  width: 132px;
  height: 40px;
  padding: 0;
  border: 1px solid var(--admin-border, #d9d9d9);
  border-radius: 6px;
  background: #eef2f7;
  cursor: pointer;
  overflow: hidden;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.2s;
}

.captcha-img:hover {
  border-color: var(--admin-brand, #1677ff);
}

.captcha-img img {
  width: 100%;
  height: 100%;
  display: block;
}

.login-panel__submit {
  margin-bottom: 0;
}

.login-panel__foot {
  margin: 14px 0 0;
  text-align: center;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.32);
}

/* ---------- 中窄屏：收窄文案宽度，避开过渡带 ---------- */
/* ---------- 移动端：纵向融合（上深下浅），表单沉到底部 ---------- */
@media (max-width: 991px) {
  .login-content {
    flex-direction: column;
  }

  .login-hero {
    flex: none;
    max-width: none;
    padding: 48px 24px 8px;
  }

  .login-hero__sub,
  .login-hero__features,
  .login-hero__foot {
    display: none;
  }

  .login-hero__title {
    font-size: 21px;
  }

  .login-main {
    flex: 1;
    align-items: flex-start;
    padding-top: 12px;
  }

  .login-panel {
    max-width: none;
  }
}
</style>
