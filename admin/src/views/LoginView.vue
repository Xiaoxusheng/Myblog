<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
import { LockOutlined, UserOutlined } from '@ant-design/icons-vue'
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
    <div class="login-box">
      <div class="login-brand">
        <span class="login-brand__logo">M</span>
        <span class="login-brand__name">MyBlog</span>
      </div>

      <h1 class="login-title">管理员登录</h1>

      <a-alert
        v-if="loginError"
        type="error"
        :message="loginError"
        show-icon
        style="margin-bottom: 12px"
      />

      <a-form
        ref="formRef"
        :model="formState"
        :rules="rules"
        layout="vertical"
        hide-required-mark
        @finish="handleSubmit"
      >
        <a-form-item name="username">
          <a-input
            v-model:value="formState.username"
            placeholder="用户名"
            autocomplete="username"
          >
            <template #prefix><UserOutlined /></template>
          </a-input>
        </a-form-item>

        <a-form-item name="password">
          <a-input-password
            v-model:value="formState.password"
            placeholder="密码"
            autocomplete="current-password"
          >
            <template #prefix><LockOutlined /></template>
          </a-input-password>
        </a-form-item>

        <a-form-item name="captcha">
          <div class="captcha-row">
            <a-input v-model:value="formState.captcha" placeholder="验证码" maxlength="4" autocomplete="off" />
            <button
              type="button"
              class="captcha-img"
              title="看不清？点击刷新"
              @click="refreshCaptcha"
            >
              <img v-if="captchaImage" :src="captchaImage" alt="验证码" />
              <span v-else class="captcha-img__loading">…</span>
            </button>
          </div>
        </a-form-item>

        <a-form-item class="login-remember">
          <a-checkbox v-model:checked="formState.remember">7 天内免登录</a-checkbox>
        </a-form-item>

        <a-form-item class="login-submit">
          <a-button type="primary" block html-type="submit" :loading="submitting">登录</a-button>
        </a-form-item>
      </a-form>
    </div>

    <p class="login-foot">© 2026 MyBlog</p>
  </div>
</template>

<style scoped>
.login-view {
  position: relative;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, #fafbfd 0%, #eef1f6 100%);
  padding: 24px;
}

/* 顶部极淡品牌光晕，打破平坦但不抢戏 */
.login-view::before {
  content: '';
  position: absolute;
  top: -140px;
  left: 50%;
  transform: translateX(-50%);
  width: 720px;
  height: 320px;
  background: radial-gradient(ellipse at center, rgba(22, 119, 255, 0.08), transparent 70%);
  pointer-events: none;
}

.login-box {
  position: relative;
  width: 100%;
  max-width: 264px;
  padding: 24px 22px 18px;
  background: #fff;
  border: 1px solid var(--admin-border, #e5e7eb);
  border-radius: 10px;
  box-shadow: 0 6px 24px rgba(0, 21, 41, 0.06);
  text-align: center;
}

.login-brand {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-bottom: 14px;
}

.login-brand__logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: linear-gradient(135deg, #1677ff, #4096ff);
  font-size: 14px;
  font-weight: 700;
  color: #fff;
}

.login-brand__name {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.03em;
  color: var(--admin-text);
}

.login-title {
  margin: 0 0 16px;
  font-size: 15px;
  font-weight: 500;
  color: var(--admin-muted);
}

.login-box :deep(.ant-form-item) {
  margin-bottom: 12px;
}

.captcha-row {
  display: flex;
  gap: 8px;
}

.captcha-img {
  flex: none;
  width: 96px;
  height: 32px;
  padding: 0;
  border: 1px solid var(--admin-border, #d9d9d9);
  border-radius: 6px;
  background: #fff;
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

.captcha-img__loading {
  font-size: 14px;
  color: var(--admin-muted);
}

.login-remember {
  text-align: left;
}

.login-remember :deep(.ant-checkbox + span) {
  font-size: 12px;
  color: var(--admin-muted);
}

.login-submit {
  margin-bottom: 0;
}

.login-foot {
  margin: 16px 0 0;
  font-size: 11px;
  color: var(--admin-muted);
}
</style>
