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
        :maxlength="CONTENT_MAX"
        placeholder="友善发言，支持 Markdown"
        :class="{ invalid: !!errors.content }"
      ></textarea>
      <div class="content-foot">
        <p v-if="errors.content" class="field-error">{{ errors.content }}</p>
        <span class="char-count" :class="{ near: form.content.length > CONTENT_MAX - 100 }">
          {{ form.content.length }} / {{ CONTENT_MAX }}
        </span>
      </div>
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

/** 与后端评论长度限制一致的单一真相来源（docs/08 §6.2） */
const CONTENT_MAX = 1000

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
  padding: 20px;
  border-radius: var(--radius-lg);
}

.replying {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: var(--brand-soft);
  color: var(--text-2);
  font-size: var(--fs-sm);
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
  font-size: var(--fs-xs);
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
  font-size: var(--fs-sm);
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
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--bg);
  padding: 9px 12px;
  font-size: var(--fs-md);
  color: var(--text-1);
  outline: none;
  transition: border-color var(--transition), box-shadow var(--transition);
}

.field textarea {
  resize: vertical;
  min-height: 92px;
  line-height: 1.75;
}

/* focus 紫色描边 + 柔光（p14 输入框规格） */
.field input:focus,
.field textarea:focus {
  border-color: var(--brand);
  box-shadow: 0 0 0 3px var(--brand-soft);
  background: var(--bg);
}

.field input.invalid,
.field textarea.invalid {
  border-color: var(--danger);
}

.field input.invalid:focus,
.field textarea.invalid:focus {
  box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.12);
}

.field-error {
  margin-top: 5px;
  font-size: var(--fs-xs);
  color: var(--danger);
}

/* 内容字段底部行：错误文案与字数计数共用一行，高度稳定不跳动 */
.content-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 20px;
  margin-top: 6px;
}

.content-foot .field-error {
  margin-top: 0;
}

.char-count {
  margin-left: auto;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
  transition: color var(--transition);
}

.char-count.near {
  color: var(--danger);
}

.form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.hint {
  font-size: var(--fs-xs);
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
