import { defineConfig } from 'vite'

// The site's own build options: the page build replaces only the input (an SPA
// entry here) and keeps the rest, such as this external.
export default defineConfig({
  build: { rollupOptions: { input: { main: 'index.html' }, external: ['weft-test-external'] } },
})
