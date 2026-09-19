<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
import {
  BarChartOutlined,
  HistoryOutlined,
  LockOutlined,
  ScheduleOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import { useAuthStore } from '@/stores/auth'

const { message } = useFeedback()
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const formRef = ref()
const submitting = ref(false)
const loginError = ref('')

const formState = reactive({
  username: '',
  password: '',
  remember: true,
})

const rules = {
  username: [{ required: true, message: '请输入用户名' }],
  password: [{ required: true, message: '请输入密码' }],
}

const features = [
  { icon: HistoryOutlined, title: '版本历史', desc: '每次保存自动存档，可对比、可回滚' },
  { icon: ScheduleOutlined, title: '定时发布', desc: '到点自动上线，重启不丢计划' },
  { icon: BarChartOutlined, title: '访问分析', desc: 'PV/UV 趋势与读者来源一目了然' },
]

async function handleSubmit() {
  loginError.value = ''
  await formRef.value?.validate()
  submitting.value = true
  try {
    await auth.login(formState.username.trim(), formState.password, formState.remember)
    message.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redirect)
  } catch (e) {
    // 拦截器已 toast;行内 Alert 提供持续可见的错误态(docs/09 §9.1)
    loginError.value = e instanceof Error && e.message ? e.message : '登录失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login-view">
    <!-- 左侧品牌面板 -->
    <aside class="login-hero">
      <div class="login-hero__inner">
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
      </div>
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
</template>

<style scoped>
.login-view {
  display: flex;
  min-height: 100vh;
  background: var(--admin-bg, #f5f6f8);
}

/* ---------- 左侧品牌面板 ---------- */
.login-hero {
  position: relative;
  flex: 1.15;
  display: flex;
  align-items: center;
  padding: 48px 64px;
  background: var(--admin-sidebar, #001529);
  overflow: hidden;
}

/* 细网格 + 品牌色光斑，克制的深度感 */
.login-hero::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(ellipse 60% 50% at 20% 15%, rgba(22, 119, 255, 0.16), transparent 70%),
    radial-gradient(ellipse 50% 40% at 85% 90%, rgba(22, 119, 255, 0.1), transparent 70%),
    linear-gradient(rgba(255, 255, 255, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.035) 1px, transparent 1px);
  background-size: 100% 100%, 100% 100%, 44px 44px, 44px 44px;
  pointer-events: none;
}

.login-hero__inner {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 460px;
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

/* ---------- 右侧表单 ---------- */
.login-main {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
}

.login-panel {
  width: 100%;
  max-width: 340px;
}

.login-panel__head {
  margin-bottom: 28px;
}

.login-panel__title {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--admin-text);
}

.login-panel__sub {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--admin-muted);
}

.login-panel__submit {
  margin-bottom: 0;
}

.login-panel__foot {
  margin: 20px 0 0;
  text-align: center;
  font-size: 12px;
  color: var(--admin-muted);
}

/* ---------- 移动端：隐藏品牌面板，表单撑满 ---------- */
@media (max-width: 991px) {
  .login-hero {
    display: none;
  }

  .login-main {
    align-items: flex-start;
    padding-top: 12vh;
  }

  .login-panel__head::before {
    content: 'MyBlog 管理后台';
    display: block;
    margin-bottom: 10px;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--admin-brand);
  }
}
</style>
