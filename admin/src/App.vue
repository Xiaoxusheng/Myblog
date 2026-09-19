<script setup lang="ts">
import { ConfigProvider } from 'ant-design-vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { useAdminTheme } from '@/theme'

const { themeConfig } = useAdminTheme()
</script>

<template>
  <ConfigProvider :locale="zhCN" :theme="themeConfig">
    <a-app>
      <RouterView />
    </a-app>
  </ConfigProvider>
</template>
