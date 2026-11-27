// Browser tests for the landing blocks and Grid on a built static page (zero
// JavaScript, no <Island>), in narrow 360 px side-by-side boxes on a wide
// 1440 px page, with long unbroken words at 375 px, and across all three
// themes and both schemes for WCAG AA contrast (C1, I1–I4).
import { spawn, type ChildProcess } from 'node:child_process'
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import type { AddressInfo } from 'node:net'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite } from '../src/build.ts'
import type { Theme } from '../src/document.ts'

function findBrowser(): string | null {
  if (process.env.WEFT_BROWSER && existsSync(process.env.WEFT_BROWSER)) return process.env.WEFT_BROWSER
  for (const p of [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
  ]) {
    if (existsSync(p)) return p
  }
  return null
}

const browserPath = findBrowser()
const longWord = 'supercalifragilisticexpialidociousunbrokencompoundwordforoverflowtestingatnarrowwidth87c'

const STATIC_PAGE = `<script setup vapor lang="ts">
import { Card, Cta, FeatureGrid, Hero, Pricing, Testimonials, type FeatureItem, type PricingPlan, type TestimonialItem } from '@weft/ui'

const features: FeatureItem[] = [
  { icon: 'zap', title: 'Vapor DOM', description: 'Direct DOM operations.' },
  { icon: 'package', title: 'One binary', description: 'Pre-compressed pages and API.' },
  { icon: 'shield', title: 'Typed routes', description: 'Go types to TypeScript.' },
]

const plans: PricingPlan[] = [
  { title: 'Starter', description: 'For side projects.', price: '$0', period: '/month', features: ['1 workspace', 'SQLite'], action: { label: 'Start for free', to: '/start' } },
  { title: 'Pro', description: 'For product teams.', price: '$29', period: '/month', badge: 'Popular', highlight: true, features: ['Unlimited workspaces', 'Team roles'], action: { label: 'Start free trial', to: '/pro' } },
  { title: 'Enterprise', description: 'Custom deployment.', price: '$99', period: '/month', features: ['Dedicated support', 'SLA'], action: { label: 'Contact sales', to: '/sales' } },
]

const quotes: TestimonialItem[] = [
  { quote: 'Deploying went from three containers to one binary.', author: 'Avery Quinn', role: 'CTO at Northline' },
  { quote: 'Forms and Go handlers share one schema.', author: 'Morgan Patel', role: 'Principal Engineer at Kestrel' },
  { quote: 'Our docs ship zero JavaScript.', author: 'Rowan Lindqvist', role: 'Director at Vela Cloud' },
]
</script>

<template>
  <main>
    <Hero
      id="hero-h"
      orientation="horizontal"
      headline="Pre-alpha"
      title="One component tree, every screen"
      description="Static sections adapt in CSS first."
      :actions="[{ label: 'Install Weft', to: '/install' }, { label: 'Read docs', to: '/docs' }]"
    >
      <Card>Preview frame</Card>
    </Hero>
    <FeatureGrid id="fg" headline="Capabilities" title="Built together" :items="features" />
    <Pricing id="pr" headline="Pricing" title="Predictable plans" :plans="plans" />
    <Testimonials id="te" headline="Voices" title="Teams shipping" :items="quotes" />
    <Cta
      id="cta-solid"
      orientation="horizontal"
      variant="solid"
      title="Ship one binary today"
      description="No separate Node process in production."
      :actions="[{ label: 'Get started', to: '/install' }, { label: 'Later', to: '/docs' }]"
    />
    <Cta
      id="cta-soft"
      orientation="horizontal"
      variant="soft"
      title="Try the example shop"
      description="SQLite, auth and a product table in one module."
      :actions="[{ label: 'Explore', to: '/shop' }]"
    />
  </main>
</template>
`

