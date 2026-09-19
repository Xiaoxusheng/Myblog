<template>
  <section class="comment-section" aria-label="评论">
    <h2 class="section-heading">
      评论
      <span v-if="totalCount > 0" class="heading-count">{{ totalCount }} 条</span>
    </h2>

    <CommentForm
      v-if="enabled"
      :slug="slug"
      :reply-target="replyTarget"
      @cancel-reply="replyTarget = null"
      @created="onCreated"
    />
    <p v-else class="comment-closed card">评论功能已关闭</p>

    <div v-if="loading" class="comment-skeletons" aria-hidden="true">
      <div v-for="i in 3" :key="i" class="sk-comment">
        <span class="skeleton sk-avatar"></span>
        <div class="sk-body">
          <span class="skeleton sk-line sk-nick"></span>
          <span class="skeleton sk-line"></span>
        </div>
      </div>
    </div>

    <ErrorState v-else-if="error" small :message="error" @retry="load" />

    <template v-else>
      <ul v-if="comments.length" class="comment-list">
        <CommentItem
          v-for="comment in comments"
          :key="comment.id"
          :comment="comment"
          :enabled="enabled"
          @reply="openReply"
        />
      </ul>
      <EmptyState v-else size="compact" title="还没有评论，来写下第一条" />
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchComments } from '@/api/post'
import { ApiError, isRequestCanceled } from '@/api/http'
import type { CommentPublic } from '@/types'
import CommentItem from './CommentItem.vue'
import CommentForm from './CommentForm.vue'
import ErrorState from '@/components/common/ErrorState.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'

const props = defineProps<{ slug: string; enabled: boolean }>()

const toast = useToast()
const comments = ref<CommentPublic[]>([])
const loading = ref(true)
const error = ref('')
const replyTarget = ref<CommentPublic | null>(null)

let controller: AbortController | null = null

const totalCount = computed(() => {
  let count = 0
  const walk = (list: CommentPublic[]): void => {
    for (const item of list) {
      count++
      walk(item.children ?? [])
    }
  }
  walk(comments.value)
  return count
})

async function load(): Promise<void> {
  controller?.abort()
  const localController = new AbortController()
  controller = localController
  loading.value = true
  error.value = ''
  try {
    const list = await fetchComments(props.slug, localController.signal)
    if (controller !== localController) return
    comments.value = list
  } catch (e) {
    if (controller !== localController || isRequestCanceled(e)) return
    error.value = e instanceof ApiError ? e.message : '评论加载失败'
  } finally {
    if (controller === localController) loading.value = false
  }
}

function openReply(target: CommentPublic): void {
  replyTarget.value = target
  document.getElementById('comment-form')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function onCreated(): void {
  toast.success('已提交，等待审核')
}

onMounted(() => {
  void load()
})
</script>

<style scoped>
.comment-section {
  margin-top: 40px;
}

.section-heading {
  font-size: 18px;
  font-weight: 600;
}

.heading-count {
  margin-left: 6px;
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  color: var(--text-3);
}

.comment-closed {
  margin-top: 14px;
  padding: 14px 18px;
  color: var(--text-3);
  font-size: 13.5px;
}

.comment-list {
  list-style: none;
  margin: 18px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  /* 顶级评论间以 hairline 分隔（CommentItem 内处理），不留 gap */
  gap: 0;
}

/* 骨架 */
.comment-skeletons {
  margin-top: 18px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.sk-comment {
  display: flex;
  gap: 12px;
}

.sk-avatar {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 50%;
}

.sk-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sk-line {
  display: block;
  height: 13px;
  width: 70%;
}

.sk-nick {
  width: 26%;
}
</style>
