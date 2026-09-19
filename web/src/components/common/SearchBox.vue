<template>
  <form class="search-box" role="search" @submit.prevent="submit">
    <input
      v-model="keyword"
      type="search"
      :placeholder="placeholder"
      aria-label="搜索文章"
    />
    <button type="submit" class="search-btn" aria-label="搜索">
      <svg viewBox="0 0 24 24" width="15" height="15" fill="currentColor" aria-hidden="true">
        <path
          d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 1 0-.7.7l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0A4.5 4.5 0 1 1 14 9.5 4.5 4.5 0 0 1 9.5 14z"
        />
      </svg>
    </button>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

withDefaults(defineProps<{ placeholder?: string }>(), { placeholder: '搜索文章…' })

const router = useRouter()
const keyword = ref('')

function submit(): void {
  const kw = keyword.value.trim()
  if (!kw) return
  router.push({ path: '/search', query: { keyword: kw } })
}
</script>

<style scoped>
.search-box {
  display: flex;
  align-items: center;
  height: 34px;
  padding: 0 4px 0 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--surface-2);
  transition: border-color var(--transition), background var(--transition);
}

.search-box:focus-within {
  border-color: var(--brand);
  background: var(--surface);
}

.search-box input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  font-size: 13.5px;
  color: var(--text-1);
}

.search-box input::placeholder {
  color: var(--text-3);
}

.search-box input::-webkit-search-cancel-button {
  -webkit-appearance: none;
}

.search-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-3);
  transition: color var(--transition);
}

.search-btn:hover {
  color: var(--brand);
}
</style>
