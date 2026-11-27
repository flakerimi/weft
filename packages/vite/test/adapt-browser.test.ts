// Browser tests for the width-adaptive components that used to adapt only
// once hydrated: on a built static page with no <Island>, Stepper and
// Timeline (horizontal) and Tabs (vertical) take their layout from the
// pre-paint `data-width` in CSS, and Pagination and Footer take theirs from a
// viewport `@media` fallback before mounting. Checked at 375 and 1440 px, in
// 360 px boxes on a 1440 px page and in shrink-to-fit parents (inline-block,
// an absolutely positioned box, an `auto` grid track, a table cell): the
// right layout, no sideways scroll, no collapse, no layout shift. Inside an
// island, what is painted never flips while hydrating, and the accessibility
// tree of a static narrow page is the one the hydrated component gives.
// Table switches its markup in JavaScript, so on a static page it stays a
// table that scrolls inside its own box; that is checked too.
import { spawn, type ChildProcess } from 'node:child_process'
import { existsSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import type { AddressInfo } from 'node:net'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite } from '../src/build.ts'

function findBrowser(): string | null {
  if (process.env.WEFT_BROWSER && existsSync(process.env.WEFT_BROWSER)) return process.env.WEFT_BROWSER
  for (const p of ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/Applications/Chromium.app/Contents/MacOS/Chromium']) {
    if (existsSync(p)) return p
  }
  return null
}
const browserPath = findBrowser()

const DATA = `
const steps = [
  { title: 'Account', description: 'Your details' },
  { title: 'Address', description: 'Where to ship' },
  { title: 'Shipping', description: 'How fast' },
  { title: 'Payment', description: 'Card or invoice' },
]
const events = [
  { title: 'Ordered', date: 'Mon 9:00', description: 'We have your order.' },
  { title: 'Packed', date: 'Mon 14:00', description: 'In a box.' },
  { title: 'Shipped', date: 'Tue 8:00', description: 'On its way.' },
  { title: 'Delivered', date: 'Wed 11:00', description: 'At the door.' },
]
const tabs = [
  { label: 'Overview', content: 'The overview panel.' },
  { label: 'Settings', content: 'The settings panel.' },
  { label: 'Billing', content: 'The billing panel.' },
]
const columns = [
  { key: 'name', label: 'Name', rowHeader: true },
  { key: 'email', label: 'Email' },
  { key: 'role', label: 'Role' },
  { key: 'team', label: 'Team' },
  { key: 'since', label: 'Since' },
]
const rows = [
  { id: 1, name: 'Avery Quinn', email: 'avery.quinn@example.com', role: 'Owner', team: 'Platform', since: '2021-04-01' },
  { id: 2, name: 'Morgan Patel', email: 'morgan.patel@example.com', role: 'Editor', team: 'Payments', since: '2022-09-12' },
  { id: 3, name: 'Rowan Lindqvist', email: 'rowan.lindqvist@example.com', role: 'Viewer', team: 'Growth', since: '2024-01-30' },
]
const href = (n) => '/list/' + n
`

const STATIC_PAGE = `<script setup vapor lang="ts">
import { Footer, Pagination, Stepper, Table, Tabs, Timeline } from '@weft/ui'
${DATA}
</script>

<template>
  <main style="padding: 16px; display: flex; flex-direction: column; gap: 32px;">
    <Stepper id="st" :items="steps" :default-value="1" />
    <Timeline id="tl" orientation="horizontal" :items="events" :default-value="2" />
    <Tabs id="tb" orientation="vertical" :items="tabs" aria-label="Sections" />
    <Pagination id="pg" :total="200" :default-page="10" show-edges :to="href" />
    <Pagination id="pg5" :total="200" :default-page="5" show-edges :to="href" />
    <Pagination id="pgw" :total="200" :default-page="10" :to="href" />
    <Table id="tbl" caption="People" :columns="columns" :rows="rows" />
  </main>
  <Footer id="ft">
    <template #leading>© 2026 Example Co</template>
    <a href="/about">About</a><a href="/blog">Blog</a><a href="/jobs">Jobs</a>
    <template #trailing>Follow us</template>
  </Footer>
</template>
`