const PAIR_PAGE = `<script setup vapor lang="ts">
import { Card, Cta, FeatureGrid, Hero, Pricing } from '@weft/ui'

const features = [
  { title: 'First', description: 'One' },
  { title: 'Second', description: 'Two' },
  { title: 'Third', description: 'Three' },
]
const plans = [
  { title: 'Starter', price: '$0', period: '/mo', features: ['A', 'B'], action: { label: 'Start for free' } },
  { title: 'Pro', price: '$29', period: '/mo', highlight: true, features: ['C', 'D'], action: { label: 'Go Pro' } },
  { title: 'Max', price: '$99', period: '/mo', features: ['E', 'F'], action: { label: 'Contact' } },
]
</script>

<template>
  <div style="display: flex; gap: 24px; align-items: flex-start; padding: 16px;">
    <div id="box-a" style="width: 360px; flex: none; border: 1px solid #ccc;">
      <Hero id="pair-hero" orientation="horizontal" title="Narrow box hero" description="Inside 360 px">
        <Card>Media</Card>
      </Hero>
      <FeatureGrid id="pair-fg" title="Narrow features" :items="features" />
    </div>
    <div id="box-b" style="width: 360px; flex: none; border: 1px solid #ccc;">
      <Pricing id="pair-pr" title="Narrow pricing" :plans="plans" />
      <Cta id="pair-cta" orientation="horizontal" title="Narrow CTA" :actions="[{ label: 'Act' }, { label: 'Skip' }]" />
    </div>
  </div>
</template>
`

const LONG_WORD_PAGE = `<script setup vapor lang="ts">
import { Cta, FeatureGrid, Hero, Pricing, Testimonials } from '@weft/ui'
const w = '${longWord}'
</script>

<template>
  <main>
    <Hero :headline="w" :title="w" :description="w" :actions="[{ label: 'Go' }]" />
    <FeatureGrid :title="w" :items="[{ title: w, description: w }]" />
    <Pricing :title="w" :plans="[{ title: w, description: w, price: '$29', badge: 'Tag', features: [w], action: { label: w } }]" />
    <Testimonials :title="w" :items="[{ quote: w, author: w, role: w }]" />
    <Cta :title="w" :description="w" :actions="[{ label: 'Act' }]" />
  </main>
</template>
`

const CYCLES_ISLAND = `<script setup vapor lang="ts">
import { ref } from 'vue'
import { Pricing } from '@weft/ui'

const cycle = ref('yearly')
const cycles = [
  { label: 'Monthly', value: 'monthly' },
  { label: 'Yearly', value: 'yearly', badge: 'Save 20%' },
]
const plans = [
  { title: 'Pro', price: { monthly: '$30', yearly: '$24' }, period: '/mo', highlight: true },
]
</script>

<template>
  <Pricing id="pr-cycles" v-model:cycle="cycle" title="Cycles" :cycles="cycles" :plans="plans" />
</template>
`

const CYCLES_PAGE = `<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import CyclesIsland from '../components/CyclesIsland.vue'
</script>

<template>
  <main>
    <Island when="load" :component="CyclesIsland" />
  </main>
</template>
`

