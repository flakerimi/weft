// The Nuxt side of the benchmark, with Nuxt's defaults. The landing is
// prerendered in both builds: `nuxt generate` writes the static site, and
// `nuxt build` the Node server (Nitro) that serves the same prerendered page.
// BENCH_OUT keeps the two outputs apart.
export default defineNuxtConfig({
  compatibilityDate: '2026-11-01',
  telemetry: false,
  css: ['../shared/landing.css'],
  app: { head: { htmlAttrs: { lang: 'en' } } },
  routeRules: { '/': { prerender: true } },
  nitro: { output: { dir: process.env.BENCH_OUT ?? '.output' } },
})
