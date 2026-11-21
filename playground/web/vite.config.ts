import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import { weftIcons } from '@weft/vite'

export default defineConfig({
  // weftIcons(): the examples draw Lucide icons, inlined at build.
  plugins: [vue(), weftIcons()],
  server: { proxy: { '/api': 'http://localhost:8080' } },
  build: { outDir: 'dist', emptyOutDir: true },
})
