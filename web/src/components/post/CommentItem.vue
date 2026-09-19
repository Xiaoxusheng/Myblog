<template>
  <li class="comment-item">
    <div class="comment-main">
      <span class="avatar" :style="{ background: avatarColor }" aria-hidden="true">
        {{ initial }}
      </span>
      <div class="comment-body">
        <div class="comment-head">
          <span class="nickname">{{ comment.nickname }}</span>
          <span v-if="comment.isAdmin" class="admin-badge">博主</span>
          <a
            v-if="comment.website"
            :href="comment.website"
            target="_blank"
            rel="noopener noreferrer nofollow"
            class="website"
          >
            {{ displayWebsite }}
          </a>
          <span class="time">{{ formatRelative(comment.createdAt) }}</span>
        </div>
        <p class="content">{{ comment.content }}</p>
        <button v-if="enabled" class="reply-btn" type="button" @click="emit('reply', comment)">
          回复
        </button>
      </div>
    </div>
    <ul v-if="comment.children?.length" class="comment-children">
      <CommentItem
        v-for="child in comment.children"
        :key="child.id"
        :comment="child"
        :enabled="enabled"
        @reply="emit('reply', $event)"
      />
    </ul>
  </li>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CommentPublic } from '@/types'
import { formatRelative } from '@/utils/format'

const props = defineProps<{ comment: CommentPublic; enabled: boolean }>()

const emit = defineEmits<{ reply: [comment: CommentPublic] }>()

// 头像：昵称首字 + 由昵称哈希决定的低饱和底色
const PALETTE = [
  '#7c9c8a',
  '#8a9dc0',
  '#c0a48a',
  '#a08ac0',
  '#c08a9d',
  '#8ab8b0',
  '#b0a48a',
  '#8aa0b8'
]

const avatarColor = computed(() => {
  const name = props.comment.nickname || ''
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  }
  return PALETTE[hash % PALETTE.length]
})

const initial = computed(() => {
  const name = (props.comment.nickname || '').trim()
  return name ? name.charAt(0).toUpperCase() : '访'
})

const displayWebsite = computed(() =>
  (props.comment.website || '').replace(/^https?:\/\//, '').replace(/\/$/, '')
)
</script>

<style scoped>
.comment-item {
  list-style: none;
}

.comment-main {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.avatar {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  user-select: none;
}

.comment-body {
  flex: 1;
  min-width: 0;
}

.comment-head {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 8px;
}

.nickname {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-1);
}

.admin-badge {
  padding: 0 7px;
  border-radius: 5px;
  background: var(--brand-soft);
  border: 1px solid var(--brand-soft-border);
  color: var(--brand);
  font-size: 11.5px;
  line-height: 1.7;
}

.website {
  font-size: 12.5px;
  color: var(--text-3);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.website:hover {
  color: var(--brand);
}

.time {
  font-size: 12px;
  color: var(--text-3);
}

.content {
  margin-top: 6px;
  font-size: 14px;
  line-height: 1.75;
  color: var(--text-1);
  white-space: pre-wrap;
  word-break: break-word;
}

.reply-btn {
  margin-top: 6px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-3);
  font-size: 12.5px;
  transition: color var(--transition);
}

.reply-btn:hover {
  color: var(--brand);
}

/* 二级回复 */
.comment-children {
  list-style: none;
  margin: 10px 0 0;
  padding: 0 0 0 50px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.comment-children .avatar {
  width: 30px;
  height: 30px;
  font-size: 12.5px;
}

.comment-children .comment-main {
  padding: 12px 14px;
  background: var(--bg);
  border-color: var(--border);
}

@media (max-width: 640px) {
  .comment-children {
    padding-left: 20px;
  }
}
</style>
