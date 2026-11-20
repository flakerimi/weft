import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue()],
  test: {
    // Node by default (renderPage); browser tests opt in with `@vitest-environment happy-dom`.
    environment: 'node',
    // Under Node, Vue's packages resolve to their CommonJS builds, which lack the
    // Vapor runtime. Tests load the bundler builds, the ones Vite uses for apps.
    // renderPage is unaffected: it imports Vue by file URL, not by these names.
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
