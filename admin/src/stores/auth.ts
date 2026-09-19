import { defineStore } from 'pinia'
import { getMe, login as loginApi } from '@/api/auth'
import { TOKEN_KEY } from '@/constants/status'
import type { User } from '@/types/api'

interface AuthState {
  token: string
  user: User | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: localStorage.getItem(TOKEN_KEY) || '',
    user: null,
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.token),
    /** 头像/下拉里展示的名字 */
    displayName: (state) => state.user?.nickname || state.user?.username || '管理员',
  },
  actions: {
    setToken(token: string) {
      this.token = token
      localStorage.setItem(TOKEN_KEY, token)
    },
    setUser(user: User | null) {
      this.user = user
    },
    async login(username: string, password: string, remember?: boolean) {
      const result = await loginApi({ username, password, remember })
      this.setToken(result.token)
      this.setUser(result.user)
    },
    /** 拉取当前用户信息（用于刷新头像/昵称） */
    async fetchMe() {
      try {
        const result = await getMe()
        this.setUser(result.user)
      } catch {
        // 失败不阻塞界面（401 时拦截器已统一处理跳登录）
      }
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
    },
  },
})