describe.skipIf(!browserPath)('landing blocks in a real browser (C1, I1–I4)', { timeout: 120_000 }, () => {
  let dir: string
  const servers: Server[] = []
  const urls: Record<Theme, string> = { neutral: '', lisa: '', weft: '' }
  let browser: ChildProcess | null = null
  let ws: WebSocket | null = null
  let seq = 0
  const waiting = new Map<number, (m: { result?: unknown; error?: { message: string } }) => void>()

  const send = (method: string, params: Record<string, unknown> = {}) =>
    new Promise<Record<string, unknown>>((resolve, reject) => {
      const id = ++seq
      waiting.set(id, (m) => (m.error ? reject(new Error(`${method}: ${m.error.message}`)) : resolve((m.result ?? {}) as Record<string, unknown>)))
      ws!.send(JSON.stringify({ id, method, params }))
    })

  async function evaluate<T>(expression: string): Promise<T> {
    const r = (await send('Runtime.evaluate', { expression: `(() => (${expression}))()`, awaitPromise: true, returnByValue: true })) as {
      result?: { value: T }
      exceptionDetails?: { text?: string; exception?: { description?: string } }
    }
    if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description ?? r.exceptionDetails.text ?? 'evaluate failed')
    return r.result!.value
  }

  beforeAll(async () => {
    dir = mkdtempSync(join(tmpdir(), 'weft-landing-browser-'))
    const root = join(dir, 'site')
    mkdirSync(join(root, 'pages'), { recursive: true })
    mkdirSync(join(root, 'components'), { recursive: true })
    symlinkSync(fileURLToPath(new URL('../node_modules', import.meta.url)), join(root, 'node_modules'))
    writeFileSync(join(root, 'pages/static.vue'), STATIC_PAGE)
    writeFileSync(join(root, 'pages/pair.vue'), PAIR_PAGE)
    writeFileSync(join(root, 'pages/long.vue'), LONG_WORD_PAGE)
    writeFileSync(join(root, 'components/CyclesIsland.vue'), CYCLES_ISLAND)
    writeFileSync(join(root, 'pages/cycles.vue'), CYCLES_PAGE)

    for (const theme of ['neutral', 'lisa', 'weft'] as const) {
      const outDir = join(dir, `dist-${theme}`)
      await buildSite({ root, outDir, theme })
      const srv = createServer((req, res) => {
        const u = (req.url ?? '/').split('?')[0]!
        const file = u.startsWith('/assets/') ? join(outDir, u) : join(outDir, 'pages', u === '/' ? 'index' : u.slice(1), 'index.html')
        if (!existsSync(file)) {
          res.statusCode = 404
          res.end('not found')
          return
        }
        res.setHeader('Content-Type', file.endsWith('.js') ? 'text/javascript; charset=utf-8' : 'text/html; charset=utf-8')
        res.end(readFileSync(file))
      })
      await new Promise<void>((done) => srv.listen(0, '127.0.0.1', done))
      servers.push(srv)
      urls[theme] = `http://127.0.0.1:${(srv.address() as AddressInfo).port}`
    }

    const profile = join(dir, 'profile')
    mkdirSync(profile, { recursive: true })
    browser = spawn(
      browserPath!,
      ['--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`, '--no-first-run', '--no-default-browser-check', 'about:blank'],
      { stdio: 'ignore' },
    )
    let port = ''
    for (let i = 0; i < 200 && !port; i++) {
      try {
        port = readFileSync(join(profile, 'DevToolsActivePort'), 'utf8').split('\n')[0]!
      } catch {
        await new Promise((r) => setTimeout(r, 50))
      }
    }
    let page: { webSocketDebuggerUrl: string } | undefined
    for (let i = 0; i < 100 && !page; i++) {
      try {
        const list = (await (await fetch(`http://127.0.0.1:${port}/json/list`)).json()) as { type: string; webSocketDebuggerUrl: string }[]
        page = list.find((t) => t.type === 'page')
      } catch {}
      if (!page) await new Promise((r) => setTimeout(r, 50))
    }
    ws = new WebSocket(page!.webSocketDebuggerUrl)
    await new Promise((resolve, reject) => {
      ws!.addEventListener('open', resolve)
      ws!.addEventListener('error', reject)
    })
    ws.addEventListener('message', (ev) => {
      const m = JSON.parse(String(ev.data)) as { id?: number; result?: unknown; error?: { message: string } }
      if (m.id && waiting.has(m.id)) {
        waiting.get(m.id)!(m)
        waiting.delete(m.id)
      }
    })
    await send('Page.enable')
    await send('Runtime.enable')
  })

  afterAll(async () => {
    ws?.close()
    browser?.kill('SIGKILL')
    for (const s of servers) await new Promise<void>((done) => s.close(() => done()))
    if (dir) rmSync(dir, { recursive: true, force: true })
  })

  async function openAt(url: string, width: number, scheme: 'light' | 'dark' = 'light') {
    await send('Emulation.setDeviceMetricsOverride', { width, height: 900, deviceScaleFactor: 1, mobile: false })
    await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: scheme }] })
    await send('Page.navigate', { url })
    for (let i = 0; i < 100; i++) {
      const state = await evaluate<string>('document.readyState')
      if (state === 'complete') break
      await new Promise((r) => setTimeout(r, 30))
    }
    await evaluate(`(document.documentElement.dataset.weftScheme = ${JSON.stringify(scheme)}, true)`)
    await new Promise((r) => setTimeout(r, 40))
  }

  it('C1: a built static page with no islands adapts to 1 column at 375 px, 2 at 768 px, and 3 / side-by-side at 1440 px, with no horizontal overflow', async () => {
    const html = readFileSync(join(dir, 'dist-neutral/pages/static/index.html'), 'utf8')
    // Pure static HTML: no module script on the page.
    expect(html).not.toContain('type="module"')

    await openAt(`${urls.neutral}/static`, 375)
    const at375 = await evaluate<{
      scrollWidth: number
      fgCols: number
      prCols: number
      teCols: number
      heroStacked: boolean
      ctaStacked: boolean
    }>(`{
      scrollWidth: document.documentElement.scrollWidth,
      fgCols: getComputedStyle(document.querySelector('#fg .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      prCols: getComputedStyle(document.querySelector('#pr .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      teCols: getComputedStyle(document.querySelector('#te .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      heroStacked: getComputedStyle(document.querySelector('#hero-h .weft-hero__inner')).gridTemplateColumns.split(' ').length === 1,
      ctaStacked: getComputedStyle(document.querySelector('#cta-solid .weft-cta__box')).gridTemplateColumns.split(' ').length === 1,
    }`)
    expect(at375).toEqual({
      scrollWidth: 375,
      fgCols: 1,
      prCols: 1,
      teCols: 1,
      heroStacked: true,
      ctaStacked: true,
    })

    await openAt(`${urls.neutral}/static`, 768)
    const at768 = await evaluate<{ fgCols: number; prCols: number; teCols: number; heroStacked: boolean; ctaSplit: boolean }>(`{
      fgCols: getComputedStyle(document.querySelector('#fg .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      prCols: getComputedStyle(document.querySelector('#pr .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      teCols: getComputedStyle(document.querySelector('#te .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      heroStacked: getComputedStyle(document.querySelector('#hero-h .weft-hero__inner')).gridTemplateColumns.split(' ').length === 1,
      ctaSplit: getComputedStyle(document.querySelector('#cta-solid .weft-cta__box')).gridTemplateColumns.split(' ').length === 2,
    }`)
    expect(at768).toEqual({ fgCols: 2, prCols: 2, teCols: 2, heroStacked: true, ctaSplit: true })

    await openAt(`${urls.neutral}/static`, 1440)
    const at1440 = await evaluate<{ fgCols: number; prCols: number; teCols: number; heroSplit: boolean; ctaSplit: boolean }>(`{
      fgCols: getComputedStyle(document.querySelector('#fg .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      prCols: getComputedStyle(document.querySelector('#pr .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      teCols: getComputedStyle(document.querySelector('#te .weft-grid__inner')).gridTemplateColumns.split(' ').length,
      heroSplit: getComputedStyle(document.querySelector('#hero-h .weft-hero__inner')).gridTemplateColumns.split(' ').length === 2,
      ctaSplit: getComputedStyle(document.querySelector('#cta-solid .weft-cta__box')).gridTemplateColumns.split(' ').length === 2,
    }`)
    expect(at1440).toEqual({ fgCols: 3, prCols: 3, teCols: 3, heroSplit: true, ctaSplit: true })
  })

  it('C1: two static blocks in 360 px boxes on a 1440 px page each adapt to 1 column and stack with no card overflow', async () => {
    await openAt(`${urls.neutral}/pair`, 1440)
    const pair = await evaluate<{
      heroStacked: boolean
      fgCols: number
      prCols: number
      ctaStacked: boolean
      overflowing: string[]
    }>(`(() => {
      const overflowing = []
      for (const box of [document.querySelector('#box-a'), document.querySelector('#box-b')]) {
        const br = box.getBoundingClientRect()
        for (const el of box.querySelectorAll('*')) {
          const r = el.getBoundingClientRect()
          if (r.right > br.right + 1) overflowing.push(el.tagName + '.' + el.className)
        }
      }
      return {
        heroStacked: getComputedStyle(document.querySelector('#pair-hero .weft-hero__inner')).gridTemplateColumns.split(' ').length === 1,
        fgCols: getComputedStyle(document.querySelector('#pair-fg .weft-grid__inner')).gridTemplateColumns.split(' ').length,
        prCols: getComputedStyle(document.querySelector('#pair-pr .weft-grid__inner')).gridTemplateColumns.split(' ').length,
        ctaStacked: getComputedStyle(document.querySelector('#pair-cta .weft-cta__box')).gridTemplateColumns.split(' ').length === 1,
        overflowing,
      }
    })()`)
    expect(pair).toEqual({
      heroStacked: true,
      fgCols: 1,
      prCols: 1,
      ctaStacked: true,
      overflowing: [],
    })
  })

  it('I4: an 87-character unbroken word wraps inside every block at 375 px without sideways scroll', async () => {
    await openAt(`${urls.neutral}/long`, 375)
    const scrollWidth = await evaluate<number>('document.documentElement.scrollWidth')
    expect(scrollWidth).toBeLessThanOrEqual(375)
  })

  it('I1–I3: Cta solid outline buttons, Cta soft/solid descriptions and Pricing pressed cycle badges meet WCAG AA (>= 4.5:1) in every theme and scheme', async () => {
    const contrastHelper = `
      function parseRgba(s) {
        const m = s.match(/rgba?\\((\\d+),\\s*(\\d+),\\s*(\\d+)(?:,\\s*([\\d.]+))?\\)/)
        if (!m) return [0, 0, 0, 0]
        return [Number(m[1]), Number(m[2]), Number(m[3]), m[4] === undefined ? 1 : Number(m[4])]
      }
      function blend(fg, bg) {
        const a = fg[3] + bg[3] * (1 - fg[3])
        if (a === 0) return [255, 255, 255, 1]
        return [
          (fg[0] * fg[3] + bg[0] * bg[3] * (1 - fg[3])) / a,
          (fg[1] * fg[3] + bg[1] * bg[3] * (1 - fg[3])) / a,
          (fg[2] * fg[3] + bg[2] * bg[3] * (1 - fg[3])) / a,
          a,
        ]
      }
      function effectiveBg(el) {
        const chain = []
        for (let cur = el; cur && cur.nodeType === 1; cur = cur.parentElement) {
          chain.unshift(parseRgba(getComputedStyle(cur).backgroundColor))
        }
        let acc = [255, 255, 255, 1]
        for (const c of chain) acc = blend(c, acc)
        return acc
      }
      function lum(rgb) {
        const [r, g, b] = rgb.slice(0, 3).map((v) => v / 255).map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
        return 0.2126 * r + 0.7152 * g + 0.0722 * b
      }
      function ratioOf(el) {
        const bg = effectiveBg(el)
        const fg = blend(parseRgba(getComputedStyle(el).color), bg)
        const [hi, lo] = [lum(fg), lum(bg)].sort((a, b) => b - a)
        return (hi + 0.05) / (lo + 0.05)
      }
    `
    for (const theme of ['neutral', 'lisa', 'weft'] as const) {
      for (const scheme of ['light', 'dark'] as const) {
        await openAt(`${urls[theme]}/static`, 1024, scheme)
        const staticRatios = await evaluate<{
          solidOutlineBtn: number
          solidDesc: number
          softDesc: number
        }>(`(() => {
          ${contrastHelper}
          return {
            solidOutlineBtn: ratioOf(document.querySelectorAll('#cta-solid .weft-button')[1]),
            solidDesc: ratioOf(document.querySelector('#cta-solid .weft-cta__description')),
            softDesc: ratioOf(document.querySelector('#cta-soft .weft-cta__description')),
          }
        })()`)
        expect(staticRatios.solidOutlineBtn, `${theme} ${scheme} Cta solid outline button`).toBeGreaterThanOrEqual(4.5)
        expect(staticRatios.solidDesc, `${theme} ${scheme} Cta solid description`).toBeGreaterThanOrEqual(4.5)
        expect(staticRatios.softDesc, `${theme} ${scheme} Cta soft description`).toBeGreaterThanOrEqual(4.5)

        await openAt(`${urls[theme]}/cycles`, 1024, scheme)
        const pressedBadgeRatio = await evaluate<number>(`(() => {
          ${contrastHelper}
          return ratioOf(document.querySelector('#pr-cycles .weft-pricing__cycle-badge'))
        })()`)
        expect(pressedBadgeRatio, `${theme} ${scheme} Pricing pressed cycle badge`).toBeGreaterThanOrEqual(4.5)
      }
    }
  })
})
