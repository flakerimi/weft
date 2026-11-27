<script setup vapor lang="ts">
// weft.base.al's landing page, after the approved design's second pass
// (docs/design/site-2026-11-21-v2/). The hero is the night band: the
// headline, the pitch, then its first action, the prompt to give a coding
// agent (agents/index-md.ts, sending it to /index.md), and below it the
// install command, each with its copy button (an island); and one app,
// Northline's device list, drawn twice by the real components, on a
// desktop and on a phone (inert pictures from the server).
// A saffron thread pulls across its foot, where the band ends, under the
// frames (CSS, the page's one automatic motion). Behind
// them, in the band's ground, the owner's picture of a weave, drawn by the
// CSS (docs/design/hero-2026-11-23/); it never moves. Then:
//   - screens you'd ship: five scenes on tiles, drawn by the server from the
//     real components, each caption linking the components it uses;
//   - the adaptive demo: VisitDay.vue's source, highlighted at build time;
//     the same file drawn by the server as the DatePicker's pointer and
//     touch forms, inert previews; and AdaptiveDemo, an island that starts
//     when it comes into view, to try the real one;
//   - the benchmark: bars and facts from the newest bench/results/*.json;
//   - for agents: the DatePicker's catalogue entry, the skills, /index.md, llms.txt;
//   - the close: the install command again, beside the owner's loom, a
//     picture loaded lazily from public/ (docs/design/hero-2026-11-23/).
// Everything but the islands is HTML from the build.
import { Calendar, Island, Link, WeftProvider, useHead } from '@weft/ui'
import type { Component } from 'vue'
import { highlightCode, useCollection } from 'virtual:weft/content'
import '@weft/vite/code.css'
import '../styles/fonts.css'
import '../styles/landing.css'
import catalogText from '../../catalog/components.json?raw'
import visitDaySource from '../components/landing/VisitDay.vue?raw'
import AdaptiveDemo from '../components/landing/AdaptiveDemo.vue'
import CopyCommand from '../components/CopyCommand.vue'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'
import VisitDay from '../components/landing/VisitDay.vue'
import NorthlineDesk from '../components/landing/NorthlineDesk.vue'
import NorthlinePhone from '../components/landing/NorthlinePhone.vue'
import Still from '../components/landing/Still.vue'
import Activity from '../components/landing/scenes/Activity.vue'
import Restore from '../components/landing/scenes/Restore.vue'
import Search from '../components/landing/scenes/Search.vue'
import Settings from '../components/landing/scenes/Settings.vue'
import Window from '../components/landing/scenes/Window.vue'
import { AGENT_PROMPT } from '../agents/index-md.ts'
import { slugOf, type Catalog } from '../gallery/model.ts'
import { benchFacts, benchRows, catalogExcerpt, formatBytes, formatCount, newestResults, type BenchResults } from '../landing/model.ts'

useHead({
  title: 'Weft — one UI system for Go and Vue apps',
  description:
    'Vue components that know whether they are under a finger or a mouse, on a landing page or in an app. Your site and your API ship as one Go binary.',
})

const install = 'curl -fsSL https://weft.base.al/install | sh'
const repoPublic = __WEFT_REPO_PUBLIC__

// The demo: the example's source as it is on disk, highlighted.
const sourceHTML = await highlightCode(visitDaySource, 'vue')
/** The day the example starts on, which the previews' calendars show open. */
const day = /ref<string \| null>\('(\d{4}-\d{2}-\d{2})'\)/.exec(visitDaySource)?.[1]
if (!day) throw new Error('landing: components/landing/VisitDay.vue starts on no ISO day; the previews draw its calendar open on it.')

// The benchmark: the newest results file, every number from it.
const bench = newestResults(import.meta.glob<BenchResults>('../../bench/results/*.json', { eager: true, import: 'default' }))
const rows = benchRows(bench.results)
const facts = benchFacts(bench.results)

// For agents: the DatePicker's catalogue entry, and the skills.
const components = (JSON.parse(catalogText) as Catalog).components
const datePicker = components.find((c) => c.name === 'DatePicker')
if (!datePicker) throw new Error('landing: catalog/components.json has no DatePicker; the agents section shows its entry.')
const entryPath = `/components/${slugOf(datePicker.name)}.json`
const excerptHTML = await highlightCode(catalogExcerpt(datePicker), 'json')
const skills = useCollection('skills')

// Screens you'd ship: each scene, the components its caption names (each a link to its page), and its tile's span.
const known = new Set(components.map((c) => c.name))
const scenes: { id: string; title: string; scene: Component; uses: string[] }[] = [
  { id: 'settings', title: 'Settings', scene: Settings, uses: ['Form', 'FormField', 'Input', 'Select', 'RadioGroup', 'Switch', 'Button'] },
  { id: 'search', title: 'Search anything', scene: Search, uses: ['CommandPalette', 'Kbd'] },
  { id: 'window', title: 'Maintenance window', scene: Window, uses: ['DateRangePicker', 'Calendar', 'Popover'] },
  { id: 'activity', title: 'Activity', scene: Activity, uses: ['Timeline', 'Avatar'] },
  { id: 'restore', title: 'Restore', scene: Restore, uses: ['FileUpload', 'Stack', 'Icon', 'Progress', 'Toast', 'Button'] },
]
for (const s of scenes) {
  for (const name of s.uses) if (!known.has(name)) throw new Error(`landing: the ${s.title} scene names ${name}, which catalog/components.json does not have; name a component it has.`)
}
</script>

