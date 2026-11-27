import { defineConfig, type Plugin } from 'vite'

// A stylesheet a plugin makes, at /generated/sheet.css. Like the site's
// gallery plugin, it answers every query on its file, ?url included.
const generated: Plugin = {
  name: 'fixture:generated',
  enforce: 'pre',
  resolveId(id) {
    if (id.startsWith('/generated/sheet.css')) return `${__dirname}/generated/sheet.css${id.slice('/generated/sheet.css'.length)}`
  },
  load(id) {
    if (id.startsWith(`${__dirname}/generated/sheet.css`)) return '.generated { color: blue; }\n'
  },
}

export default defineConfig({ plugins: [generated] })
