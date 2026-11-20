import { defineConfig } from 'vite'

// An input of the site's own (an SPA entry): the page build must not merge it into its client build.
export default defineConfig({
  build: { rollupOptions: { input: { main: 'index.html' } } },
})
