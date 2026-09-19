<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import type { TableColumnsType, TablePaginationConfig } from 'ant-design-vue'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons-vue'
import PageHeader from '@/components/PageHeader.vue'
import { deletePost, getPosts, updatePostStatus } from '@/api/posts'
import { getCategories } from '@/api/taxonomy'
import { POST_STATUS_MAP } from '@/constants/status'
import { formatTime } from '@/utils/format'
import type { AdminPostItem, Category, PostStatus } from '@/types/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const list = ref<AdminPostItem[]>([])
const total = ref(0)
const page = ref(Number(route.query.page) || 1)
const pageSize = ref(10)
const categories = ref<Category[]>([])

// 筛选条件：keyword 需回车/点击搜索才生效
const keywordInput = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const keyword = ref(keywordInput.value)
const status = ref<PostStatus | ''>(
  route.query.status !== undefined && route.query.status !== '' ? (Number(route.query.status) as PostStatus) : '',
)
const categoryId = ref<number | ''>(
  route.query.categoryId !== undefined && route.query.categoryId !== '' ? Number(route.query.categoryId) : '',
)

const columns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', key: 'title', width: 240, ellipsis: true },
  { title: '分类', dataIndex: ['category', 'name'], key: 'category', width: 110 },
  { title: '标签', key: 'tags', width: 160 },
  { title: '状态', key: 'status', width: 120 },
  { title: '浏览/评论', key: 'stats', width: 90 },
  { title: '发布时间', key: 'publishedAt', width: 130 },
  { title: '操作', key: 'action', width: 160, fixed: 'right' },
]

const pagination = computed<TablePaginationConfig>(() => ({
  current: page.value,
  pageSize: pageSize.value,
  total: total.value,
  showSizeChanger: true,
  pageSizeOptions: ['10', '20', '50'],
  showTotal: (t: number) => `共 ${t} 条`,
}))

function syncQuery() {
  void router.replace({
    query: {
      page: page.value > 1 ? String(page.value) : undefined,
      keyword: keyword.value || undefined,
      status: status.value !== '' ? String(status.value) : undefined,
      categoryId: categoryId.value !== '' ? String(categoryId.value) : undefined,
    },
  })
}

async function load() {
  loading.value = true
  syncQuery()
  try {
    const result = await getPosts({
      keyword: keyword.value || undefined,
      status: status.value === '' ? undefined : status.value,
      categoryId: categoryId.value === '' ? undefined : categoryId.value,
      page: page.value,
      pageSize: pageSize.value,
    })
    list.value = result.list
    total.value = result.total
  } finally {
    loading.value = false
  }
}

async function loadCategories() {
  const result = await getCategories({ page: 1, pageSize: 100 })
  categories.value = result.list
}

function onSearch() {
  keyword.value = keywordInput.value.trim()
  page.value = 1
  void load()
}

function onFilterChange() {
  page.value = 1
  void load()
}

function onTableChange(paginationConfig: TablePaginationConfig) {
  page.value = paginationConfig.current || 1
  pageSize.value = paginationConfig.pageSize || 10
  void load()
}

async function togglePublish(record: AdminPostItem) {
  const next: PostStatus = record.status === 1 ? 2 : 1
  await updatePostStatus(record.id, next)
  message.success(next === 1 ? '已发布' : '已下架')
  await load()
}

async function onDelete(record: AdminPostItem) {
  await deletePost(record.id)
  message.success('删除成功')
  // 当前页删空后回退一页
  if (list.value.length === 1 && page.value > 1) {
    page.value -= 1
  }
  await load()
}

function goCreate() {
  void router.push('/posts/edit')
}

function goEdit(record: AdminPostItem) {
  void router.push(`/posts/edit/${record.id}`)
}

onMounted(() => {
  void load()
  void loadCategories()
})
</script>

<template>
  <div class="page">
    <PageHeader title="文章管理" description="管理博客文章：草稿、发布、隐藏与置顶">
      <template #actions>
        <a-button type="primary" @click="goCreate">
          <template #icon><PlusOutlined /></template>
          新建文章
        </a-button>
      </template>
    </PageHeader>

    <a-card :bordered="false">
      <div class="table-toolbar">
        <div class="table-toolbar__filters">
          <a-input
            v-model:value="keywordInput"
            placeholder="搜索标题 / 摘要 / 内容"
            style="width: 220px"
            allow-clear
            @press-enter="onSearch"
          />
          <a-select
            v-model:value="status"
            placeholder="状态"
            style="width: 120px"
            allow-clear
            :options="[
              { label: '草稿', value: 0 },
              { label: '已发布', value: 1 },
              { label: '隐藏', value: 2 },
            ]"
            @change="onFilterChange"
          />
          <a-select
            v-model:value="categoryId"
            placeholder="分类"
            style="width: 140px"
            allow-clear
            :options="categories.map((item) => ({ label: item.name, value: item.id }))"
            @change="onFilterChange"
          />
          <a-button @click="onSearch">
            <template #icon><SearchOutlined /></template>
            搜索
          </a-button>
        </div>
        <a-button :loading="loading" @click="load">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </div>

      <a-table
        :columns="columns"
        :data-source="list"
        :loading="loading"
        :pagination="pagination"
        :scroll="{ x: 1010 }"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'title'">
            <a class="post-title" @click="goEdit(record)">{{ record.title }}</a>
            <div class="cell-secondary">/ {{ record.slug }}</div>
          </template>

          <template v-else-if="column.key === 'category'">
            {{ record.category ? record.category.name : '未分类' }}
          </template>

          <template v-else-if="column.key === 'tags'">
            <a-space :size="4" wrap>
              <a-tag v-for="tag in record.tags.slice(0, 3)" :key="tag.id">{{ tag.name }}</a-tag>
              <a-tag v-if="record.tags.length > 3">+{{ record.tags.length - 3 }}</a-tag>
              <span v-if="record.tags.length === 0" class="cell-secondary">-</span>
            </a-space>
          </template>

          <template v-else-if="column.key === 'status'">
            <a-tag :color="POST_STATUS_MAP[record.status as PostStatus].color">
              {{ POST_STATUS_MAP[record.status as PostStatus].text }}
            </a-tag>
            <a-tag v-if="record.isTop" color="orange">置顶</a-tag>
          </template>

          <template v-else-if="column.key === 'stats'">
            {{ record.viewCount }} / {{ record.commentCount }}
          </template>

          <template v-else-if="column.key === 'publishedAt'">
            {{ formatTime(record.publishedAt || record.createdAt) }}
          </template>

          <template v-else-if="column.key === 'action'">
            <a-space :size="0">
              <a-button type="link" size="small" @click="goEdit(record)">编辑</a-button>
              <a-button type="link" size="small" @click="togglePublish(record)">
                {{ record.status === 1 ? '下架' : '发布' }}
              </a-button>
              <a-popconfirm
                title="删除后不可恢复，该文章的标签关联与评论将一并删除，确定删除吗？"
                ok-text="删除"
                cancel-text="取消"
                @confirm="onDelete(record)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<style scoped>
.post-title {
  color: var(--admin-text);
}

.post-title:hover {
  color: var(--admin-brand);
}
</style>
