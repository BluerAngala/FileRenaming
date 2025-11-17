import {createApp} from 'vue'
import App from './App.vue'
import naive from 'naive-ui'
import 'virtual:uno.css'
import './style.css';

const app = createApp(App)
app.use(naive)
app.mount('#app')
