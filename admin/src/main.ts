import { createApp } from 'vue'
import { createPinia } from 'pinia'
import dayjs from 'dayjs'
import Antd from 'ant-design-vue'

import App from './App.vue'
import router from './router'

import 'ant-design-vue/dist/reset.css'
import './styles/main.css'

dayjs.locale('zh-cn')

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(Antd)
app.mount('#app')
