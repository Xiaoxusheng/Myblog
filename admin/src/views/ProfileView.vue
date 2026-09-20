<script setup lang="ts">
import { nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useFeedback } from '@/composables/useFeedback'
const { message } = useFeedback()
import type { FormInstance } from 'ant-design-vue'
import type { UploadRequestOption } from 'ant-design-vue/es/vc-upload/interface'
import { UserOutlined } from '@ant-design/icons-vue'
import { updatePassword, updateProfile } from '@/api/auth'
import { uploadImage } from '@/api/media'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()

// ---------- 个人资料 ----------
const profileSaving = ref(false)
const profileRef = ref<FormInstance>()
const avatarUploading = ref(false)
const passwordCard = ref<HTMLDivElement | null>(null)

const profileForm = reactive({
  nickname: '',
  email: '',
  avatar: '',
})

const profileRules = {
  nickname: [{ required: true, message: '请输入昵称' }],
  email: [
    { required: true, message: '请输入邮箱' },
    { type: 'email' as const, message: '邮箱格式不正确' },
  ],
}

function fillProfile() {
  profileForm.nickname = auth.user?.nickname || ''
  profileForm.email = auth.user?.email || ''
  profileForm.avatar = auth.user?.avatar || ''
}

async function saveProfile() {
  await profileRef.value?.validate()
  profileSaving.value = true
  try {
    const result = await updateProfile({
      nickname: profileForm.nickname.trim(),
      email: profileForm.email.trim(),
      avatar: profileForm.avatar.trim(),
    })
    auth.setUser(result.user)
    message.success('个人资料已保存')
  } finally {
    profileSaving.value = false
  }
}

async function customAvatarRequest(options: UploadRequestOption) {
  avatarUploading.value = true
  try {
    const result = await uploadImage(options.file as File)
    profileForm.avatar = result.url
    options.onSuccess?.(options.file)
    message.success('头像已上传，点击保存生效')
  } catch (error) {
    options.onError?.(error as Error)
  } finally {
    avatarUploading.value = false
  }
}

function beforeAvatarUpload(file: File) {
  if (!file.type.startsWith('image/')) {
    message.error('仅支持上传图片文件')
    return false
  }
  if (file.size > 10 * 1024 * 1024) {
    message.error('图片大小不能超过 10MB')
    return false
  }
  return true
}

// ---------- 修改密码 ----------
const passwordSaving = ref(false)
const passwordRef = ref<FormInstance>()

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const passwordRules = {
  oldPassword: [{ required: true, message: '请输入旧密码' }],
  newPassword: [
    { required: true, message: '请输入新密码' },
    { min: 6, message: '新密码长度至少 6 位' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码' },
    {
      validator: (_rule: unknown, value: string) => {
        if (value && value !== passwordForm.newPassword) {
          return Promise.reject('两次输入的密码不一致')
        }
        return Promise.resolve()
      },
    },
  ],
}

async function savePassword() {
  await passwordRef.value?.validate()
  passwordSaving.value = true
  try {
    const result = await updatePassword({
      oldPassword: passwordForm.oldPassword,
      newPassword: passwordForm.newPassword,
    })
    // 旧 token 已被服务端吊销（契约 #13），替换为新 token 保持会话
    if (result?.token) {
      auth.setToken(result.token)
    }
    message.success('密码已修改')
    passwordRef.value?.resetFields()
  } finally {
    passwordSaving.value = false
  }
}

onMounted(() => {
  // 从用户下拉“修改密码”进入时，定位到修改密码卡片
  if (route.query.focus === 'password') {
    void nextTick(() => {
      passwordCard.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }
})

// 直达 /profile 时 fetchMe 可能未返回，用户信息到达后回填
watch(
  () => auth.user,
  () => fillProfile(),
)
</script>

<template>
  <div class="page">
    <a-row :gutter="16">
      <a-col :xs="24" :lg="12">
        <a-card title="个人资料" :bordered="false">
          <a-form ref="profileRef" :model="profileForm" :rules="profileRules" layout="vertical">
            <a-form-item label="头像">
              <a-space :size="16">
                <a-avatar :size="64" :src="profileForm.avatar || undefined">
                  <template #icon><UserOutlined /></template>
                </a-avatar>
                <a-space direction="vertical" :size="4">
                  <a-upload
                    accept="image/*"
                    :show-upload-list="false"
                    :custom-request="customAvatarRequest"
                    :before-upload="beforeAvatarUpload"
                  >
                    <a-button size="small" :loading="avatarUploading">上传头像</a-button>
                  </a-upload>
                  <a-button v-if="profileForm.avatar" type="link" size="small" @click="profileForm.avatar = ''">
                    清除头像
                  </a-button>
                </a-space>
              </a-space>
            </a-form-item>
            <a-form-item label="用户名">
              <a-input :value="auth.user?.username" disabled />
            </a-form-item>
            <a-form-item label="昵称" name="nickname">
              <a-input v-model:value="profileForm.nickname" placeholder="展示在前台评论区的昵称" :maxlength="50" />
            </a-form-item>
            <a-form-item label="邮箱" name="email">
              <a-input v-model:value="profileForm.email" placeholder="name@example.com" :maxlength="100" />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" :loading="profileSaving" @click="saveProfile">保存资料</a-button>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="12">
        <div ref="passwordCard">
          <a-card title="修改密码" :bordered="false">
            <a-form ref="passwordRef" :model="passwordForm" :rules="passwordRules" layout="vertical">
              <a-form-item label="旧密码" name="oldPassword">
                <a-input-password v-model:value="passwordForm.oldPassword" placeholder="请输入旧密码" autocomplete="current-password" />
              </a-form-item>
              <a-form-item label="新密码" name="newPassword">
                <a-input-password v-model:value="passwordForm.newPassword" placeholder="至少 6 位" autocomplete="new-password" />
              </a-form-item>
              <a-form-item label="确认新密码" name="confirmPassword">
                <a-input-password v-model:value="passwordForm.confirmPassword" placeholder="再次输入新密码" autocomplete="new-password" />
              </a-form-item>
              <a-form-item>
                <a-button type="primary" :loading="passwordSaving" @click="savePassword">修改密码</a-button>
              </a-form-item>
            </a-form>
          </a-card>
        </div>
      </a-col>
    </a-row>
  </div>
</template>