const BOX_PAGE = `<script setup vapor lang="ts">
import { Pagination, Stepper, Tabs, Timeline } from '@weft/ui'
${DATA}
</script>

<template>
  <div style="display: flex; gap: 24px; align-items: flex-start; padding: 16px;">
    <div id="box-a" style="width: 360px; flex: none;">
      <Stepper id="bst" :items="steps" />
      <Timeline id="btl" orientation="horizontal" :items="events" />
      <Tabs id="btb" orientation="vertical" :items="tabs" aria-label="Boxed" />
      <Pagination id="bpg" :total="200" :default-page="10" show-edges :to="href" />
    </div>
    <div id="box-b" style="width: 900px; flex: none;">
      <Stepper id="rst" :items="steps" />
      <Timeline id="rtl" orientation="horizontal" :items="events" />
      <Tabs id="rtb" orientation="vertical" :items="tabs" aria-label="Regular" />
    </div>
  </div>
</template>
`

const each = (prefix: string) => `
        <Stepper id="${prefix}-st" :items="steps" />
        <Timeline id="${prefix}-tl" orientation="horizontal" :items="events" />
        <Tabs id="${prefix}-tb" orientation="vertical" :items="tabs" aria-label="${prefix}" />
        <Pagination id="${prefix}-pg" :total="200" :default-page="10" show-edges :to="href" />`

const SHRINK_PAGE = `<script setup vapor lang="ts">
import { Pagination, Stepper, Tabs, Timeline } from '@weft/ui'
${DATA}
</script>

<template>
  <main style="padding: 16px; display: flex; flex-direction: column; gap: 32px;">
    <div><div id="c-ib" style="display: inline-block; border: 1px solid #ccc;">${each('ib')}
    </div></div>
    <div style="position: relative; height: 700px;"><div id="c-abs" style="position: absolute; top: 0; left: 0; min-inline-size: 8rem; border: 1px solid #ccc;">${each('abs')}
    </div></div>
    <div style="display: grid; grid-template-columns: auto 1fr; gap: 16px;"><div id="c-auto" style="border: 1px solid #ccc;">${each('auto')}
    </div><div>Sibling</div></div>
    <table><tbody><tr><td id="c-td" style="border: 1px solid #ccc;">${each('td')}
    </td><td>Next cell</td></tr></tbody></table>
  </main>
</template>
`

const ISLAND = `<script setup vapor lang="ts">
import { Footer, Pagination, Stepper, Table, Tabs, Timeline } from '@weft/ui'
${DATA}
</script>

<template>
  <div style="display: flex; flex-direction: column; gap: 32px;">
    <Stepper id="st" :items="steps" :default-value="1" />
    <Timeline id="tl" orientation="horizontal" :items="events" :default-value="2" />
    <Tabs id="tb" orientation="vertical" :items="tabs" aria-label="Sections" />
    <Pagination id="pg5" :total="200" :default-page="5" show-edges :to="href" />
    <Table id="tbl" caption="People" :columns="columns" :rows="rows" />
  </div>
  <Footer id="ft">
    <template #leading>© 2026 Example Co</template>
    <a href="/about">About</a><a href="/blog">Blog</a><a href="/jobs">Jobs</a>
    <template #trailing>Follow us</template>
  </Footer>
</template>
`

const ISLAND_PAGE = `<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Adaptive from '../components/Adaptive.vue'
</script>

<template>
  <main style="padding: 16px;">
    <Island when="load" :component="Adaptive" />
  </main>
</template>
`

