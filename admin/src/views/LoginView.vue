<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { LockOutlined, UserOutlined } from '@ant-design/icons-vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const formRef = ref()
const submitting = ref(false)

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
  await formRef.value?.validate()
  submitting.value = true
  try {
    await auth.login(formState.username.trim(), formState.password, formState.remember)
    message.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redirect)
  } catch {
    // 错误提示由 API 拦截器统一 message.error（如 20001 用户名或密码错误）
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <a-card class="login-card" :bordered="false">
      <div class="login-card__head">
        <a-typography-title :level="3" style="margin: 0">MyBlog</a-typography-title>
        <a-typography-text type="secondary">管理后台</a-typography-text>
      </div>

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
</style>
