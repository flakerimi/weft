// The Astro side of the benchmark: a static build with Astro's defaults and
// its Vue integration for the counter island.
import vue from '@astrojs/vue'
import { defineConfig } from 'astro/config'

export default defineConfig({
  integrations: [vue()],
})
