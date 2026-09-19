<script setup lang="ts">
import { computed } from 'vue'
import BaseTrendChart from './BaseTrendChart.vue'
import type { TrendPoint } from '@/types/api'

/**
 * 近 7 天发布/评论趋势折线图(基础组件薄壳,对外 props 契约不变)
 */
const props = defineProps<{
  data: TrendPoint[]
}>()

const series = computed(() => [
  { name: '发布文章', data: props.data.map((point) => point.posts) },
  { name: '新增评论', data: props.data.map((point) => point.comments) }
])
const labels = computed(() => props.data.map((point) => point.date.slice(5)))
</script>

<template>
  <BaseTrendChart :series="series" :labels="labels" />
</template>
