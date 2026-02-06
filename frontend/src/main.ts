import { createApp } from 'vue'
import App from './App.vue'
import i18n from './i18n'
import './style.css'

// 導入 NumberFlow 以確保 Web Component 被註冊
import '@number-flow/vue'

createApp(App).use(i18n).mount('#app')
