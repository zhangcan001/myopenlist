import './assets/css/index.css'

import { createPinia } from 'pinia'
import { createApp } from 'vue'

import App from './App.vue'
import { setupI18n } from './i18n'
import router from './router'

const app = createApp(App)
const pinia = createPinia()
const i18n = setupI18n()

app.use(pinia)
app.use(router)
app.use(i18n)

app.mount('#app')
