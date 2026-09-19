<script setup lang="ts">
import { ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { exportSite } from '@/api/backups'
import { importSite } from '@/api/data'
import type { ImportSummary } from '@/api/data'

// ---------- 导出 ----------
const exporting = ref(false)

async function onExport() {
  exporting.value = true
  try {
    await exportSite()
    message.success('导出已开始下载')
  } catch (err) {
    message.error(err instanceof Error ? err.message : '导出失败')
  } finally {
    exporting.value = false
  }
}

// ---------- 导入 ----------
const importing = ref(false)
const previewing = ref(false)
const importFile = ref<File | null>(null)
const strategy = ref<'skip' | 'update'>('skip')
const previewSummary = ref<ImportSummary | null>(null)
const conflicts = ref<{ type: string; value: string }[]>([])
const resultSummary = ref<ImportSummary | null>(null)

const summaryLabels: Record<keyof ImportSummary, string> = {
  posts: '文章',
  pages: '页面',
  categories: '分类',
  tags: '标签',
  links: '友链',
  comments: '评论',
  media: '媒体',
}

function beforeUpload(file: File): boolean {
  if (!file.name.toLowerCase().endsWith('.zip')) {
    message.error('仅支持 zip 包')
    return false
  }
  if (file.size > 50 * 1024 * 1024) {
    message.error('文件超过 50MB 上限')
    return false
  }
  importFile.value = file
  previewSummary.value = null
  conflicts.value = []
  resultSummary.value = null
  return false // 手动控制上传
}

async function onPreview() {
  if (!importFile.value) {
    message.error('请先选择 zip 包')
    return
  }
  previewing.value = true
  try {
    const result = await importSite(importFile.value, { dryRun: true, strategy: strategy.value })
    previewSummary.value = result.summary
    conflicts.value = result.conflicts
    resultSummary.value = null
    message.success('预览完成')
  } finally {
    previewing.value = false
  }
}

function onRunImport() {
  if (!importFile.value) {
    message.error('请先选择 zip 包')
    return
  }
  const conflictNote =
    conflicts.value.length > 0
      ? `检测到 ${conflicts.value.length} 处冲突，将按「${strategy.value === 'skip' ? '跳过重复' : '更新已有'}」策略处理。`
      : '未检测到冲突。'
  Modal.confirm({
    title: '执行导入',
    content: `将导入所选包中的内容。${conflictNote}导入过程会合并数据，默认不覆盖已有内容。`,
    okText: '执行导入',
    cancelText: '取消',
    onOk: async () => {
      importing.value = true
      try {
        const result = await importSite(importFile.value as File, {
          dryRun: false,
          strategy: strategy.value,
        })
        resultSummary.value = result.summary
        message.success(`导入完成，更新 ${result.updated} 条`)
      } finally {
        importing.value = false
      }
    },
  })
}

const summaryText = (summary: ImportSummary | null): string => {
  if (!summary) return ''
  return Object.entries(summary)
    .filter(([, v]) => typeof v === 'number' && (v as number) > 0)
    .map(([k, v]) => `${summaryLabels[k as keyof ImportSummary]} ${v}`)
    .join(' · ')
}
</script>

<template>
  <div class="page">
    <PageHeader title="导入导出" description="全站内容打包导出为 zip（JSON + 媒体），或从导出包恢复内容" />

    <a-card :bordered="false" class="io-card">
      <template #title>导出</template>
      <p class="io-hint">导出文章、页面、分类、标签、友链、评论、设置为 JSON，并打包 media/ 目录。</p>
      <a-button type="primary" :loading="exporting" @click="onExport">
        <template #icon><DownloadOutlined /></template>
        导出全站 zip
      </a-button>
    </a-card>

    <a-card :bordered="false" class="io-card">
      <template #title>导入</template>
      <a-space direction="vertical" :size="12" style="width: 100%">
        <a-upload
          :max-count="1"
          :before-upload="beforeUpload"
          accept=".zip"
        >
          <a-button>
            <template #icon><UploadOutlined /></template>
            选择导出 zip 包（≤50MB）
          </a-button>
        </a-upload>

        <div>
          <span class="io-label">冲突处理策略：</span>
          <a-radio-group v-model:value="strategy">
            <a-radio value="skip">跳过重复（默认，不覆盖已有内容）</a-radio>
            <a-radio value="update">更新已有（按 slug/名称覆盖）</a-radio>
          </a-radio-group>
        </div>

        <a-space wrap>
          <a-button :disabled="!importFile" :loading="previewing" @click="onPreview">预览检查</a-button>
          <a-button type="primary" danger :disabled="!importFile" :loading="importing" @click="onRunImport">
            执行导入
          </a-button>
        </a-space>

        <div v-if="importFile" class="io-file">已选择：{{ importFile.name }}</div>

        <div v-if="previewSummary" class="io-result">
          <p class="io-result__title">预览结果：{{ summaryText(previewSummary) || '包内无内容' }}</p>
          <p v-if="conflicts.length > 0" class="io-result__warn">
            检测到 {{ conflicts.length }} 处冲突：
            <a-tag v-for="(c, i) in conflicts.slice(0, 10)" :key="i" color="warning">
              {{ c.type }}: {{ c.value }}
            </a-tag>
            <span v-if="conflicts.length > 10">等</span>
          </p>
          <p v-else class="io-result__ok">无冲突</p>
        </div>

        <div v-if="resultSummary" class="io-result">
          <p class="io-result__ok">导入完成：{{ summaryText(resultSummary) || '无变化' }}</p>
        </div>
      </a-space>
    </a-card>
  </div>
</template>

<style scoped>
.io-card {
  margin-bottom: 16px;
}

.io-hint {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--admin-muted);
}

.io-label {
  font-size: 13px;
  color: var(--admin-text);
  margin-right: 8px;
}

.io-file {
  font-size: 12px;
  color: var(--admin-muted);
}

.io-result {
  padding: 10px 12px;
  background: var(--admin-surface-2);
  border-radius: var(--admin-radius-sm);
}

.io-result p {
  margin: 4px 0;
  font-size: 13px;
}

.io-result__warn {
  color: var(--admin-warning);
}

.io-result__ok {
  color: var(--admin-text);
}
</style>