/** What is painted, per component, sampled on every animation frame (before each paint). */
const SAMPLER = `
window.__cls = 0; window.__pageErrors = []; window.__painted = {};
addEventListener('error', (e) => window.__pageErrors.push(String(e.message)));
new PerformanceObserver((l) => { for (const e of l.getEntries()) if (!e.hadRecentInput) window.__cls += e.value }).observe({ type: 'layout-shift', buffered: true });
const fd = (el) => el ? getComputedStyle(el).flexDirection : '-';
const shown = (el) => el && getComputedStyle(el).display !== 'none';
const sigs = {
  st: (el) => fd(el.querySelector('.weft-stepper__list')),
  tl: (el) => fd(el),
  tb: (el) => fd(el) + '/' + fd(el.querySelector('.weft-tabs__list')),
  pg5: (el) => [...el.querySelectorAll('li')].filter(shown).map((li) => li.textContent.trim() || li.querySelector('[aria-label]')?.getAttribute('aria-label')).join(' '),
  tbl: (el) => shown(el.querySelector('table')) ? 'table' : shown(el.querySelector('.weft-table__cards')) ? 'cards' : '-',
  ft: (el) => fd(el.querySelector('.weft-footer__row')),
};
const tick = () => {
  for (const id in sigs) {
    const el = document.getElementById(id);
    if (!el) continue;
    const s = sigs[id](el);
    const h = (window.__painted[id] ||= []);
    if (h[h.length - 1] !== s) h.push(s);
  }
  requestAnimationFrame(tick);
};
requestAnimationFrame(tick);
window.__widths = {};
new MutationObserver((recs) => {
  for (const r of recs) {
    const id = r.target.id;
    if (!id) continue;
    const w = r.target.getAttribute('data-width');
    if (r.oldValue !== w) (window.__widths[id] ||= []).push(r.oldValue + '>' + w);
  }
}).observe(document, { subtree: true, attributes: true, attributeOldValue: true, attributeFilter: ['data-width'] });
`

