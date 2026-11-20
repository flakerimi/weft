import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// A site's own config: buildSite uses it (and does not add a second vue()).
export default defineConfig({
  plugins: [vue()],
  define: { __SITE_NAME__: JSON.stringify('Static from vite.config') },
  build: { outDir: 'dist', emptyOutDir: true },
})