<template>
  <div class="site landing" data-weft-surface="landing">
    <div class="landing-top" data-weft-scheme="dark"><SiteHeader path="/" :sticky="false" /></div>
    <main id="main" class="landing-main">
      <section class="landing-hero" aria-labelledby="landing-title">
        <div class="landing-hero__ground" data-weft-scheme="dark" />
        <div class="landing-wrap landing-hero__grid">
          <div class="landing-hero__copy" data-weft-scheme="dark">
            <h1 id="landing-title">One component tree, every screen.</h1>
            <p class="landing-lead">Vue components that know when they’re under a finger or a mouse, served with your Go API from one binary.</p>
            <p class="landing-label">Give this to your agent</p>
            <div class="landing-prompt">
              <p id="agent-prompt">{{ AGENT_PROMPT }}</p>
              <Island when="idle" :component="CopyCommand" :props="{ from: 'agent-prompt', name: 'Copy prompt' }" />
            </div>
            <p class="landing-label">or install it yourself</p>
            <div class="landing-install">
              <Island when="idle" :component="CopyCommand" :props="{ command: install }" />
            </div>
            <p class="landing-after">Already installed? <code>weft update</code> moves you to the newest release.</p>
          </div>
          <figure class="landing-frames">
            <Still class="hero-frame hero-frame--desk" scope="hd" input="pointer" width="wide"><NorthlineDesk /></Still>
            <Still class="hero-frame hero-frame--phone" scope="hp" input="touch" width="narrow"><NorthlinePhone /></Still>
            <figcaption class="landing-frames-note weft-visually-hidden">
              One app drawn twice by Weft’s components: Northline, a made-up fibre provider’s device list. On a desktop it is a
              table, a row’s menu open beside it; on a phone the same table is a list of cards and the same menu a bottom sheet.
            </figcaption>
          </figure>
        </div>
        <span class="landing-thread" aria-hidden="true" />
      </section>

      <section id="screens" class="landing-wrap landing-band" aria-labelledby="screens-title">
        <div class="landing-intro">
          <h2 id="screens-title">Screens you’d ship, made from the parts on this site.</h2>
          <p>Every tile is drawn by the components themselves, not a picture of them. Each name opens its page, with the props, the code and how it adapts.</p>
        </div>
        <div class="landing-bento">
          <figure v-for="(s, i) in scenes" :key="s.id" class="landing-tile" :class="`landing-tile--${s.id}`">
            <Still class="landing-tile__stage" :scope="`s${i}`"><component :is="s.scene" /></Still>
            <figcaption>
              <strong>{{ s.title }}</strong>
              <Link v-for="name in s.uses" :key="name" :to="`/components/${slugOf(name)}`">{{ name }}</Link>
            </figcaption>
          </figure>
        </div>
      </section>

      <section id="adapts" class="landing-wrap landing-band" aria-labelledby="adapts-title">
        <div class="landing-intro">
          <h2 id="adapts-title">Write a component once. It adapts.</h2>
          <p>
            You write <code>&lt;DatePicker&gt;</code>. On a trackpad it opens a calendar beside the field; under a thumb it
            rises as a sheet with targets big enough to hit. No mobile variant to remember.
          </p>
        </div>
        <div class="landing-adapt">
          <figure class="landing-code landing-code--source" data-weft-scheme="dark">
            <figcaption>site-src/components/landing/VisitDay.vue</figcaption>
            <div v-html="sourceHTML" />
          </figure>
          <figure class="landing-frame landing-frame--pointer">
            <figcaption class="landing-frame__title">Mouse and keyboard <span class="landing-tag">Preview</span></figcaption>
            <div class="landing-preview" inert>
              <WeftProvider input="pointer" surface="app">
                <VisitDay />
                <div class="landing-popover"><Calendar :model-value="day" locale="en-GB" /></div>
              </WeftProvider>
            </div>
            <p class="landing-caption">A popover calendar, typed dates welcome.</p>
          </figure>
          <figure class="landing-frame landing-frame--touch">
            <figcaption class="landing-frame__title">Touch <span class="landing-tag">Preview</span></figcaption>
            <div class="landing-preview" inert>
              <WeftProvider input="touch" surface="app">
                <div class="landing-behind"><VisitDay /></div>
                <div class="landing-sheet">
                  <span class="landing-sheet__grab" />
                  <p class="landing-sheet__title">Day</p>
                  <Calendar :model-value="day" locale="en-GB" />
                </div>
              </WeftProvider>
            </div>
            <p class="landing-caption">A bottom sheet, its dates 44&nbsp;px a side.</p>
          </figure>
          <div class="landing-demo">
            <Island when="visible" :component="AdaptiveDemo" />
          </div>
        </div>
      </section>

      <section id="speed" class="landing-wrap landing-band" aria-labelledby="speed-title">
        <div class="landing-intro">
          <h2 id="speed-title">One binary, nothing left to do per request.</h2>
          <p>
            Pages are rendered and compressed when you build. The Go binary keeps every byte in memory and only sends it.
            The benchmark’s landing page, each framework behind the server it ships with, requests per second:
          </p>
        </div>
        <div class="landing-bench" role="table" aria-label="Requests per second">
          <div v-for="row in rows" :key="row.id" class="landing-bench__row" role="row">
            <div class="landing-bench__who" role="rowheader">{{ row.label }}<small>{{ row.server }}</small></div>
            <div class="landing-bench__bar" :data-ours="row.ours ? 'true' : undefined" role="cell" aria-hidden="true">
              <i :style="{ inlineSize: `${(row.share * 100).toFixed(1)}%` }" />
            </div>
            <div class="landing-bench__n" role="cell">{{ formatCount(row.rps) }}</div>
          </div>
        </div>
        <p class="landing-bench-note">
          Requests per second, the median of {{ facts.runs }} runs on an {{ facts.machine }} ({{ facts.date }}). Weft's
          server runs on one core; the others are Node and are not held to one. The gap is the server, not the pages.
          <template v-if="facts.packed">
            Behind a plain Go server that also keeps brotli made at build in memory, the other sites' static builds
            answer {{ formatCount(facts.packed.min) }} to {{ formatCount(facts.packed.max) }}. Weft puts that server in
            your binary.
          </template>
          Astro's page with no framework
          island still ships less JavaScript than ours ({{ formatBytes(facts.plainJS) }} against
          {{ formatBytes(facts.weftJS) }}), and the benchmark says where,
          <template v-if="repoPublic">in <a href="https://github.com/flakerimi/weft/tree/main/bench">its README</a>.</template>
          <template v-else>in <code>bench/README.md</code>.</template>
        </p>
        <div class="landing-facts">
          <p><strong>No JavaScript unless you ask</strong>A page ships script only for the parts you mark interactive.</p>
          <p><strong>{{ facts.ttfb }} to first byte</strong>Pages come from memory, already compressed.</p>
          <p><strong>Deploy one file</strong>No Node in production. Copy it, run it.</p>
        </div>
      </section>

      <section id="agents" class="landing-wrap landing-band" aria-labelledby="agents-title">
        <div class="landing-intro">
          <h2 id="agents-title">Made to be built with an agent.</h2>
          <p>
            Agents do best with big pieces and one right way. Weft gives them both, and tells them when they’re wrong
            before you see it.
          </p>
        </div>
        <div class="landing-agents">
          <ul>
            <li>
              <strong>A catalogue they can read</strong>
              <span>
                Each of the {{ components.length }} components' purpose, props, slots, events and how it adapts, as JSON:
                all of them at <Link to="/catalog.json">/catalog.json</Link>, one at <Link :to="entryPath">{{ entryPath }}</Link>.
              </span>
            </li>
            <li>
              <strong>Mistakes fail the build</strong>
              <span>
                Frontmatter that breaks its schema, a link that leads nowhere or an island prop JSON cannot carry stops the
                build with one sentence that names the file.
              </span>
            </li>
            <li>
              <strong>Skills, ready to install</strong>
              <span>
                {{ skills.length }} Claude Code skills:
                <template v-for="(s, i) in skills" :key="s.slug"><Link :to="s.path">{{ s.title }}</Link>{{ i < skills.length - 2 ? ', ' : i === skills.length - 2 ? ' and ' : '. ' }}</template>
                <Link to="/skills">Install them</Link>, or point an agent at <Link to="/index.md">/index.md</Link> or <Link to="/llms.txt">/llms.txt</Link>.
              </span>
            </li>
          </ul>
          <figure class="landing-code" data-weft-scheme="dark">
            <figcaption>{{ entryPath }}, shortened</figcaption>
            <div v-html="excerptHTML" />
          </figure>
        </div>
      </section>

      <section id="start" class="landing-wrap landing-band" aria-labelledby="start-title">
        <div class="landing-end" data-weft-scheme="dark">
          <div>
            <h2 id="start-title">Start with one command.</h2>
            <p>It installs the <code>weft</code> command on macOS or Linux.</p>
            <div class="landing-install">
              <Island when="idle" :component="CopyCommand" :props="{ command: install }" />
            </div>
          </div>
          <picture>
            <source srcset="/loom-480.avif, /loom-960.avif 2x" type="image/avif" />
            <img src="/loom-480.webp" srcset="/loom-960.webp 2x" width="480" height="480" alt="" loading="lazy" decoding="async" style="aspect-ratio: 1" />
          </picture>
        </div>
      </section>
    </main>
    <SiteFooter />
  </div>
</template>
