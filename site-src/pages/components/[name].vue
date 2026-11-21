<script lang="ts">
import { components } from '../../gallery/catalog.ts'

export const getPaths = async () => components.map((c) => ({ name: c.slug }))
</script>

<script setup vapor lang="ts">
// A component's page, /components/<slug>, all from its catalogue entry and
// its example files: what it is for; the preview stage (an island, its
// first example under a toolbar); that example's source; what adapts; its
// props, slots and events; every other example, drawn by the server and
// hydrated only when it needs JavaScript; and the raw entry for agents,
// written to /components/<slug>.json and shown from there.
import { Breadcrumb, Island, Link, Page, WeftProvider, useHead, type BreadcrumbItem } from '@weft/ui'
import { useFile } from 'virtual:weft/files'
import '@weft/vite/code.css'
import '../../styles/fonts.css'
import '../../styles/shell.css'
import '../../styles/gallery.css'
import { componentBySlug } from '../../gallery/catalog.ts'
import { groupLabel, inlineHTML, neighbours } from '../../gallery/model.ts'
import CopyCommand from '../../components/CopyCommand.vue'
import GalleryNav from '../../components/gallery/GalleryNav.vue'
import SiteFooter from '../../components/SiteFooter.vue'
import SiteHeader from '../../components/SiteHeader.vue'

const props = defineProps<{ name: string }>()
const c = componentBySlug(props.name)
const [first, ...more] = c.examples as [(typeof c.examples)[number], ...typeof c.examples]
const { prev, next } = neighbours(components, c.name)

