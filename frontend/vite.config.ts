/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  root: __dirname,
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:5070',
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'es2020',
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/echarts') || id.includes('node_modules\\echarts')) return 'echarts'
          if (id.includes('node_modules/element-plus') || id.includes('node_modules\\element-plus') || id.includes('node_modules/@element-plus') || id.includes('node_modules\\@element-plus')) return 'element-plus'
          if (id.includes('node_modules/vue') || id.includes('node_modules\\vue') || id.includes('node_modules/vue-router') || id.includes('node_modules\\vue-router') || id.includes('node_modules/pinia') || id.includes('node_modules\\pinia')) return 'vue-vendor'
        },
      },
    },
  },
  test: {
    environment: 'happy-dom',
    setupFiles: ['src/__tests__/setup.ts'],
    globals: true,
    css: true,
    server: {
      deps: {
        inline: ['element-plus'],
      },
    },
  },
})
