<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import { getPageRevisions, getPageRevision, restorePageRevision } from '@/api/pages'
import { countLines, diffLines, DIFF_RENDER_LINE_LIMIT } from '@/utils/diff'
import { formatTime } from '@/utils/format'
import type { PageItem, PageRevisionItem } from '@/types/api'

/**
 * 页面版本历史抽屉：版本列表 / 行级对比（统一视图 Diff）/ 恢复。
 * 与 PostRevisionDrawer 同范式——恢复由服务端先把当前内容快照为新版本
 * （「恢复前快照」），再应用目标版本，因此恢复本身可再次恢复撤销。
 * 恢复只应用 title/slug/content，不改当前发布状态与计划时间。
 */
const props = defineProps<{
  open: boolean
  pageId: number | null
  /** 父组件当前编辑器内容，作为「当前版本」参与对比 */
  currentContent: string
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'restored', page: PageItem): void
}>()

const PAGE_SIZE = 20

// ---------------------------------------------------------------------------
// 版本列表（打开抽屉时才加载，避免无谓请求）
// ---------------------------------------------------------------------------
const list = ref<PageRevisionItem[]>([])
const total = ref(0)
const page = ref(1)
const listLoading = ref(false)
const listFailed = ref(false)

async function loadList() {
  if (props.pageId === null) return
  listLoading.value = true
  listFailed.value = false
  try {
    const result = await getPageRevisions(props.pageId, { page: page.value, pageSize: PAGE_SIZE })
    list.value = result.list
    total.value = result.total
  } catch {
    listFailed.value = true
  } finally {
    listLoading.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      page.value = 1
      baseVersion.value = null
      void loadList()
    }
  },
)

function changePage(next: number) {
  page.value = next
  void loadList()
}

function close() {
  emit('update:open', false)
}

// ---------------------------------------------------------------------------
// 对比（统一视图 Diff）：基准 = 选中的版本，对比对象 = 当前版本或其他版本
// ---------------------------------------------------------------------------
const baseVersion = ref<number | null>(null)
const compareTarget = ref<'current' | number>('current')
const baseContent = ref('')
/** 对比对象为历史版本时的内容；为「当前版本」时直接使用 props.currentContent（保持实时） */
const fetchedContent = ref('')
const compareLoading = ref(false)
let compareSeq = 0

const targetOptions = computed(() => {
  const options: { label: string; value: 'current' | number }[] = [
    { label: '当前版本', value: 'current' },
  ]
  for (const item of list.value) {
    if (item.version === baseVersion.value) continue
    options.push({ label: `v${item.version}`, value: item.version })
  }
  return options
})

const targetContent = computed(() =>
  compareTarget.value === 'current' ? props.currentContent : fetchedContent.value,
)

const diffRows = computed(() => diffLines(baseContent.value, targetContent.value))

const diffStats = computed(() => {
  let add = 0
  let del = 0
  for (const row of diffRows.value) {
    if (row.type === 'add') add += 1
    else if (row.type === 'del') del += 1
  }
  return { add, del }
})

/** 单侧超过渲染上限时不逐行渲染，只显示统计与提示 */
const overRenderLimit = computed(
  () =>
    countLines(baseContent.value) > DIFF_RENDER_LINE_LIMIT ||
    countLines(targetContent.value) > DIFF_RENDER_LINE_LIMIT,
)

async function openCompare(version: number) {
  const pageIdValue = props.pageId
  if (pageIdValue === null) return
  const seq = ++compareSeq
  baseVersion.value = version
  compareTarget.value = 'current'
  fetchedContent.value = ''
  compareLoading.value = true
  try {
    const result = await getPageRevision(pageIdValue, version)
    if (seq !== compareSeq) return
    baseContent.value = result.revision.content
  } finally {
    if (seq === compareSeq) {
      compareLoading.value = false
    }
  }
}

async function onTargetChange() {
  const pageIdValue = props.pageId
  if (pageIdValue === null || baseVersion.value === null) return
  if (compareTarget.value === 'current') {
    fetchedContent.value = ''
    return
  }
  const seq = ++compareSeq
  compareLoading.value = true
  try {
    const result = await getPageRevision(pageIdValue, compareTarget.value)
    if (seq !== compareSeq) return
    fetchedContent.value = result.revision.content
  } finally {
    if (seq === compareSeq) {
      compareLoading.value = false
    }
  }
}

// ---------------------------------------------------------------------------
// 恢复：二次确认 → 服务端先快照当前内容再应用目标版本
// ---------------------------------------------------------------------------
const restoringVersion = ref<number | null>(null)

function onRestore(item: PageRevisionItem) {
  const pageIdValue = props.pageId
  if (pageIdValue === null) return
  Modal.confirm({
    title: `恢复到 v${item.version}`,
    content: `将用 v${item.version} 的内容覆盖当前内容；当前内容会先自动保存为新版本，可再次恢复撤销。页面的发布状态与计划发布时间不会改变。`,
    okText: '恢复',
    cancelText: '取消',
    okButtonProps: { danger: true },
    onOk: async () => {
      if (props.pageId === null) return
      restoringVersion.value = item.version
      try {
        const result = await restorePageRevision(props.pageId, item.version)
        message.success(`已恢复到 v${item.version}，编辑器内容已同步更新`)
        emit('restored', result.page)
        // 服务端已生成「恢复前快照」新版本，回到列表首页刷新
        baseVersion.value = null
        page.value = 1
        await loadList()
      } finally {
        restoringVersion.value = null
      }
    },
  })
}
</script>

