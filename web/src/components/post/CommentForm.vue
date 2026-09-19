<template>
  <form id="comment-form" class="comment-form card" novalidate @submit.prevent="submit">
    <p v-if="replyTarget" class="replying">
      回复 <b>@{{ replyTarget.nickname }}</b>
      <button type="button" class="cancel" @click="emit('cancel-reply')">取消回复</button>
    </p>

    <div class="form-row">
      <div class="field">
        <label for="cf-nickname">昵称 <i aria-hidden="true">*</i></label>
        <input
          id="cf-nickname"
          v-model.trim="form.nickname"
          type="text"
          maxlength="30"
          placeholder="你的昵称"
          :class="{ invalid: !!errors.nickname }"
          autocomplete="nickname"
        />
        <p v-if="errors.nickname" class="field-error">{{ errors.nickname }}</p>
      </div>
      <div class="field">
        <label for="cf-email">邮箱 <i aria-hidden="true">*</i></label>
        <input
          id="cf-email"
          v-model.trim="form.email"
          type="email"
          maxlength="60"
          placeholder="不会公开显示"
          :class="{ invalid: !!errors.email }"
          autocomplete="email"
        />
        <p v-if="errors.email" class="field-error">{{ errors.email }}</p>
      </div>
      <div class="field">
        <label for="cf-website">网站 <span class="optional">（选填）</span></label>
        <input
          id="cf-website"
          v-model.trim="form.website"
          type="url"
          maxlength="120"
          placeholder="https://example.com"
        />
      </div>
    </div>

    <div class="field">
      <label for="cf-content">内容 <i aria-hidden="true">*</i></label>
      <textarea
        id="cf-content"
        v-model.trim="form.content"
        rows="4"
        maxlength="1000"
        placeholder="写下你的评论…"
        :class="{ invalid: !!errors.content }"
      ></textarea>
      <p v-if="errors.content" class="field-error">{{ errors.content }}</p>
    </div>

    <div class="form-actions">
      <span class="hint">提交后需审核通过才会展示</span>
      <button class="btn btn-primary" type="submit" :disabled="submitting">
        {{ submitting ? '提交中…' : '提交评论' }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { createComment } from '@/api/post'
import { ApiError } from '@/api/http'
import type { CommentPublic } from '@/types'
import { isValidEmail } from '@/utils/format'
import { loadCommentProfile, saveCommentProfile } from '@/utils/storage'
import { useToast } from '@/composables/useToast'

const props = defineProps<{ slug: string; replyTarget: CommentPublic | null }>()

const emit = defineEmits<{ 'cancel-reply': []; created: [] }>()

const toast = useToast()
const form = reactive({ nickname: '', email: '', website: '', content: '' })
const errors = reactive({ nickname: '', email: '', content: '' })
const submitting = ref(false)

onMounted(() => {
  const profile = loadCommentProfile()
  form.nickname = profile.nickname
  form.email = profile.email
  form.website = profile.website
})

watch(
  () => props.replyTarget,
  (target) => {
    if (target) errors.content = ''
  }
)

function validate(): boolean {
  errors.nickname = form.nickname ? '' : '请填写昵称'
  errors.email = form.email
    ? isValidEmail(form.email)
      ? ''
      : '邮箱格式不正确'
    : '请填写邮箱'
  errors.content = form.content ? '' : '请填写评论内容'
  return !errors.nickname && !errors.email && !errors.content
}

function normalizeWebsite(raw: string): string {
  const value = raw.trim()
  if (!value) return ''
  return /^https?:\/\//i.test(value) ? value : `https://${value}`
}

async function submit(): Promise<void> {
  if (submitting.value) return
  if (!validate()) return

  submitting.value = true
  try {
    // 两级树：回复二级评论时挂到其顶级评论下（顶级评论 parentId 为 0/null）
    const target = props.replyTarget
    const parentId = target
      ? target.parentId && target.parentId > 0
        ? target.parentId
        : target.id
      : undefined
    await createComment(props.slug, {
      parentId,
      nickname: form.nickname,
      email: form.email,
      website: normalizeWebsite(form.website) || undefined,
      content: form.content
    })
    saveCommentProfile({
      nickname: form.nickname,
      email: form.email,
      website: normalizeWebsite(form.website)
    })
    form.content = ''
    emit('created')
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.code === 20002) toast.error('评论功能已关闭')
      else if (e.code === 20003) toast.error('操作过于频繁，请稍后再试')
      else toast.error(e.message)
    } else {
      toast.error('提交失败，请稍后重试')
    }
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.comment-form {
  margin-top: 16px;
  padding: 18px 20px;
}

.replying {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--brand-soft);
  color: var(--text-2);
  font-size: 13.5px;
}

.replying b {
  color: var(--brand);
  font-weight: 600;
}

.replying .cancel {
  margin-left: auto;
  border: none;
  background: transparent;
  color: var(--text-3);
  font-size: 12.5px;
}

.replying .cancel:hover {
  color: var(--brand);
}

.form-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

@media (max-width: 640px) {
  .form-row {
    grid-template-columns: 1fr;
  }
}

.field {
  display: flex;
  flex-direction: column;
  margin-bottom: 12px;
}

.field label {
  margin-bottom: 6px;
  font-size: 13px;
  color: var(--text-2);
}

.field label i {
  color: var(--danger);
  font-style: normal;
}

.optional {
  color: var(--text-3);
}

.field input,
.field textarea {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  padding: 8px 12px;
  font-size: 14px;
  color: var(--text-1);
  outline: none;
  transition: border-color var(--transition), background var(--transition);
}

.field textarea {
  resize: vertical;
  min-height: 88px;
  line-height: 1.7;
}

.field input:focus,
.field textarea:focus {
  border-color: var(--brand);
  background: var(--surface);
}

.field input.invalid,
.field textarea.invalid {
  border-color: var(--danger);
}

.field-error {
  margin-top: 5px;
  font-size: 12.5px;
  color: var(--danger);
}

.form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.hint {
  font-size: 12.5px;
  color: var(--text-3);
}

@media (max-width: 640px) {
  .form-actions {
    flex-direction: column-reverse;
    align-items: stretch;
  }

  .form-actions .btn {
    width: 100%;
  }

  .hint {
    text-align: center;
  }
}
</style>
