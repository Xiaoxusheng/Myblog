<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { FormInstance } from 'ant-design-vue'
import { LockOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons-vue'
import { getCaptcha } from '@/api/auth'
import { ApiError } from '@/api/http'
import { fetchPublicSite } from '@/api/site'
import { useAuthStore } from '@/stores/auth'
import { useFeedback } from '@/composables/useFeedback'

const { message } = useFeedback()
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const formRef = ref<FormInstance>()
const submitting = ref(false)
const loginError = ref('')

const captchaId = ref('')
const captchaImage = ref('')
const captchaLoading = ref(false)

/** 站点名从公开接口读取，失败则静默降级 */
const siteName = ref('MyBlog')

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

const brandInitial = computed(() => (siteName.value.trim()[0] || 'M').toUpperCase())

const year = new Date().getFullYear()

/**
 * 拉取图形验证码（进入页面 / 点击图片 / 登录失败后刷新）。
 *
 * 用递增序号丢弃过期响应：连点刷新或「挂载取题」与「失败后重取」并发时，
 * 先发出的请求可能后返回，若不丢弃就会用旧 captchaId 覆盖新图片，
 * 造成「图上字符」与「提交的 id」错配，服务端一律判为验证码错误/过期。
 */
let captchaSeq = 0
async function refreshCaptcha() {
  const seq = ++captchaSeq
  captchaLoading.value = true
  try {
    const data = await getCaptcha()
    if (seq !== captchaSeq) return // 已有更新的请求在途/已完成，丢弃本次结果
    captchaId.value = data.captchaId
    captchaImage.value = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(data.image)}`
  } catch {
    if (seq !== captchaSeq) return
    // 取题失败：清掉旧图，避免用户对着过期图片输答案（拦截器已 toast）
    captchaId.value = ''
    captchaImage.value = ''
  } finally {
    if (seq === captchaSeq) captchaLoading.value = false
  }
}

async function loadSiteName() {
  try {
    const info = await fetchPublicSite()
    if (info?.siteName) siteName.value = info.siteName
  } catch {
    /* 读不到站点名不影响登录，保留默认值 */
  }
}

async function handleSubmit() {
  loginError.value = ''
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
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
    // 行内错误区提供持续可见的错误态（全局 toast 会消失）
    loginError.value = e instanceof Error && e.message ? e.message : '登录失败，请稍后重试'

    // 验证码单次有效：服务端无论对错都会销毁该题，因此只要这次提交消费掉了题目
    // （即错误码为 10001 = 验证码已过期/不正确），就必须换新题并清空输入。
    // 密码错误（20001）、限流（20003）等情况验证码已经校验通过，不该被牵连刷新——
    // 否则用户每次改密码都要重看验证码，容易被误解成「验证码一直过期」。
    const code = e instanceof ApiError ? e.code : undefined
    if (code === 10001 || captchaId.value === '') {
      formState.captcha = ''
      void refreshCaptcha()
    }
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  void refreshCaptcha()
  void loadSiteName()
})
</script>

<template>
  <div class="login">
    <!-- 左：品牌面板（宽屏展示，作用是把「这是什么产品」讲清楚） -->
    <aside class="login-brand-panel">
      <div class="login-brand-panel__inner">
        <div class="login-brand">
          <span class="login-brand__mark">{{ brandInitial }}</span>
          <span class="login-brand__name">{{ siteName }}</span>
        </div>

        <div class="login-brand-copy">
          <h2 class="login-brand-copy__title">内容管理后台</h2>
          <p class="login-brand-copy__desc">
            在这里撰写与发布文章、管理分类与标签、处理评论审核，并查看站点访问数据。
          </p>
        </div>

        <ul class="login-brand-list">
          <li>
            <span class="login-brand-list__dot" />
            <span class="login-brand-list__label">文章与草稿</span>
            <span class="login-brand-list__hint">自动保存 · 冲突保护</span>
          </li>
          <li>
            <span class="login-brand-list__dot" />
            <span class="login-brand-list__label">评论与审核</span>
            <span class="login-brand-list__hint">待审队列 · 敏感词</span>
          </li>
          <li>
            <span class="login-brand-list__dot" />
            <span class="login-brand-list__label">站点数据</span>
            <span class="login-brand-list__hint">访问趋势 · 内容统计</span>
          </li>
        </ul>
      </div>
    </aside>

    <!-- 右：表单区 -->
    <main class="login-form-panel">
      <div class="login-form-panel__inner">
        <!-- 窄屏时品牌面板隐藏，这里补一个轻量品牌头 -->
        <div class="login-compact-brand">
          <span class="login-brand__mark">{{ brandInitial }}</span>
          <span class="login-brand__name">{{ siteName }}</span>
        </div>

        <header class="login-head">
          <h1 class="login-head__title">登录</h1>
          <p class="login-head__sub">使用管理员账号继续</p>
        </header>

        <a-alert
          v-if="loginError"
          class="login-alert"
          type="error"
          :message="loginError"
          show-icon
          closable
          @close="loginError = ''"
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
              placeholder="请输入用户名"
              autocomplete="username"
            >
              <template #prefix><UserOutlined class="login-field-icon" /></template>
            </a-input>
          </a-form-item>

          <a-form-item label="密码" name="password">
            <a-input-password
              v-model:value="formState.password"
              size="large"
              placeholder="请输入密码"
              autocomplete="current-password"
            >
              <template #prefix><LockOutlined class="login-field-icon" /></template>
            </a-input-password>
          </a-form-item>

          <a-form-item label="验证码" name="captcha">
            <div class="login-captcha">
              <a-input
                v-model:value="formState.captcha"
                size="large"
                placeholder="请输入右侧字符"
                maxlength="4"
                autocomplete="off"
              />
              <button
                type="button"
                class="login-captcha__box"
                :class="{ 'is-loading': captchaLoading, 'is-blank': !captchaImage }"
                title="看不清？点击换一张"
                aria-label="刷新验证码"
                @click="refreshCaptcha"
              >
                <img
                  v-if="captchaImage"
                  :src="captchaImage"
                  alt=""
                  aria-hidden="true"
                  @error="captchaImage = ''"
                />
                <span v-else class="login-captcha__placeholder" />
                <span class="login-captcha__refresh"><ReloadOutlined /></span>
              </button>
            </div>
          </a-form-item>

          <div class="login-options">
            <a-checkbox v-model:checked="formState.remember">7 天内免登录</a-checkbox>
          </div>

          <a-button
            class="login-submit"
            type="primary"
            size="large"
            block
            html-type="submit"
            :loading="submitting"
          >
            登录
          </a-button>
        </a-form>

        <p class="login-foot">© {{ year }} {{ siteName }} · 管理后台</p>
      </div>
    </main>
  </div>
</template>

<style scoped>
.login {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  min-height: 100vh;
  background: var(--admin-surface);
}

/* ---------- 左：品牌面板 ---------- */

.login-brand-panel {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 56px;
  background: var(--admin-bg);
  border-right: 1px solid var(--admin-border);
  overflow: hidden;
}

/* 极淡的品牌光晕：只做氛围，不做装饰 */
.login-brand-panel::before {
  content: '';
  position: absolute;
  top: -180px;
  left: -120px;
  width: 560px;
  height: 560px;
  background: radial-gradient(circle, rgba(22, 119, 255, 0.1), transparent 68%);
  pointer-events: none;
}

.login-brand-panel__inner {
  position: relative;
  width: 100%;
  max-width: 380px;
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 10px;
}

.login-brand__mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--admin-radius-sm);
  background: var(--admin-brand);
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  line-height: 1;
}

.login-brand__name {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.01em;
  color: var(--admin-text);
}

.login-brand-copy {
  margin-top: 40px;
}

.login-brand-copy__title {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
  line-height: 1.3;
  letter-spacing: -0.01em;
  color: var(--admin-text);
}

.login-brand-copy__desc {
  margin: 12px 0 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--admin-muted);
}

.login-brand-list {
  margin: 36px 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 14px;
}

.login-brand-list li {
  display: flex;
  align-items: baseline;
  gap: 10px;
  font-size: 13px;
}

.login-brand-list__dot {
  flex: none;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--admin-brand);
  transform: translateY(-2px);
}

.login-brand-list__label {
  color: var(--admin-text);
  font-weight: 500;
}

.login-brand-list__hint {
  color: var(--admin-muted);
}

/* ---------- 右：表单区 ---------- */

.login-form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 40px;
  background: var(--admin-surface);
}

.login-form-panel__inner {
  width: 100%;
  max-width: 340px;
}

.login-compact-brand {
  display: none;
  align-items: center;
  gap: 10px;
  margin-bottom: 32px;
}

.login-head {
  margin-bottom: 28px;
}

.login-head__title {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--admin-text);
}

.login-head__sub {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--admin-muted);
}

.login-alert {
  margin-bottom: 20px;
}

/* 表单项：标签在上，间距靠节奏建立 */
.login-form-panel :deep(.ant-form-item) {
  margin-bottom: 18px;
}

.login-form-panel :deep(.ant-form-item-label) {
  padding-bottom: 6px;
}

.login-form-panel :deep(.ant-form-item-label > label) {
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-text);
  height: auto;
}

.login-field-icon {
  color: var(--admin-muted);
}

/* 验证码：输入框 + 可点击图块 */
.login-captcha {
  display: flex;
  gap: 10px;
}

.login-captcha :deep(.ant-input-affix-wrapper),
.login-captcha :deep(.ant-input) {
  flex: 1 1 auto;
  min-width: 0;
}

.login-captcha__box {
  position: relative;
  flex: none;
  width: 112px;
  height: 40px;
  padding: 0;
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-sm);
  background: var(--admin-surface-2);
  cursor: pointer;
  overflow: hidden;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition:
    border-color 0.18s ease-out,
    background-color 0.18s ease-out;
}

.login-captcha__box:hover {
  border-color: var(--admin-brand);
}

.login-captcha__box:focus-visible {
  outline: 2px solid var(--admin-brand);
  outline-offset: 1px;
}

.login-captcha__box.is-loading {
  opacity: 0.6;
}

/* 加载失败 / 未取到验证码：显示可点击的重试占位，而不是浏览器的裂图 */
.login-captcha__box.is-blank {
  background: var(--admin-surface-2);
}

.login-captcha__box img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.login-captcha__placeholder {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 1.5px dashed var(--admin-muted);
  opacity: 0.5;
}

/* 刷新角标：hover 才显出，避免静态干扰 */
.login-captcha__refresh {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  color: #fff;
  background: rgba(0, 0, 0, 0.42);
  opacity: 0;
  transition: opacity 0.18s ease-out;
}

.login-captcha__box:hover .login-captcha__refresh {
  opacity: 1;
}

.login-options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.login-options :deep(.ant-checkbox + span) {
  font-size: 13px;
  color: var(--admin-muted);
}

.login-submit {
  height: 42px;
  font-weight: 500;
}

.login-foot {
  margin: 24px 0 0;
  font-size: 12px;
  color: var(--admin-muted);
  text-align: center;
}

/* ---------- 响应式 ---------- */

/* 中等宽度：收起品牌面板，改用顶部紧凑品牌头 */
@media (max-width: 900px) {
  .login {
    grid-template-columns: minmax(0, 1fr);
  }

  .login-brand-panel {
    display: none;
  }

  .login-compact-brand {
    display: flex;
  }

  .login-form-panel {
    padding: 40px 24px;
  }
}

/* 窄屏：进一步收紧留白 */
@media (max-width: 480px) {
  .login-form-panel {
    padding: 32px 20px;
    align-items: flex-start;
  }

  .login-head__title {
    font-size: 22px;
  }

  .login-captcha__box {
    width: 96px;
  }
}
</style>
