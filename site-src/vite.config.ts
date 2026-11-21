import { defineConfig } from 'vite'

// The repository is private for now: the header links it only when the
// build is told it is public, with WEFT_REPO_PUBLIC=1.
export default defineConfig({
  define: { __WEFT_REPO_PUBLIC__: JSON.stringify(process.env.WEFT_REPO_PUBLIC === '1') },
})