useHead({ title: `${c.name} — Weft components`, description: c.purpose.replace(/`/g, '') })
useFile(`/components/${c.slug}.json`, c.json)

const crumbs: BreadcrumbItem[] = [{ label: 'Components', to: '/components' }, { label: groupLabel(c.group) }, { label: c.name }]
const axes = [
  { axis: 'Input', note: c.adapts.input, values: 'pointer, touch' },
  { axis: 'Surface', note: c.adapts.surface, values: 'app, landing' },
  { axis: 'Width', note: c.adapts.width, values: 'narrow, regular, wide' },
]
/** An id for each code block, so its copy button finds the text. */
const codeId = (example: string) => `code-${example}`
</script>

<template>
  <div class="site">
    <SiteHeader current="components" :path="c.path" menu="components-nav" />
    <div class="docs">
      <GalleryNav :current="c.name" />
      <Page id="main" class="docs-page gallery-page">
        <template #header>
          <Breadcrumb :items="crumbs" />
          <h1 class="docs-title">{{ c.name }}</h1>
          <p class="docs-lead" v-html="inlineHTML(c.purpose)" />
          <dl class="gallery-when">
            <div><dt>Use it</dt><dd v-html="inlineHTML(c.use)" /></div>
            <div><dt>Not for</dt><dd v-html="inlineHTML(c.avoid)" /></div>
          </dl>
        </template>

        <Island when="idle" :component="first.island" />

        <section class="gallery-section" aria-labelledby="code">
          <h2 id="code">Code</h2>
          <figure class="gallery-code">
            <figcaption>
              <span>{{ first.file.replace('packages/ui/', '') }}</span>
              <Island when="idle" :component="CopyCommand" :props="{ from: codeId(first.name) }" />
            </figcaption>
            <div :id="codeId(first.name)" v-html="first.html" />
          </figure>
        </section>

        <section class="gallery-section" aria-labelledby="adapts">
          <h2 id="adapts">Adapts</h2>
          <div class="gallery-table">
            <table>
              <thead><tr><th scope="col">Axis</th><th scope="col">Between</th><th scope="col">What changes</th></tr></thead>
              <tbody>
                <tr v-for="a in axes" :key="a.axis"><th scope="row">{{ a.axis }}</th><td>{{ a.values }}</td><td v-html="inlineHTML(a.note)" /></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="gallery-section" aria-labelledby="api">
          <h2 id="api">API</h2>
          <h3 id="props">Props</h3>
          <div v-if="c.props.length" class="gallery-table">
            <table class="gallery-api">
              <thead><tr><th scope="col">Prop</th><th scope="col">Type</th><th scope="col">Default</th><th scope="col">Description</th></tr></thead>
              <tbody>
                <tr v-for="p in c.props" :key="p.name">
                  <th scope="row"><code>{{ p.name }}</code><span v-if="p.required" class="gallery-required"> required</span></th>
                  <td><code>{{ p.type }}</code></td>
                  <td><code v-if="p.default !== null">{{ p.default }}</code><template v-else><span aria-hidden="true">—</span><span class="weft-visually-hidden">none</span></template></td>
                  <td v-html="inlineHTML(p.description)" />
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else>None.</p>
          <h3 id="slots">Slots</h3>
          <div v-if="c.slots.length" class="gallery-table">
            <table class="gallery-api">
              <thead><tr><th scope="col">Slot</th><th scope="col">Props</th></tr></thead>
              <tbody>
                <tr v-for="s in c.slots" :key="s.name">
                  <th scope="row"><code>{{ s.name }}</code><span v-if="s.dynamic"> (dynamic)</span></th>
                  <td>
                    <template v-if="s.props.length"><code v-for="sp in s.props" :key="sp.name">{{ sp.name }}: {{ sp.type }}</code></template>
                    <template v-else><span aria-hidden="true">—</span><span class="weft-visually-hidden">none</span></template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else>None.</p>
          <h3 id="events">Events</h3>
          <div v-if="c.emits.length" class="gallery-table">
            <table class="gallery-api">
              <thead><tr><th scope="col">Event</th><th scope="col">Payload</th><th scope="col">Description</th></tr></thead>
              <tbody>
                <tr v-for="e in c.emits" :key="e.name">
                  <th scope="row"><code>{{ e.name }}</code></th>
                  <td><code>{{ e.payload }}</code></td>
                  <td v-html="inlineHTML(e.description)" />
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else>None.</p>
        </section>

        <section v-if="more.length" class="gallery-section" aria-labelledby="examples">
          <h2 id="examples">More examples</h2>
          <article v-for="e in more" :key="e.name" class="gallery-more" :aria-labelledby="`example-${e.name}`">
            <h3 :id="`example-${e.name}`">{{ e.title }}</h3>
            <div class="gallery-preview" data-weft-no-prefetch :data-example="e.file">
              <Island v-if="e.interactive" when="visible" :component="e.island" />
              <WeftProvider v-else inset><component :is="e.component" /></WeftProvider>
            </div>
            <figure class="gallery-code">
              <figcaption>
                <span>{{ e.file.replace('packages/ui/', '') }}</span>
                <Island when="idle" :component="CopyCommand" :props="{ from: codeId(e.name) }" />
              </figcaption>
              <div :id="codeId(e.name)" v-html="e.html" />
            </figure>
          </article>
        </section>

        <section class="gallery-section" aria-labelledby="agents">
          <h2 id="agents">For agents</h2>
          <p>
            This page's catalogue entry, as JSON: <a :href="`/components/${c.slug}.json`">/components/{{ c.slug }}.json</a>.
            Every component's: <a href="/catalog.json">/catalog.json</a>.
          </p>
          <!-- The file itself, as an agent gets it, loaded when it is near: the page does not carry it twice. -->
          <iframe class="gallery-raw" :src="`/components/${c.slug}.json`" loading="lazy" :title="`${c.name}'s catalogue entry, as JSON`" />
        </section>

        <nav v-if="prev || next" class="docs-pager" :aria-label="`Previous and next in ${groupLabel(c.group)}`">
          <Link v-if="prev" :to="prev.path" rel="prev"><span>Previous</span>{{ prev.name }}</Link>
          <Link v-if="next" :to="next.path" rel="next"><span>Next</span>{{ next.name }}</Link>
        </nav>
      </Page>
    </div>
    <SiteFooter />
  </div>
</template>
