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
        timeout: 30000,
        configure: (proxy) => {
          proxy.on('error', (err) => {
            console.log('proxy error', err)
          })
        },
      },
    },
  },
  build: {
    target: 'es2020',
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        manualChunks(id) {
          const normalizedId = id.replace(/\\/g, '/')
          if (normalizedId.includes('node_modules/zrender/')) return 'zrender'
          if (normalizedId.includes('node_modules/echarts/charts')) return 'echarts-charts'
          if (normalizedId.includes('node_modules/echarts/components')) return 'echarts-components'
          if (normalizedId.includes('node_modules/echarts/renderers')) return 'echarts-renderers'
          if (normalizedId.includes('node_modules/echarts/core') || normalizedId.includes('node_modules/echarts/lib') || normalizedId.includes('node_modules/echarts/')) return 'echarts-core'
          if (id.includes('node_modules/element-plus') || id.includes('node_modules\\element-plus') || id.includes('node_modules/@element-plus') || id.includes('node_modules\\@element-plus')) return 'element-plus'
          if (id.includes('node_modules/vue') || id.includes('node_modules\\vue') || id.includes('node_modules/vue-router') || id.includes('node_modules\\vue-router') || id.includes('node_modules/pinia') || id.includes('node_modules\\pinia')) return 'vue-vendor'
        },
      },
    },
  },
  test: {
    environment: 'happy-dom',
    setupFiles: ['src/app/__tests__/setup.ts'],
    globals: true,
    css: true,
    server: {
      deps: {
        inline: ['element-plus'],
      },
    },
  },
})
