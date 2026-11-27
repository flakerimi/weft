import { defineConfig } from 'vite'
import { gallery } from './gallery/plugin.ts'
import { siteThemes } from './themes/plugin.ts'

// The repository is private for now: the header links it only when the
// build is told it is public, with WEFT_REPO_PUBLIC=1. The gallery plugin
// makes the /components pages' modules from the component catalogue; the
// site-themes plugin, the stylesheets of the header's Theme menu.
export default defineConfig({
  define: { __WEFT_REPO_PUBLIC__: JSON.stringify(process.env.WEFT_REPO_PUBLIC === '1') },
  plugins: [gallery({ catalog: '../catalog/components.json' }), siteThemes()],
})
