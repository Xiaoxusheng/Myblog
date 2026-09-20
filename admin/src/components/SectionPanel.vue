<script setup lang="ts">
import type { Component } from 'vue'
import { InboxOutlined } from '@ant-design/icons-vue'

/**
 * 区块面板：统一卡片语言 + 内置三态（加载 / 失败 / 空数据）。
 *
 * 存在意义：把「圆角、边框、阴影、内边距、标题层级、状态呈现」收敛到单一来源，
 * 页面只声明内容与状态，不再各自写一套卡片壳和空盒子。
 *
 * 状态优先级：error > loading > empty > 内容。
 */
withDefaults(
  defineProps<{
    title?: string
    /** 副标题：补充口径说明（如"按天分桶"），不与标题抢层级 */
    description?: string
    /**
     * card  = 主内容面板（白底 + 描边 + 轻投影）
     * plain = 次级面板（浅底、无投影、无描边），用于制造区块间的层次节奏
     */
    variant?: 'card' | 'plain'
    /** 去掉主体内边距，让表格/列表贴边铺满 */
    flush?: boolean
    /** 首次加载（无内容）：渲染骨架 */
    loading?: boolean
    /** 骨架行数 */
    skeletonRows?: number
    /** 失败态：说明发生了什么并提供重试 */
    error?: boolean
    errorText?: string
    /** 空数据态 */
    empty?: boolean
    emptyTitle?: string
    emptyHint?: string
    /** 空态图标，默认收件箱 */
    emptyIcon?: Component
  }>(),
  {
    variant: 'card',
    skeletonRows: 4,
    errorText: '数据加载失败，请检查网络或服务状态后重试',
    emptyTitle: '暂无数据',
    emptyHint: '',
  },
)

defineEmits<{ retry: [] }>()
</script>

<template>
  <section class="panel" :class="[`panel--${variant}`, { 'panel--flush': flush }]">
    <header v-if="title || description || $slots.actions" class="panel__head">
      <div class="panel__heading">
        <h3 v-if="title" class="panel__title">{{ title }}</h3>
        <p v-if="description" class="panel__desc">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="panel__actions">
        <slot name="actions" />
      </div>
    </header>

    <div class="panel__body">
      <!-- 失败：明确说明 + 恢复动作，绝不落回空数据伪装 -->
      <div v-if="error" class="panel__state">
        <a-result status="warning" title="加载失败" :sub-title="errorText" class="panel__result">
          <template #extra>
            <a-button size="small" type="primary" @click="$emit('retry')">重新加载</a-button>
          </template>
        </a-result>
      </div>

      <!-- 首次加载：骨架屏（保留区块尺寸，避免内容到位时跳版）；可传入区块自有骨架形态 -->
      <div v-else-if="loading" class="panel__state panel__state--loading">
        <slot name="skeleton">
          <a-skeleton active :title="false" :paragraph="{ rows: skeletonRows }" />
        </slot>
      </div>

      <!-- 空数据：说明 + 可选操作，而非一个空盒子 -->
      <div v-else-if="empty" class="panel__state panel__state--empty">
        <div class="panel__empty" role="status">
          <span class="panel__empty-icon" aria-hidden="true">
            <component :is="emptyIcon || InboxOutlined" />
          </span>
          <p class="panel__empty-title">{{ emptyTitle }}</p>
          <p v-if="emptyHint" class="panel__empty-hint">{{ emptyHint }}</p>
          <div v-if="$slots['empty-action']" class="panel__empty-action">
            <slot name="empty-action" />
          </div>
        </div>
      </div>

      <slot v-else />
    </div>
  </section>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--admin-surface);
  border: 1px solid var(--admin-border);
  border-radius: var(--admin-radius-md);
  box-shadow: var(--admin-shadow-sm);
}

/* 次级面板：浅底无投影，与主面板拉开层次（避免全页同一张卡片堆叠） */
.panel--plain {
  background: var(--admin-surface-2);
  border-color: transparent;
  box-shadow: none;
}

.panel--plain:hover {
  border-color: var(--admin-border);
}

.panel__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px 0;
}

.panel--plain .panel__head {
  padding: 14px 16px 0;
}

.panel__heading {
  min-width: 0;
}

.panel__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  line-height: 22px;
  letter-spacing: -0.1px;
  color: var(--admin-text);
}

.panel__desc {
  margin: 2px 0 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

.panel__actions {
  flex: none;
  display: flex;
  align-items: center;
  gap: 12px;
}

.panel__body {
  flex: 1;
  min-width: 0;
  padding: 16px 20px 20px;
}

.panel--plain .panel__body {
  padding: 12px 16px 16px;
}

/* 表格类主体：贴边铺满，由单元格自身控制内边距 */
.panel--flush .panel__body {
  padding: 8px 0 0;
}

.panel--flush .panel__head {
  padding-bottom: 4px;
}

.panel__state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 180px;
  padding: 8px 0;
}

.panel--flush .panel__state {
  padding-bottom: 20px;
}

.panel__state--loading {
  display: block;
  min-height: 0;
}

.panel__result {
  padding: 0;
}

/* ---------- 空态：图标 + 说明 + 可选操作 ---------- */
.panel__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-width: 360px;
}

.panel__empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin-bottom: 10px;
  font-size: 18px;
  color: var(--admin-muted);
  background: var(--admin-surface-2);
  border-radius: 50%;
}

.panel--plain .panel__empty-icon {
  background: var(--admin-surface);
}

.panel__empty-title {
  margin: 0;
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  color: var(--admin-text);
}

.panel__empty-hint {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-muted);
}

.panel__empty-action {
  margin-top: 12px;
}
</style>