<template>
  <a-drawer :open="open" title="版本历史" width="min(720px, 100vw)" @close="close">
    <!-- 版本列表 -->
    <template v-if="baseVersion === null">
      <a-spin :spinning="listLoading">
        <div v-if="listFailed" class="revision-drawer__empty">
          <a-empty description="版本列表加载失败">
            <a-button size="small" @click="loadList">重试</a-button>
          </a-empty>
        </div>
        <div v-else-if="list.length === 0" class="revision-drawer__empty">
          <a-empty description="暂无历史版本，保存页面后自动生成" />
        </div>
        <div v-else class="revision-drawer__list">
          <div v-for="item in list" :key="item.id" class="revision-item">
            <span class="revision-item__version tabular-nums">v{{ item.version }}</span>
            <div class="revision-item__meta">
              <div class="revision-item__remark">{{ item.remark || '保存' }}</div>
              <div class="revision-item__time">{{ formatTime(item.createdAt) }}</div>
            </div>
            <a-space :size="0" class="revision-item__actions">
              <a-button type="link" size="small" @click="openCompare(item.version)">对比</a-button>
              <a-button
                type="link"
                size="small"
                :loading="restoringVersion === item.version"
                @click="onRestore(item)"
              >
                恢复
              </a-button>
            </a-space>
          </div>
          <a-pagination
            v-if="total > PAGE_SIZE"
            :current="page"
            :total="total"
            :page-size="PAGE_SIZE"
            size="small"
            class="revision-drawer__pager"
            @change="changePage"
          />
        </div>
      </a-spin>
    </template>

    <!-- 对比视图 -->
    <template v-else>
      <div class="revision-compare__toolbar">
        <a-button size="small" @click="baseVersion = null">
          <template #icon><ArrowLeftOutlined /></template>
          返回
        </a-button>
        <span class="revision-compare__base tabular-nums">基准 v{{ baseVersion }}</span>
        <span class="revision-compare__arrow">→</span>
        <a-select
          v-model:value="compareTarget"
          :options="targetOptions"
          size="small"
          class="revision-compare__select"
          @change="onTargetChange"
        />
        <span v-if="diffStats.add === 0 && diffStats.del === 0" class="revision-compare__same">
          内容一致
        </span>
        <span v-else class="revision-compare__stats tabular-nums">
          +{{ diffStats.add }} -{{ diffStats.del }} 行
        </span>
      </div>

      <a-spin :spinning="compareLoading">
        <div v-if="overRenderLimit" class="revision-compare__overflow">
          内容超过 {{ DIFF_RENDER_LINE_LIMIT }} 行，已省略逐行对比，仅显示差异统计。
        </div>
        <div v-else class="revision-compare__diff">
          <div
            v-for="(row, index) in diffRows"
            :key="index"
            class="diff-row"
            :class="`diff-row--${row.type}`"
          >
            <span class="diff-row__marker">{{
              row.type === 'add' ? '+' : row.type === 'del' ? '-' : ' '
            }}</span>
            <span class="diff-row__text">{{ row.text }}</span>
          </div>
        </div>
      </a-spin>
    </template>
  </a-drawer>
</template>

<style scoped>
.revision-drawer__empty {
  padding: 48px 0;
}

/* 版本列表 */
.revision-drawer__list {
  display: flex;
  flex-direction: column;
}

.revision-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 4px;
  border-bottom: 1px solid var(--admin-border);
}

.revision-item:last-child {
  border-bottom: none;
}

.revision-item__version {
  flex: none;
  min-width: 44px;
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-text);
}

.revision-item__meta {
  flex: 1;
  min-width: 0;
}

.revision-item__remark {
  font-size: 13px;
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.revision-item__time {
  margin-top: 2px;
  font-size: 12px;
  color: var(--admin-muted);
}

.revision-item__actions {
  flex: none;
}

.revision-drawer__pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

/* 对比视图 */
.revision-compare__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.revision-compare__base {
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-text);
}

.revision-compare__arrow {
  color: var(--admin-muted);
}

.revision-compare__select {
  width: 150px;
}

.revision-compare__stats,
.revision-compare__same {
  margin-left: auto;
  font-size: 12px;
  color: var(--admin-muted);
}

.revision-compare__overflow {
  padding: 16px;
  font-size: 13px;
  color: var(--admin-muted);
  background: var(--admin-surface-2);
  border: 1px dashed var(--admin-border);
  border-radius: var(--admin-radius-md);
}

/* Diff：等宽字体 + 低饱和底色，长行换行不产生横向滚动 */
.revision-compare__diff {
  max-height: calc(100vh - 220px);
  overflow: auto;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12px;
  line-height: 20px;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
}

.diff-row {
  display: flex;
  min-height: 20px;
}

.diff-row__marker {
  flex: none;
  width: 24px;
  padding: 0 4px;
  color: var(--admin-muted);
  text-align: center;
  user-select: none;
}

.diff-row__text {
  flex: 1;
  min-width: 0;
  padding-right: 12px;
  color: var(--admin-text);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.diff-row--del {
  background: rgba(255, 77, 79, 0.08);
}

.diff-row--del .diff-row__marker {
  color: var(--admin-danger);
}

.diff-row--add {
  background: rgba(82, 196, 26, 0.08);
}

.diff-row--add .diff-row__marker {
  color: var(--admin-success);
}
</style>
