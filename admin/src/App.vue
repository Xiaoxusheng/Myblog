<script setup lang="ts">
import { ConfigProvider } from 'ant-design-vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { antdTokens } from '@/theme/tokens'
</script>

<template>
  <ConfigProvider :locale="zhCN" :theme="{ token: { ...antdTokens } }">
    <a-app>
      <RouterView />
    </a-app>
  </ConfigProvider>
</template>
