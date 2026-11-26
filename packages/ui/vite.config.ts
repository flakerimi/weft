import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue()],
  // `vite build` bundles the stylesheet only; components ship as source SFCs.
  build: {
    outDir: 'dist',
    rolldownOptions: {
      input: { weft: 'src/styles/weft.css' },
      output: { assetFileNames: '[name][extname]' },
    },
  },
  test: {
    environment: 'happy-dom',
    // A date picker's on-demand chunk, taken as fetched (see test/setup.ts).
    setupFiles: ['test/setup.ts'],
    // A fresh context per test file: the headless layer keeps page-wide state
    // (the overlay stack, scroll locks), which must not leak between files.
    // vmThreads gives each file its own context in a worker that keeps one
    // happy-dom, where a fresh worker made one per file (vitest's hint).
    isolate: true,
    pool: 'vmThreads',
    // Under Node, Vue's packages resolve to their CommonJS builds, which lack the
    // Vapor runtime. Tests load the bundler builds, the ones Vite uses for apps.
    alias: [
      { find: /^vue$/, replacement: 'vue/dist/vue.runtime.esm-bundler.js' },
      {
        find: /^@vue\/(runtime-dom|runtime-core|runtime-vapor|reactivity|shared)$/,
        replacement: '@vue/$1/dist/$1.esm-bundler.js',
      },
    ],
    server: { deps: { inline: [/[\\/]@vue[\\/]/, /[\\/]vue[\\/]/] } },
  },
})
