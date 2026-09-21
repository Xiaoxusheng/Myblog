<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { SearchOutlined } from '@ant-design/icons-vue'

/**
 * 命令面板（侧栏「搜索或跳转 Ctrl K」入口，v2 设计稿）。
 *
 * 纯前端导航工具：数据源就是管理端菜单树 + 几个全局动作，不引入新接口。
 * 键盘：↑↓ 移动、Enter 打开、Esc 关闭；打开时输入框自动聚焦。
 */
export interface PaletteItem {
  key: string
  title: string
  /** 所属分组，用于结果里显示面包屑式的定位信息 */
  group?: string
  path: string
}

const props = defineProps<{ open: boolean; items: PaletteItem[] }>()
const emit = defineEmits<{ (e: 'update:open', value: boolean): void }>()

const router = useRouter()
const keyword = ref('')
const activeIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLElement | null>(null)

/** 简单子序列匹配：拼音/英文/中文都按"字符顺序出现"命中，够用且零依赖 */
function matches(title: string, q: string): boolean {
  if (!q) return true
  const lowerTitle = title.toLowerCase()
  const lowerQ = q.toLowerCase()
  if (lowerTitle.includes(lowerQ)) return true
  let i = 0
  for (const ch of lowerTitle) {
    if (ch === lowerQ[i]) i += 1
    if (i === lowerQ.length) return true
  }
  return false
}

const results = computed(() => props.items.filter((item) => matches(item.title, keyword.value)))

watch(
  () => props.open,
  (open) => {
    if (open) {
      keyword.value = ''
      activeIndex.value = 0
      void nextTick(() => inputRef.value?.focus())
    }
  },
)

watch(results, () => {
  activeIndex.value = 0
})

function close() {
  emit('update:open', false)
}

function choose(item: PaletteItem | undefined) {
  if (!item) return
  close()
  if (router.currentRoute.value.path !== item.path) void router.push(item.path)
}

function onKeydown(e: KeyboardEvent) {
  if (results.value.length === 0) {
    if (e.key === 'Escape') close()
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = (activeIndex.value + 1) % results.value.length
    scrollActiveIntoView()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = (activeIndex.value - 1 + results.value.length) % results.value.length
    scrollActiveIntoView()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    choose(results.value[activeIndex.value])
  } else if (e.key === 'Escape') {
    close()
  }
}

/** 键盘移动时保证命中项可见（长列表必需） */
function scrollActiveIntoView() {
  void nextTick(() => {
    listRef.value
      ?.querySelector<HTMLElement>('.palette__item--active')
      ?.scrollIntoView({ block: 'nearest' })
  })
}
</script>

<template>
  <a-modal
    :open="open"
    :footer="null"
    :closable="false"
    :width="560"
    centered
    wrap-class-name="palette-modal"
    @cancel="close"
  >
    <div class="palette" @keydown="onKeydown">
      <div class="palette__search">
        <SearchOutlined class="palette__search-icon" />
        <input
          ref="inputRef"
          v-model="keyword"
          type="text"
          class="palette__input"
          placeholder="搜索或跳转…"
          aria-label="搜索或跳转"
        />
        <kbd class="palette__kbd">Esc</kbd>
      </div>

      <div ref="listRef" class="palette__list" role="listbox" aria-label="跳转结果">
        <div v-if="results.length === 0" class="palette__empty">
          没有匹配「{{ keyword }}」的页面
        </div>
        <button
          v-for="(item, index) in results"
          :key="item.key"
          type="button"
          class="palette__item"
          :class="{ 'palette__item--active': index === activeIndex }"
          role="option"
          :aria-selected="index === activeIndex"
          @mouseenter="activeIndex = index"
          @click="choose(item)"
        >
          <span class="palette__item-title">{{ item.title }}</span>
          <span v-if="item.group" class="palette__item-group">{{ item.group }}</span>
        </button>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.palette__search {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 4px 12px;
  border-bottom: 1px solid var(--admin-border);
}

.palette__search-icon {
  flex: none;
  font-size: 15px;
  color: var(--admin-muted);
}

.palette__input {
  flex: 1;
  min-width: 0;
  height: 36px;
  font-family: inherit;
  font-size: 15px;
  color: var(--admin-text);
  background: transparent;
  border: 0;
  outline: none;
}

.palette__input::placeholder {
  color: var(--admin-muted);
}

.palette__kbd {
  flex: none;
  padding: 2px 6px;
  font-family: inherit;
  font-size: 11px;
  color: var(--admin-muted);
  background: var(--admin-surface-2);
  border: 1px solid var(--admin-border);
  border-radius: 4px;
}

.palette__list {
  max-height: 320px;
  overflow-y: auto;
  margin-top: 8px;
}

.palette__empty {
  padding: 28px 0;
  text-align: center;
  font-size: 13px;
  color: var(--admin-muted);
}

.palette__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  padding: 9px 10px;
  font-family: inherit;
  font-size: 14px;
  color: var(--admin-text);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--admin-radius-sm);
  cursor: pointer;
}

/* 选中态：品牌色浅底，键盘与鼠标共用同一视觉 */
.palette__item--active {
  background: var(--admin-brand-bg);
  color: var(--admin-brand-active);
}

.palette__item-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.palette__item-group {
  flex: none;
  font-size: 12px;
  color: var(--admin-muted);
}
</style>
