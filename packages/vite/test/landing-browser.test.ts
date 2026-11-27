// Browser tests for the landing blocks and Grid on a built static page (no
// <Island>, using useWidth's pre-paint ResizeObserver), in narrow 360 px
// side-by-side boxes on a wide 1440 px page, inside shrink-to-fit parents
// (inline-block, positioned popover box, auto grid track, table cell), with
// long unbroken words and long Hero/Cta button labels at 375 px, and across
// all three themes and both schemes for WCAG AA contrast at rest, on hover,
// and on keyboard focus (C1, N1–N4, I1–I4).
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
const longBtnLabel = 'Start your free thirty-day trial of the Pro plan today'

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
      description="Static sections adapt before paint."
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

const SHRINK_PAGE = `<script setup vapor lang="ts">
import { Card, FeatureGrid, Grid, Pricing, Stack } from '@weft/ui'
</script>

<template>
  <main style="padding: 16px; display: flex; flex-direction: column; gap: 32px;">
    <div>
      <div id="case-inline-block" style="display: inline-block; border: 1px solid #ccc;">
        <Grid id="grid-inline-block"><Card>Revenue</Card><Card>Customers</Card><Card>Churn</Card></Grid>
      </div>
    </div>
    <div style="position: relative; height: 120px;">
      <div id="case-popover" style="position: absolute; top: 0; left: 0; min-inline-size: 8rem; border: 1px solid #ccc;">
        <Grid id="grid-popover"><Card>Revenue</Card><Card>Customers</Card><Card>Churn</Card></Grid>
      </div>
    </div>
    <div style="display: grid; grid-template-columns: auto 1fr; gap: 16px;">
      <div id="case-auto-track" style="border: 1px solid #ccc;">
        <Grid id="grid-auto-track"><Card>Revenue</Card><Card>Customers</Card><Card>Churn</Card></Grid>
      </div>
      <div>Sibling 1fr column</div>
    </div>
    <table>
      <tbody>
        <tr>
          <td id="case-table-cell" style="border: 1px solid #ccc;">
            <Grid id="grid-table-cell"><Card>Revenue</Card><Card>Customers</Card><Card>Churn</Card></Grid>
          </td>
          <td>Next table cell</td>
        </tr>
      </tbody>
    </table>
    <Stack id="case-row-stack" direction="row" :gap="4">
      <Grid id="grid-row-stack"><Card>One</Card><Card>Two</Card></Grid>
      <Card id="sibling-card">Sibling card</Card>
    </Stack>
    <div style="display: grid; justify-items: center;">
      <FeatureGrid id="fg-shrink" title="Centred FeatureGrid" :items="[{ title: 'Vapor', description: 'Direct DOM' }, { title: 'Go', description: 'One binary' }]" />
      <Pricing id="pr-shrink" title="Centred Pricing" :plans="[{ title: 'Starter', price: '$0' }, { title: 'Pro', price: '$29' }]" />
    </div>
    <div id="inert-box" inert style="width: 280px;">
      <Grid id="grid-inert"><Card>A</Card><Card>B</Card><Card>C</Card></Grid>
    </div>
  </main>
</template>
`

const LONG_WORD_PAGE = `<script setup vapor lang="ts">
import { Cta, FeatureGrid, Hero, Pricing, Testimonials } from '@weft/ui'
const w = '${longWord}'
const btn = '${longBtnLabel}'
</script>

<template>
  <main>
    <Hero :headline="w" :title="w" :description="w" :actions="[{ label: btn }, { label: btn }]" />
    <FeatureGrid :title="w" :items="[{ title: w, description: w }]" />
    <Pricing :title="w" :plans="[{ title: w, description: w, price: '$29', badge: 'Tag', features: [w], action: { label: w } }]" />
    <Testimonials :title="w" :items="[{ quote: w, author: w, role: w }]" />
    <Cta :title="w" :description="w" :actions="[{ label: btn }, { label: btn }]" />
  </main>
</template>
`

const CYCLES_ISLAND = `<script setup vapor lang="ts">
import { ref } from 'vue'
import { Card, Footer, Grid, Pricing, WeftProvider } from '@weft/ui'

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
  <div>
    <Footer id="cycles-footer"><template #leading>2026</template></Footer>
    <WeftProvider width="wide">
      <Grid id="cycles-provider-grid"><Card>A</Card><Card>B</Card></Grid>
    </WeftProvider>
    <Pricing id="pr-cycles" v-model:cycle="cycle" title="Cycles" :cycles="cycles" :plans="plans" />
    <div style="width: 300px;">
      <Grid id="cycles-boxed-grid"><Card>1</Card><Card>2</Card><Card>3</Card></Grid>
    </div>
  </div>
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

const SNIPPETS_PAGE = `<script setup vapor lang="ts">
import { reactive } from 'vue'
import { AppShell, Banner, Button, Card, Form, FormField, Grid, Input } from '@weft/ui'

