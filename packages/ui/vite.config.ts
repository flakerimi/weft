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
    // A fresh worker per test file: the headless layer keeps page-wide state
    // (the overlay stack, scroll locks), which must not leak between files.
    // Set explicitly, which also settles vitest's hint about turning it off.
    isolate: true,
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
