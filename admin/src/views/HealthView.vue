<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { getHealth } from '@/api/health'
import { formatTime } from '@/utils/format'
import type { HealthInfo } from '@/types/api'

const loading = ref(false)
const info = ref<HealthInfo | null>(null)

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

async function load() {
  loading.value = true
  try {
    info.value = await getHealth()
  } catch {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}

async function onRefresh() {
  await load()
  message.success('已刷新')
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader title="系统状态" description="站点运行环境与内容概览（版本号由构建注入，开发环境显示 dev）">
      <template #actions>
        <a-button :loading="loading" @click="onRefresh">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>
    </PageHeader>

    <a-spin :spinning="loading">
      <a-card :bordered="false">
        <a-descriptions :column="{ xs: 1, sm: 2 }" bordered size="middle">
          <a-descriptions-item label="MyBlog 版本">
            <a-tag color="blue">{{ info?.version ?? '-' }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="Go 版本">{{ info?.goVersion ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="数据库类型">
            {{ info?.dbType === 'sqlite' ? 'SQLite' : info?.dbType === 'mysql' ? 'MySQL' : (info?.dbType ?? '-') }}
          </a-descriptions-item>
          <a-descriptions-item label="数据库状态">
            <a-tag v-if="info?.dbStatus === 'ok'" color="success">正常</a-tag>
            <a-tag v-else-if="info" color="error">异常：{{ info.dbStatus }}</a-tag>
            <span v-else>-</span>
          </a-descriptions-item>
          <a-descriptions-item label="文章数">{{ info?.postCount ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="评论数">{{ info?.commentCount ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="媒体数量">{{ info?.mediaCount ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="上传目录大小">{{ info ? humanSize(info.uploadSize) : '-' }}</a-descriptions-item>
          <a-descriptions-item label="最近备份" :span="2">
            {{ info?.latestBackupAt ? formatTime(info.latestBackupAt) : '尚未备份（可在「备份」页创建）' }}
          </a-descriptions-item>
        </a-descriptions>

        <p class="health-note">
          采集范围刻意克制：不含磁盘空间、服务器地址等平台敏感信息。
        </p>
      </a-card>
    </a-spin>
  </div>
</template>

<style scoped>
.health-note {
  margin: 16px 0 0;
  font-size: 12px;
  color: var(--admin-muted);
}
</style>
