<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
import { LockOutlined, UserOutlined } from '@ant-design/icons-vue'
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
  <div class="login-page">
    <a-card class="login-card" :bordered="false">
      <div class="login-card__head">
        <p class="login-card__brand">MyBlog</p>
        <p class="login-card__sub">管理后台</p>
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

        <a-form-item>
          <a-button type="primary" size="large" block html-type="submit" :loading="submitting">
            登录
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<style scoped>
.login-card {
  width: 380px;
  max-width: 100%;
}

.login-card__head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  margin-bottom: 24px;
}

.login-card__brand {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: var(--admin-text);
}

.login-card__sub {
  margin: 0;
  font-size: 13px;
  color: var(--admin-muted);
}
</style>
