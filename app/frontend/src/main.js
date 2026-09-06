import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import en from 'element-plus/es/locale/lang/en'
import './style.css'
import App from './App.vue'

createApp(App).use(ElementPlus, { locale: en }).mount('#app')
