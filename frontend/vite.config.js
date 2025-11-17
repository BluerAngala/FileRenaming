import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from '@unocss/vite'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    UnoCSS()
  ],
  define: {
    __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'false'
  },
  server: {
    fs: {
      strict: false
    }
  }
})
