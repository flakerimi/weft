import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'
import { weftIcons } from './src/icons.ts'

export default defineConfig({
  // weftIcons(), as a site's build has it: components under test draw their icons.
  plugins: [vue(), weftIcons()],
  test: {
    // Node by default (renderPage); browser tests opt in with `@vitest-environment happy-dom`.
    environment: 'node',
    // A date picker's on-demand chunk, taken as fetched (see test/setup.ts).
    setupFiles: ['test/setup.ts'],
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
