// The SvelteKit side of the benchmark, with SvelteKit's defaults. The landing
// is prerendered in both builds (src/routes/+layout.js); adapter.js says
// which one this is.
import { sveltekit } from '@sveltejs/kit/vite'
import { defineConfig } from 'vite'
import adapter from './adapter.js'

export default defineConfig({
  plugins: [sveltekit({ adapter })],
})