describe.skipIf(!browserPath)('width-adaptive components on a static page, in a real browser', { timeout: 180_000 }, () => {
  let dir: string
  let server: Server
  let url = ''
  let outDir = ''
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
    // realpath: on macOS tmpdir() is a symlink, and the build must see the real root.
    dir = realpathSync(mkdtempSync(join(tmpdir(), 'weft-adapt-browser-')))
    const root = join(dir, 'site')
    mkdirSync(join(root, 'pages'), { recursive: true })
    mkdirSync(join(root, 'components'), { recursive: true })
    symlinkSync(fileURLToPath(new URL('../node_modules', import.meta.url)), join(root, 'node_modules'))
    writeFileSync(join(root, 'pages/static.vue'), STATIC_PAGE)
    writeFileSync(join(root, 'pages/box.vue'), BOX_PAGE)
    writeFileSync(join(root, 'pages/shrink.vue'), SHRINK_PAGE)
    writeFileSync(join(root, 'components/Adaptive.vue'), ISLAND)
    writeFileSync(join(root, 'pages/island.vue'), ISLAND_PAGE)
    outDir = join(dir, 'dist')
    await buildSite({ root, outDir })
    server = createServer((req, res) => {
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
    await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
    url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`

    const profile = join(dir, 'profile')
    mkdirSync(profile, { recursive: true })
    browser = spawn(browserPath!, ['--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`, '--no-first-run', '--no-default-browser-check', 'about:blank'], { stdio: 'ignore' })
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
    await send('DOM.enable')
    await send('Accessibility.enable')
    await send('Page.addScriptToEvaluateOnNewDocument', { source: SAMPLER })
  }, 180_000)

  afterAll(async () => {
    ws?.close()
    if (browser && browser.exitCode === null) {
      browser.kill('SIGKILL')
      await new Promise((r) => browser!.once('close', r))
    }
    if (server) await new Promise<void>((done) => server.close(() => done()))
    if (dir) rmSync(dir, { recursive: true, force: true, maxRetries: 10, retryDelay: 50 })
  })

  async function openAt(path: string, width: number, settle = 150) {
    await send('Emulation.setDeviceMetricsOverride', { width, height: 900, deviceScaleFactor: 1, mobile: false })
    await send('Page.navigate', { url: `${url}${path}` })
    for (let i = 0; i < 200; i++) {
      if ((await evaluate<string>('document.readyState')) === 'complete') break
      await new Promise((r) => setTimeout(r, 25))
    }
    await new Promise((r) => setTimeout(r, settle))
  }

  /** The accessibility tree under `selector`: role, name and orientation of every node that is not ignored. */
  async function axTree(selector: string): Promise<string[]> {
    const { root } = (await send('DOM.getDocument', { depth: 0 })) as { root: { nodeId: number } }
    const { nodeId } = (await send('DOM.querySelector', { nodeId: root.nodeId, selector })) as { nodeId: number }
    const { node } = (await send('DOM.describeNode', { nodeId })) as { node: { backendNodeId: number } }
    const { nodes } = (await send('Accessibility.getFullAXTree')) as {
      nodes: { nodeId: string; ignored: boolean; role?: { value: string }; name?: { value: string }; childIds?: string[]; backendDOMNodeId?: number; properties?: { name: string; value: { value: unknown } }[] }[]
    }
    const byId = new Map(nodes.map((n) => [n.nodeId, n]))
    const start = nodes.find((n) => n.backendDOMNodeId === node.backendNodeId)
    const out: string[] = []
    const walk = (id: string, depth: number) => {
      const n = byId.get(id)
      if (!n) return
      const role = n.role?.value ?? ''
      const keep = !n.ignored && !['generic', 'none', 'InlineTextBox', 'LineBreak'].includes(role)
      if (keep) {
        const orientation = n.properties?.find((p) => p.name === 'orientation')?.value.value
        out.push(`${'  '.repeat(depth)}${role}${n.name?.value ? ` "${n.name.value}"` : ''}${orientation ? ` [${orientation}]` : ''}`)
      }
      for (const c of n.childIds ?? []) walk(c, keep ? depth + 1 : depth)
    }
    if (start) walk(start.nodeId, 0)
    return out
  }

  const LAYOUT = `(() => {
    const fd = (el) => getComputedStyle(el).flexDirection
    const shown = (el) => getComputedStyle(el).display !== 'none'
    const entries = (id) => [...document.querySelectorAll('#' + id + ' li')].filter(shown).map((li) => li.textContent.trim() || li.querySelector('[aria-label]').getAttribute('aria-label'))
    return {
      scrollWidth: document.documentElement.scrollWidth,
      stepper: fd(document.querySelector('#st .weft-stepper__list')),
      stepperWidth: document.querySelector('#st').getAttribute('data-width'),
      timeline: fd(document.querySelector('#tl')),
      tabs: fd(document.querySelector('#tb')) + '/' + fd(document.querySelector('#tb .weft-tabs__list')),
      pg: entries('pg').join(' '),
      pg5: entries('pg5').join(' '),
      pgw: entries('pgw').join(' '),
      table: shown(document.querySelector('#tbl table')) ? 'table' : 'cards',
      tableInside: document.querySelector('#tbl').getBoundingClientRect().right <= document.documentElement.clientWidth,
      footer: fd(document.querySelector('#ft .weft-footer__row')),
      cls: window.__cls,
      pageErrors: window.__pageErrors,
    }
  })()`

  it('a static page with no island loads no module, and lays out narrow at 375 px and wide at 1440 px with no sideways scroll and no layout shift', async () => {
    const html = readFileSync(join(outDir, 'pages/static/index.html'), 'utf8')
    expect(html).not.toContain('type="module"')
    expect(html).toContain('data-weft-prepaint')

    await openAt('/static', 375)
    expect(await evaluate(LAYOUT)).toEqual({
      scrollWidth: 375,
      stepper: 'column',
      stepperWidth: 'narrow',
      timeline: 'column',
      tabs: 'column/row',
      pg: 'Previous page 1 … 9 10 11 … 20 Next page',
      pg5: 'Previous page 1 … 4 5 6 … 20 Next page',
      pgw: 'Previous page 9 10 11 Next page',
      table: 'table',
      tableInside: true,
      footer: 'column',
      cls: 0,
      pageErrors: [],
    })

    await openAt('/static', 1440)
    expect(await evaluate(LAYOUT)).toEqual({
      scrollWidth: 1440,
      stepper: 'row',
      stepperWidth: 'wide',
      timeline: 'row',
      tabs: 'row/column',
      pg: 'First page Previous page 1 … 8 9 10 11 12 … 20 Next page Last page',
      pg5: 'First page Previous page 1 2 3 4 5 6 7 … 20 Next page Last page',
      pgw: 'First page Previous page 8 9 10 11 12 Next page Last page',
      table: 'table',
      tableInside: true,
      footer: 'row',
      cls: 0,
      pageErrors: [],
    })
  })

  it('resizing a static page re-lays it out without hydration', async () => {
    await openAt('/static', 1440)
    await send('Emulation.setDeviceMetricsOverride', { width: 375, height: 900, deviceScaleFactor: 1, mobile: false })
    await new Promise((r) => setTimeout(r, 200))
    const after = await evaluate<{ stepper: string; timeline: string; tabs: string; pg5: string; scrollWidth: number }>(LAYOUT)
    expect(after).toMatchObject({ stepper: 'column', timeline: 'column', tabs: 'column/row', pg5: 'Previous page 1 … 4 5 6 … 20 Next page', scrollWidth: 375 })
  })

  it('the pre-paint script and a mounted component measure the same box: a padded Footer is measured once, and no width from 360 to 1000 px logs a ResizeObserver loop error', async () => {
    for (let width = 360; width <= 1000; width += 10) {
      await openAt('/static', width, 150)
      expect(await evaluate<string[]>('window.__pageErrors'), `at ${width}`).toEqual([])
    }
    for (let width = 630; width <= 680; width += 10) {
      await openAt('/static', width, 300)
      const res = await evaluate<{ footer: string[]; pageErrors: string[]; padding: string }>(`({
        footer: window.__widths.ft ?? [],
        pageErrors: window.__pageErrors,
        padding: getComputedStyle(document.getElementById('ft')).paddingInlineStart,
      })`)
      expect(res.padding, 'the Footer is padded').not.toBe('0px')
      // The border box, everywhere: 630 px is narrow and 640 px and up regular, written once.
      expect(res.footer, `footer at ${width}`).toEqual(width < 640 ? ['wide>narrow'] : ['wide>regular'])
      expect(res.pageErrors, `at ${width}`).toEqual([])
    }
  })

  it('in a 360 px box on a 1440 px page they lay out narrow, in a 900 px box regular, and nothing leaves its box', async () => {
    await openAt('/box', 1440)
    const res = await evaluate<Record<string, unknown>>(`(() => {
      const fd = (s) => getComputedStyle(document.querySelector(s)).flexDirection
      const overflowing = []
      for (const box of document.querySelectorAll('#box-a, #box-b')) {
        const br = box.getBoundingClientRect()
        for (const el of box.querySelectorAll('*')) if (el.getBoundingClientRect().right > br.right + 1) overflowing.push(box.id + ' ' + el.className)
      }
      return {
        bst: fd('#bst .weft-stepper__list'), btl: fd('#btl'), btb: fd('#btb') + '/' + fd('#btb .weft-tabs__list'),
        rst: fd('#rst .weft-stepper__list'), rtl: fd('#rtl'), rtb: fd('#rtb') + '/' + fd('#rtb .weft-tabs__list'),
        widths: ['bst', 'btl', 'btb', 'rst', 'rtl', 'rtb'].map((id) => document.getElementById(id).getAttribute('data-width')).join(' '),
        overflowing, scrollWidth: document.documentElement.scrollWidth, cls: window.__cls,
      }
    })()`)
    expect(res).toEqual({
      bst: 'column', btl: 'column', btb: 'column/row',
      rst: 'row', rtl: 'row', rtb: 'row/column',
      widths: 'narrow narrow narrow regular regular regular',
      overflowing: [], scrollWidth: 1440, cls: 0,
    })
  })

  it('in shrink-to-fit parents (inline-block, absolute box, auto track, table cell) they never collapse, lay out by their own data-width, and stay inside', async () => {
    for (const width of [375, 1440]) {
      await openAt('/shrink', width)
      const res = await evaluate<{ cases: { id: string; width: number; dataWidth: string; layout: string; want: string }[]; overflowing: string[]; scrollWidth: number; cls: number; pageErrors: string[] }>(`(() => {
        const fd = (el) => getComputedStyle(el).flexDirection
        const cases = []
        const overflowing = []
        for (const prefix of ['ib', 'abs', 'auto', 'td']) {
          for (const kind of ['st', 'tl', 'tb']) {
            const el = document.getElementById(prefix + '-' + kind)
            const dataWidth = el.getAttribute('data-width')
            const layout = kind === 'st' ? fd(el.querySelector('.weft-stepper__list')) : kind === 'tl' ? fd(el) : fd(el)
            const narrow = dataWidth === 'narrow'
            const want = kind === 'tb' ? (narrow ? 'column' : 'row') : narrow ? 'column' : 'row'
            cases.push({ id: el.id, width: Math.round(el.getBoundingClientRect().width), dataWidth, layout, want })
          }
          // Pagination follows the viewport: it only has to keep its size and stay inside.
          const pg = document.getElementById(prefix + '-pg')
          cases.push({ id: pg.id, width: Math.round(pg.getBoundingClientRect().width), dataWidth: '-', layout: '-', want: '-' })
          const box = document.getElementById('c-' + prefix)
          const br = box.getBoundingClientRect()
          for (const el of box.querySelectorAll('*')) if (el.getBoundingClientRect().right > br.right + 1) overflowing.push(prefix + ' ' + el.className)
        }
        return { cases, overflowing, scrollWidth: document.documentElement.scrollWidth, cls: window.__cls, pageErrors: window.__pageErrors }
      })()`)
      for (const c of res.cases) {
        expect(c.width, `${c.id} at ${width}`).toBeGreaterThan(80)
        expect(c.layout, `${c.id} at ${width} (${c.dataWidth}, ${c.width} px)`).toBe(c.want)
      }
      expect(res.overflowing, `at ${width}`).toEqual([])
      expect(res.scrollWidth, `at ${width}`).toBeLessThanOrEqual(width)
      expect(res.cls, `at ${width}`).toBe(0)
      expect(res.pageErrors, `at ${width}`).toEqual([])
    }
  })

  it('inside an island, nothing painted flips while hydrating, at 375 and 1440 px (Table aside, which turns into cards once hydrated)', async () => {
    for (const [width, want] of [
      [375, { st: ['column'], tl: ['column'], tb: ['column/row'], pg5: ['Previous page 1 … 4 5 6 … 20 Next page'], tbl: ['table', 'cards'], ft: ['column'] }],
      [1440, { st: ['row'], tl: ['row'], tb: ['row/column'], pg5: ['First page Previous page 1 2 3 4 5 6 7 … 20 Next page Last page'], tbl: ['table'], ft: ['row'] }],
    ] as const) {
      await openAt('/island', width, 400)
      const res = await evaluate<{ painted: Record<string, string[]>; widths: Record<string, (string | null)[]>; hydrated: boolean; cls: number; pageErrors: string[] }>(`({
        painted: window.__painted,
        widths: window.__widths,
        hydrated: document.querySelector('#st').hasAttribute('data-weft-width') && document.querySelector('#ft').hasAttribute('data-weft-width'),
        cls: window.__cls,
        pageErrors: window.__pageErrors,
      })`)
      expect(res.hydrated, `hydrated at ${width}`).toBe(true)
      expect(res.painted, `painted at ${width}`).toEqual(want)
      // data-width changes once, by the pre-paint script, and never back to "wide" while hydrating (Footer included).
      const w = width === 375 ? ['wide>narrow'] : []
      expect({ st: res.widths.st ?? [], tl: res.widths.tl ?? [], tb: res.widths.tb ?? [], ft: res.widths.ft ?? [] }, `data-width at ${width}`).toEqual({ st: w, tl: w, tb: w, ft: w })
      expect(res.cls, `cls at ${width}`).toBeLessThan(width === 375 ? 0.1 : 0.001)
      expect(res.pageErrors).toEqual([])
    }
  })

  it('Pagination, hydrating in an island, switches where its media query does: a classic scrollbar (here a 17 px narrower root) does not flip it at 645 px', async () => {
    const { identifier } = (await send('Page.addScriptToEvaluateOnNewDocument', {
      source: `new MutationObserver((_, mo) => { if (document.documentElement) { document.documentElement.style.cssText += ';width: calc(100% - 17px) !important'; mo.disconnect() } }).observe(document, { childList: true })`,
    })) as { identifier: string }
    try {
      for (const width of [630, 645, 660]) {
        await openAt('/island', width, 400)
        const res = await evaluate<{ root: number; painted: string[]; hydrated: boolean }>(`({
          root: Math.round(document.documentElement.getBoundingClientRect().width),
          painted: window.__painted.pg5,
          hydrated: document.querySelector('#st').hasAttribute('data-weft-width'),
        })`)
        expect(res.root, 'the root is narrower than the viewport').toBe(width - 17)
        expect(res.hydrated).toBe(true)
        expect(res.painted, `pg5 at ${width}`).toEqual([width < 640 ? 'Previous page 1 … 4 5 6 … 20 Next page' : 'First page Previous page 1 2 3 4 5 6 7 … 20 Next page Last page'])
      }
    } finally {
      await send('Page.removeScriptToEvaluateOnNewDocument', { identifier })
    }
  })

  it('the accessibility tree of a static page is the hydrated one: the same at 375 and 1440 px, and what the island gives at 375 px', async () => {
    await openAt('/static', 375)
    const narrow = { st: await axTree('#st'), tl: await axTree('#tl'), tb: await axTree('#tb'), pg5: await axTree('#pg5'), tbl: await axTree('#tbl') }
    await openAt('/static', 1440)
    const wide = { st: await axTree('#st'), tl: await axTree('#tl'), tb: await axTree('#tb'), pg5: await axTree('#pg5') }
    await openAt('/island', 375, 400)
    const island = { st: await axTree('#st'), tl: await axTree('#tl'), tb: await axTree('#tb'), pg5: await axTree('#pg5'), tbl: await axTree('#tbl') }
    writeFileSync(join(dir, 'ax.json'), JSON.stringify({ narrow, wide, island }, null, 2))
    if (process.env.WEFT_AX_OUT) writeFileSync(process.env.WEFT_AX_OUT, JSON.stringify({ narrow, wide, island }, null, 2))

    expect(narrow.st).toEqual(wide.st)
    expect(narrow.tl).toEqual(wide.tl)
    expect(narrow.tb).toEqual(wide.tb)
    expect(narrow.st).toEqual(island.st)
    expect(narrow.tl).toEqual(island.tl)
    expect(narrow.pg5).toEqual(island.pg5)
    // Tabs: the same tabs and panel; only the tab list's orientation, which the
    // arrow keys follow once hydrated, is the prop's until then.
    expect(narrow.tb.map((l) => l.replace(/ \[(horizontal|vertical)\]/, ''))).toEqual(island.tb.map((l) => l.replace(/ \[(horizontal|vertical)\]/, '')))
    expect(narrow.tb.some((l) => /tablist.*\[vertical\]/.test(l))).toBe(true)
    expect(island.tb.some((l) => /tablist.*\[horizontal\]/.test(l))).toBe(true)
    // Table: a table on the static page; a named list of cards, each a list of
    // label-value terms, once hydrated narrow.
    expect(narrow.tbl.some((l) => /^\s*table "People"/.test(l))).toBe(true)
    expect(island.tbl.some((l) => /^\s*list "People"/.test(l))).toBe(true)
    expect(island.tbl.filter((l) => /^\s*listitem/.test(l))).toHaveLength(3)
  })
})
