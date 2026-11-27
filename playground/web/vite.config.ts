import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import { weftIcons } from '@weft/vite'

export default defineConfig({
  // weftIcons(): the examples draw Lucide icons, inlined at build.
  plugins: [vue(), weftIcons()],
  server: { proxy: { '/api': 'http://localhost:8080' } },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // Vue's runtime in a chunk of its own, as the page build has it, beside every component's example.
    rolldownOptions: { output: { codeSplitting: { groups: [{ name: 'vue', test: /[\\/]node_modules[\\/](?:\.pnpm[\\/][^\\/]+[\\/]node_modules[\\/])?(?:vue|@vue[\\/][^\\/]+)[\\/]/ }] } } },
  },
})
