import { computed, ref } from 'vue'
import type { Ref } from 'vue'
import type { TablePaginationConfig } from 'ant-design-vue'

export interface TablePageRequest {
  page: number
  pageSize: number
}

export interface PagedResult<T> {
  list: T[]
  total: number
}

export interface UseTableOptions {
  defaultPageSize?: number
  /** 分页档位(审计日志等页 20/50 起步) */
  pageSizeOptions?: string[]
  /** 初始页码(URL ?page= 恢复场景) */
  initialPage?: number
}

/**
 * 表格列表状态机(docs/09 §5.1):loading / error / 数据 / 分页 / 删除回退
 * 收敛各列表页复制样板;加载失败置 error 供页面渲染错误块,禁止落回空表伪装无数据。
 * 业务错误提示由 http 拦截器统一 toast,这里只维护页面持续状态。
 */
export function useTable<T>(
  fetcher: (req: TablePageRequest) => Promise<PagedResult<T>>,
  options: UseTableOptions = {},
) {
  const loading = ref(false)
  const error = ref('')
  const list = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const page = ref(options.initialPage ?? 1)
  const pageSize = ref(options.defaultPageSize ?? 10)
  /** 首屏判定:从未成功加载 → 页面用骨架而不是表格 spinner */
  const initialized = ref(false)

  async function load(): Promise<void> {
    loading.value = true
    error.value = ''
    try {
      const result = await fetcher({ page: page.value, pageSize: pageSize.value })
      list.value = result.list
      total.value = result.total
      initialized.value = true
    } catch {
      error.value = '数据加载失败，请检查网络后重试'
    } finally {
      loading.value = false
    }
  }

  const pagination = computed<TablePaginationConfig>(() => ({
    current: page.value,
    pageSize: pageSize.value,
    total: total.value,
    showSizeChanger: true,
    pageSizeOptions: options.pageSizeOptions ?? ['10', '20', '50'],
    showTotal: (t: number) => `共 ${t} 条`,
  }))

  function onTableChange(config: TablePaginationConfig) {
    page.value = config.current || 1
    pageSize.value = config.pageSize || 10
    void load()
  }

  /** 删除后当前页删空自动回退一页再刷新 */
  async function reloadAfterDelete(deletedCount = 1): Promise<void> {
    if (list.value.length <= deletedCount && page.value > 1) {
      page.value -= 1
    }
    await load()
  }

  return { loading, error, list, total, page, pageSize, initialized, pagination, load, onTableChange, reloadAfterDelete }
}