const state = reactive({ email: '' })
</script>

<template>
  <AppShell persist="weft:app-shell:collapsed">
    <Banner id="news-1" close title="Release announcement" />
    <Grid id="snip-grid"><Card>One</Card><Card>Two</Card><Card>Three</Card></Grid>
    <Form id="snip-form" :state="state">
      <FormField name="email" label="Email">
        <Input v-model="state.email" />
      </FormField>
      <Button type="submit" label="Send" />
    </Form>
  </AppShell>
</template>
`

describe.skipIf(!browserPath)('landing blocks in a real browser (C1, N1–N4, I1–I4)', { timeout: 120_000 }, () => {
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
    writeFileSync(join(root, 'pages/shrink.vue'), SHRINK_PAGE)
    writeFileSync(join(root, 'pages/long.vue'), LONG_WORD_PAGE)
    writeFileSync(join(root, 'components/CyclesIsland.vue'), CYCLES_ISLAND)
    writeFileSync(join(root, 'pages/cycles.vue'), CYCLES_PAGE)
    writeFileSync(join(root, 'pages/snippets.vue'), SNIPPETS_PAGE)

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
    await send('Page.addScriptToEvaluateOnNewDocument', {
      source: `window.r = 'author-r'; window.s = 'author-s'; window.m = 1; window.__cls = 0; window.__pageErrors = []; window.addEventListener('error', (e) => window.__pageErrors.push(String(e.message))); new PerformanceObserver((l) => { for (const e of l.getEntries()) if (!e.hadRecentInput) window.__cls += e.value }).observe({ type: 'layout-shift', buffered: true });`,
    })
  })

  afterAll(async () => {
    ws?.close()
    if (browser && browser.exitCode === null) {
      browser.kill('SIGKILL')
      await new Promise((r) => browser!.once('close', r))
    }
    for (const s of servers) await new Promise<void>((done) => s.close(() => done()))
    if (dir) rmSync(dir, { recursive: true, force: true, maxRetries: 10, retryDelay: 50 })
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
    await new Promise((r) => setTimeout(r, 50))
  }

  it('C1 & N3: a built static page with no islands adapts to 1 column at 375 px, 2 at 768 px, and 3 / side-by-side at 1024 and 1440 px, with component CSS loaded and no horizontal overflow', async () => {
    const html = readFileSync(join(dir, 'dist-neutral/pages/static/index.html'), 'utf8')
    // No island runtime module script, and the pre-paint width observer is in <head>.
    expect(html).not.toContain('type="module"')
    expect(html).toContain('data-weft-prepaint')

    await openAt(`${urls.neutral}/static`, 375)
    const at375 = await evaluate<{
      gridDisplay: string
      scrollWidth: number
      fgWidthAttr: string
      fgCols: number
      prCols: number
      teCols: number
      heroStacked: boolean
      ctaStacked: boolean
      cls: number
      pageErrors: string[]
      globalsPreserved: boolean
    }>(`{
      gridDisplay: getComputedStyle(document.querySelector('#fg .weft-grid')).display,
      scrollWidth: document.documentElement.scrollWidth,
      fgWidthAttr: document.querySelector('#fg .weft-grid').getAttribute('data-width'),
      fgCols: getComputedStyle(document.querySelector('#fg .weft-grid')).gridTemplateColumns.split(' ').length,
      prCols: getComputedStyle(document.querySelector('#pr .weft-grid')).gridTemplateColumns.split(' ').length,
      teCols: getComputedStyle(document.querySelector('#te .weft-grid')).gridTemplateColumns.split(' ').length,
      heroStacked: getComputedStyle(document.querySelector('#hero-h .weft-hero__inner')).gridTemplateColumns.split(' ').length === 1,
      ctaStacked: getComputedStyle(document.querySelector('#cta-solid .weft-cta__box')).gridTemplateColumns.split(' ').length === 1,
      cls: window.__cls,
      pageErrors: window.__pageErrors,
      globalsPreserved: window.r === 'author-r' && window.s === 'author-s' && window.m === 1,
    }`)
    expect(at375).toEqual({
      gridDisplay: 'grid',
      scrollWidth: 375,
      fgWidthAttr: 'narrow',
      fgCols: 1,
      prCols: 1,
      teCols: 1,
      heroStacked: true,
      ctaStacked: true,
      cls: 0,
      pageErrors: [],
      globalsPreserved: true,
    })

    await openAt(`${urls.neutral}/static`, 768)
    const at768 = await evaluate<{ fgWidthAttr: string; fgCols: number; prCols: number; teCols: number; heroStacked: boolean; ctaSplit: boolean }>(`{
      fgWidthAttr: document.querySelector('#fg .weft-grid').getAttribute('data-width'),
      fgCols: getComputedStyle(document.querySelector('#fg .weft-grid')).gridTemplateColumns.split(' ').length,
      prCols: getComputedStyle(document.querySelector('#pr .weft-grid')).gridTemplateColumns.split(' ').length,
      teCols: getComputedStyle(document.querySelector('#te .weft-grid')).gridTemplateColumns.split(' ').length,
      heroStacked: getComputedStyle(document.querySelector('#hero-h .weft-hero__inner')).gridTemplateColumns.split(' ').length === 1,
      ctaSplit: getComputedStyle(document.querySelector('#cta-solid .weft-cta__box')).gridTemplateColumns.split(' ').length === 2,
    }`)
    expect(at768).toEqual({ fgWidthAttr: 'regular', fgCols: 2, prCols: 2, teCols: 2, heroStacked: true, ctaSplit: true })

    await openAt(`${urls.neutral}/static`, 1440)
    const at1440 = await evaluate<{ fgWidthAttr: string; fgCols: number; prCols: number; teCols: number; heroSplit: boolean; ctaSplit: boolean }>(`{
      fgWidthAttr: document.querySelector('#fg .weft-grid').getAttribute('data-width'),
      fgCols: getComputedStyle(document.querySelector('#fg .weft-grid')).gridTemplateColumns.split(' ').length,
      prCols: getComputedStyle(document.querySelector('#pr .weft-grid')).gridTemplateColumns.split(' ').length,
      teCols: getComputedStyle(document.querySelector('#te .weft-grid')).gridTemplateColumns.split(' ').length,
      heroSplit: getComputedStyle(document.querySelector('#hero-h .weft-hero__inner')).gridTemplateColumns.split(' ').length === 2,
      ctaSplit: getComputedStyle(document.querySelector('#cta-solid .weft-cta__box')).gridTemplateColumns.split(' ').length === 2,
    }`)
    expect(at1440).toEqual({ fgWidthAttr: 'wide', fgCols: 3, prCols: 3, teCols: 3, heroSplit: true, ctaSplit: true })
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
        fgCols: getComputedStyle(document.querySelector('#pair-fg .weft-grid')).gridTemplateColumns.split(' ').length,
        prCols: getComputedStyle(document.querySelector('#pair-pr .weft-grid')).gridTemplateColumns.split(' ').length,
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

  it('N1, N2 & N7: Grid and landing blocks inside shrink-to-fit parents (inline-block, popover, auto track, table cell, row Stack, centred grid) never collapse to 0 px, and elements inside [inert] are measured too', async () => {
    await openAt(`${urls.neutral}/shrink`, 1024)
    const sizes = await evaluate<{
      inlineBlock: number
      popover: number
      autoTrack: number
      tableCell: number
      rowStackGrid: number
      siblingCard: number
      fgShrink: number
      prShrink: number
      inertWidth: string | null
      inertCols: number
    }>(`{
      inlineBlock: Math.round(document.querySelector('#grid-inline-block').getBoundingClientRect().width),
      popover: Math.round(document.querySelector('#grid-popover').getBoundingClientRect().width),
      autoTrack: Math.round(document.querySelector('#grid-auto-track').getBoundingClientRect().width),
      tableCell: Math.round(document.querySelector('#grid-table-cell').getBoundingClientRect().width),
      rowStackGrid: Math.round(document.querySelector('#grid-row-stack').getBoundingClientRect().width),
      siblingCard: Math.round(document.querySelector('#sibling-card').getBoundingClientRect().width),
      fgShrink: Math.round(document.querySelector('#fg-shrink').getBoundingClientRect().width),
      prShrink: Math.round(document.querySelector('#pr-shrink').getBoundingClientRect().width),
      inertWidth: document.querySelector('#grid-inert').getAttribute('data-width'),
      inertCols: getComputedStyle(document.querySelector('#grid-inert')).gridTemplateColumns.split(' ').length,
    }`)
    expect(sizes.inlineBlock).toBeGreaterThan(80)
    expect(sizes.popover).toBeGreaterThan(80)
    expect(sizes.autoTrack).toBeGreaterThan(80)
    expect(sizes.tableCell).toBeGreaterThan(80)
    expect(sizes.rowStackGrid).toBeGreaterThan(60)
    expect(sizes.rowStackGrid).toBeLessThan(400)
    expect(sizes.siblingCard).toBeGreaterThan(60)
    expect(sizes.fgShrink).toBeGreaterThan(400)
    expect(sizes.prShrink).toBeGreaterThan(400)
    expect(sizes.inertWidth).toBe('narrow')
    expect(sizes.inertCols).toBe(1)
  })

  it('I4: an 87-character unbroken word and a 53-character Hero/Cta button label wrap inside every block at 375 px without sideways scroll', async () => {
    await openAt(`${urls.neutral}/long`, 375)
    const res = await evaluate<{ scrollWidth: number; heroBtnHeight: number }>(`{
      scrollWidth: document.documentElement.scrollWidth,
      heroBtnHeight: Math.round(document.querySelector('.weft-hero__actions .weft-button').getBoundingClientRect().height),
    }`)
    expect(res.scrollWidth).toBeLessThanOrEqual(375)
    expect(res.heroBtnHeight).toBeGreaterThanOrEqual(44)
  })

  it('I1–I3 & N4: Cta solid outline buttons (rest, hover and focus ring), Cta soft/solid descriptions and Pricing pressed cycle badges meet WCAG AA in every theme and scheme', async () => {
    const contrastHelper = `
      const canvas = document.createElement('canvas')
      canvas.width = canvas.height = 1
      const ctx = canvas.getContext('2d', { willReadFrequently: true })
      function parseCssColor(s) {
        if (!s || s === 'transparent' || s === 'rgba(0, 0, 0, 0)') return [0, 0, 0, 0]
        ctx.clearRect(0, 0, 1, 1)
        ctx.fillStyle = s
        ctx.fillRect(0, 0, 1, 1)
        const d = ctx.getImageData(0, 0, 1, 1).data
        return [d[0], d[1], d[2], d[3] / 255]
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
          chain.unshift(parseCssColor(getComputedStyle(cur).backgroundColor))
        }
        let acc = [255, 255, 255, 1]
        for (const c of chain) acc = blend(c, acc)
        return acc
      }
      function lum(rgb) {
        const [r, g, b] = rgb.slice(0, 3).map((v) => v / 255).map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
        return 0.2126 * r + 0.7152 * g + 0.0722 * b
      }
      function contrastRgb(a, b) {
        const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x)
        return (hi + 0.05) / (lo + 0.05)
      }
      function ratioOf(el) {
        const bg = effectiveBg(el)
        const fg = blend(parseCssColor(getComputedStyle(el).color), bg)
        return contrastRgb(fg, bg)
      }
    `
    for (const theme of ['neutral', 'lisa', 'weft'] as const) {
      for (const scheme of ['light', 'dark'] as const) {
        await openAt(`${urls[theme]}/static`, 1024, scheme)
        const btnBox = await evaluate<{ x: number; y: number }>(`(() => {
          const b = document.querySelectorAll('#cta-solid .weft-button')[1].getBoundingClientRect()
          return { x: Math.round(b.left + b.width / 2), y: Math.round(b.top + b.height / 2) }
        })()`)
        // Move mouse onto the outline button so :hover rules apply.
        await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x: btnBox.x, y: btnBox.y })
        // Keyboard focus on the first button in solid Cta so :focus-visible outline applies.
        await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9 })
        await send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9 })
        await evaluate(`(() => {
          const btn = document.querySelector('#cta-solid .weft-button')
          btn.focus()
          return true
        })()`)

        const staticRatios = await evaluate<{
          solidOutlineHoverBtn: number
          solidFocusRing: number
          solidDesc: number
          softDesc: number
        }>(`(() => {
          ${contrastHelper}
          const box = document.querySelector('#cta-solid .weft-cta__box')
          const btn0 = document.querySelectorAll('#cta-solid .weft-button')[0]
          const btn1 = document.querySelectorAll('#cta-solid .weft-button')[1]
          const boxBg = effectiveBg(box)
          const ringColor = parseCssColor(getComputedStyle(btn0).outlineColor)
          return {
            solidOutlineHoverBtn: ratioOf(btn1),
            solidFocusRing: contrastRgb(blend(ringColor, boxBg), boxBg),
            solidDesc: ratioOf(document.querySelector('#cta-solid .weft-cta__description')),
            softDesc: ratioOf(document.querySelector('#cta-soft .weft-cta__description')),
          }
        })()`)
        expect(staticRatios.solidOutlineHoverBtn, `${theme} ${scheme} Cta solid outline button on hover`).toBeGreaterThanOrEqual(4.5)
        expect(staticRatios.solidFocusRing, `${theme} ${scheme} Cta solid focus ring`).toBeGreaterThanOrEqual(3)
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

  it('N8 & Minor 1: a page with all four pre-paint snippets (AppShell, Banner, Grid, Form) runs every snippet without console or page errors, and hydrating an island at 375 px never briefly writes data-width="wide"', async () => {
    const snippetsHtml = readFileSync(join(dir, 'dist-neutral/pages/snippets/index.html'), 'utf8')
    const prepaintMatch = snippetsHtml.match(/<script data-weft-prepaint>([\s\S]*?)<\/script>/)
    expect(prepaintMatch).not.toBeNull()
    expect(prepaintMatch![1]!.split(';\n')).toHaveLength(4)

    await openAt(`${urls.neutral}/static`, 375)
    await evaluate(`(localStorage.setItem('weft:app-shell:collapsed', 'true'), localStorage.setItem('weft-banner:news-1', 'closed'), true)`)
    await openAt(`${urls.neutral}/snippets`, 375)
    const allFour = await evaluate<{
      pageErrors: string[]
      sidebar: string | null
      bannerDisplay: string
      gridWidth: string | null
      gridCols: number
      formSubmitPrevented: boolean
    }>(`(() => {
      const form = document.querySelector('#snip-form')
      const ev = new Event('submit', { bubbles: true, cancelable: true })
      form.dispatchEvent(ev)
      return {
        pageErrors: window.__pageErrors,
        sidebar: document.documentElement.dataset.weftSidebar ?? null,
        bannerDisplay: getComputedStyle(document.querySelector('.weft-banner[data-id="news-1"]')).display,
        gridWidth: document.querySelector('#snip-grid').getAttribute('data-width'),
        gridCols: getComputedStyle(document.querySelector('#snip-grid')).gridTemplateColumns.split(' ').length,
        formSubmitPrevented: ev.defaultPrevented,
      }
    })()`)
    expect(allFour).toEqual({
      pageErrors: [],
      sidebar: 'collapsed',
      bannerDisplay: 'none',
      gridWidth: 'narrow',
      gridCols: 1,
      formSubmitPrevented: true,
    })

    // Minor 1: record every distinct data-width attribute mutation on #pr-cycles .weft-grid and #cycles-boxed-grid (both preceded in the same island by a Footer with prepaint: false and a WeftProvider width="wide" Grid) — island hydration at 375 px and 1440 px must never briefly write "wide".
    await send('Page.addScriptToEvaluateOnNewDocument', {
      source: `window.__prHistory = []; window.__boxedHistory = []; new MutationObserver((recs) => { for (const r of recs) { if (r.type === 'attributes' && r.target instanceof Element) { if (r.target.matches('#pr-cycles .weft-grid')) { const w = r.target.getAttribute('data-width'); if (window.__prHistory[window.__prHistory.length - 1] !== w) window.__prHistory.push(w); } if (r.target.matches('#cycles-boxed-grid')) { const w = r.target.getAttribute('data-width'); if (window.__boxedHistory[window.__boxedHistory.length - 1] !== w) window.__boxedHistory.push(w); } } } }).observe(document, { subtree: true, attributes: true, attributeFilter: ['data-width'] });`,
    })
    await openAt(`${urls.neutral}/cycles`, 375)
    const hydration375 = await evaluate<{ prHistory: string[]; boxedHistory: string[]; finalWidth: string | null; hydrated: boolean }>(`{
      prHistory: window.__prHistory,
      boxedHistory: window.__boxedHistory,
      finalWidth: document.querySelector('#pr-cycles .weft-grid').getAttribute('data-width'),
      hydrated: document.querySelector('#pr-cycles .weft-grid').hasAttribute('data-weft-width'),
    }`)
    expect(hydration375).toEqual({
      prHistory: ['narrow'],
      boxedHistory: ['narrow'],
      finalWidth: 'narrow',
      hydrated: true,
    })
    await openAt(`${urls.neutral}/cycles`, 1440)
    const hydration1440 = await evaluate<{ boxedHistory: string[]; boxedFinal: string | null }>(`{
      boxedHistory: window.__boxedHistory,
      boxedFinal: document.querySelector('#cycles-boxed-grid').getAttribute('data-width'),
    }`)
    expect(hydration1440).toEqual({
      boxedHistory: ['narrow'],
      boxedFinal: 'narrow',
    })
  })
})
