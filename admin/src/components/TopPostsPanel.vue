<script setup lang="ts">
import { computed } from 'vue'
import type { TableColumnsType } from 'ant-design-vue'
import SectionPanel from './SectionPanel.vue'
import { useMediaQuery } from '@/composables/useMediaQuery'
import { formatCount, formatNumber } from '@/utils/format'
import type { AnalyticsTopPost } from '@/types/api'

/**
 * 热门文章面板：表格（桌面）/ 卡片列表（移动端）双形态。
 * 排名、数字对齐、标题省略与完整标题悬浮、无数据说明都在这里收敛，
 * 页面只负责给数据和响应事件。
 */
const props = withDefaults(
  defineProps<{
    items: AnalyticsTopPost[]
    loading?: boolean
    /** 所选范围内整站是否有访问——决定空态给出哪种解释 */
    hasSiteTraffic?: boolean
  }>(),
  { loading: false, hasSiteTraffic: false },
)

const emit = defineEmits<{ select: [postId: number]; browse: [] }>()

/** ≤768px 用卡片列表，避免把 6 列表格硬压进手机宽度 */
const isMobile = useMediaQuery('(max-width: 768px)')

/** 标题列吃掉剩余宽度，数字列固定窄宽并右对齐（便于纵向比对数量级） */
const columns = computed<TableColumnsType>(() => [
  { title: '排名', key: 'rank', width: 64, align: 'center' },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: 'PV', dataIndex: 'pv', key: 'pv', width: 96, align: 'right' },
  { title: 'UV', dataIndex: 'uv', key: 'uv', width: 96, align: 'right' },
  { title: '点赞', dataIndex: 'likeCount', key: 'likeCount', width: 88, align: 'right' },
  { title: '评论', dataIndex: 'commentCount', key: 'commentCount', width: 88, align: 'right' },
])

const emptyHint = computed(() =>
  props.hasSiteTraffic
    ? '所选范围内有访问量，但都集中在非文章页，因此没有文章上榜'
    : '所选时间范围内还没有产生访问记录，等站点有流量后这里会自动出现排行',
)

/** 紧凑数字的精确值：Number 悬浮提示用，避免用户只能读到近似值 */
function exactTitle(value: number): string {
  return formatNumber(value)
}
</script>

<template>
  <SectionPanel
    title="热门文章"
    description="按文章访问量降序，最多 10 条"
    flush
    :loading="loading"
    :empty="!loading && items.length === 0"
    empty-title="暂无文章访问记录"
    :empty-hint="emptyHint"
    :skeleton-rows="5"
  >
    <template #empty-action>
      <a-button size="small" @click="emit('browse')">前往文章管理</a-button>
    </template>

    <!-- 移动端：卡片列表 -->
    <div v-if="isMobile" class="top-posts__cards">
      <div
        v-for="(item, index) in items"
        :key="item.postId"
        class="top-post-card"
        role="button"
        tabindex="0"
        :aria-label="`第 ${index + 1} 名：${item.title}`"
        @click="emit('select', item.postId)"
        @keydown.enter.prevent="emit('select', item.postId)"
      >
        <span class="top-posts__rank" :class="{ 'top-posts__rank--top': index < 3 }">
          {{ index + 1 }}
        </span>
        <div class="top-post-card__body">
          <div class="top-post-card__title">{{ item.title }}</div>
          <div class="top-post-card__metrics">
            <span class="top-post-card__metric">
              PV <b class="tabular-nums">{{ formatCount(item.pv) }}</b>
            </span>
            <span class="top-post-card__metric">
              UV <b class="tabular-nums">{{ formatCount(item.uv) }}</b>
            </span>
            <span class="top-post-card__metric">
              赞 <b class="tabular-nums">{{ formatCount(item.likeCount) }}</b>
            </span>
            <span class="top-post-card__metric">
              评 <b class="tabular-nums">{{ formatCount(item.commentCount) }}</b>
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 桌面端：表格（表头轻量化，标题列占主空间） -->
    <a-table
      v-else
      class="table-quiet top-posts__table"
      :columns="columns"
      :data-source="items"
      :pagination="false"
      :show-sorter-tooltip="false"
      row-key="postId"
      size="middle"
    >
      <template #bodyCell="{ column, record, index }">
        <template v-if="column.key === 'rank'">
          <span class="top-posts__rank" :class="{ 'top-posts__rank--top': Number(index) < 3 }">
            {{ Number(index) + 1 }}
          </span>
        </template>
        <template v-else-if="column.key === 'title'">
          <a-tooltip :title="record.title" placement="topLeft">
            <a class="top-posts__title" @click="emit('select', record.postId)">
              {{ record.title }}
            </a>
          </a-tooltip>
        </template>
        <template v-else-if="column.key === 'pv'">
          <span class="top-posts__num tabular-nums" :title="exactTitle(record.pv)">
            {{ formatCount(record.pv) }}
          </span>
        </template>
        <template v-else-if="column.key === 'uv'">
          <span class="top-posts__num tabular-nums" :title="exactTitle(record.uv)">
            {{ formatCount(record.uv) }}
          </span>
        </template>
        <template v-else-if="column.key === 'likeCount'">
          <span class="top-posts__num tabular-nums" :title="exactTitle(record.likeCount)">
            {{ formatCount(record.likeCount) }}
          </span>
        </template>
        <template v-else-if="column.key === 'commentCount'">
          <span class="top-posts__num tabular-nums" :title="exactTitle(record.commentCount)">
            {{ formatCount(record.commentCount) }}
          </span>
        </template>
      </template>
    </a-table>
  </SectionPanel>
</template>

<style scoped>
/* ---------- 排名：前三名用柔和品牌底强调，其余保持低对比中性 ---------- */
.top-posts__rank {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--admin-muted);
  border-radius: var(--admin-radius-sm);
}

.top-posts__rank--top {
  color: var(--admin-brand);
  background: var(--admin-brand-bg);
}

/* ---------- 标题列：占主空间，超长省略，悬浮看图例外的完整标题 ---------- */
.top-posts__title {
  display: block;
  overflow: hidden;
  font-size: 14px;
  color: var(--admin-text);
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color var(--admin-dur) var(--admin-ease);
}

.top-posts__title:hover {
  color: var(--admin-brand);
}

.top-posts__num {
  display: block;
  font-size: 13px;
  color: var(--admin-text);
}

/* 表格工艺（轻表头 / 细分割线 / 行悬浮）由全局 .table-quiet 提供，此处不再重复覆盖 */

/* ---------- 移动端卡片列表 ---------- */
.top-posts__cards {
  display: flex;
  flex-direction: column;
  padding: 4px 0 0;
}

.top-post-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--admin-border);
  cursor: pointer;
  transition: background-color var(--admin-dur) var(--admin-ease);
}

.top-post-card:last-child {
  border-bottom: none;
}

.top-post-card:hover,
.top-post-card:focus-visible {
  background: var(--admin-surface-2);
  outline: none;
}

.top-post-card__body {
  flex: 1;
  min-width: 0;
}

/* 移动端允许两行，比单行省略更易读 */
.top-post-card__title {
  display: -webkit-box;
  overflow: hidden;
  font-size: 14px;
  line-height: 20px;
  color: var(--admin-text);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.top-post-card__metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  margin-top: 6px;
  font-size: 12px;
  color: var(--admin-muted);
}

.top-post-card__metric b {
  font-weight: 600;
  color: var(--admin-text);
}
</style>
