<script lang="ts">
import { contentPaths } from 'virtual:weft/content'

export const getPaths = async () => contentPaths('docs')
</script>

<script setup vapor lang="ts">
// A docs page: content/docs/<slug>.md, with the docs' sidebar in the order of
// their `order`, the page's headings as "On this page", and links to the
// pages before and after it. Everything is drawn at build time: the page
// ships no script, except a copy button for a one-line command marked `copy`.
// On a narrow screen the sidebar is a drawer the header's Menu button opens,
// through the HTML popover attribute, so that works without JavaScript too.
import { Breadcrumb, Button, Island, Link, NavigationMenu, Page, useHead, type BreadcrumbItem, type NavItem } from '@weft/ui'
import { useCollection, useContent } from 'virtual:weft/content'
import '@weft/vite/code.css'
import '../../styles/fonts.css'
import '../../styles/docs.css'
import CopyCommand from '../../components/CopyCommand.vue'
import SiteFooter from '../../components/SiteFooter.vue'
import SiteHeader from '../../components/SiteHeader.vue'

const props = defineProps<{ slug: string }>()
const page = useContent('docs', props.slug)
// A table scrolls inside a container of its own on a narrow screen, keeping its table semantics.
const html = (part: { html: string }) => part.html.replaceAll('<table>', '<div class="docs-table"><table>').replaceAll('</table>', '</table></div>')
// A copy block is one command, shown and copied by the landing page's CopyCommand island.
for (const part of page.parts) {
  if ('code' in part && part.code.trim().includes('\n')) {
    throw new Error(`docs: ${page.file} has a copy block of several lines; mark only one-line commands copy.`)
  }
}
useHead({ title: `${page.frontmatter.title} — Weft docs`, description: page.frontmatter.description })

/** The sidebar's groups: a folder of content/docs/ is one, under its name; a page at the top stands alone. */
const SECTIONS: Record<string, string> = {
  'getting-started': 'Getting started',
  guides: 'Guides',
  reference: 'Reference',
}
const sectionOf = (slug: string) => (slug.includes('/') ? slug.slice(0, slug.indexOf('/')) : '')
const groups: NavItem[][] = []
let last: string | null = null
for (const doc of useCollection('docs')) {
  const section = sectionOf(doc.slug)
  if (section !== last || !section) {
    if (section && !SECTIONS[section]) throw new Error(`docs: content/docs/${section}/ has no name in the sidebar; add it to SECTIONS.`)
    groups.push(section ? [{ type: 'label', label: SECTIONS[section] }] : [])
    last = section
  }
  groups.at(-1)!.push({ label: doc.title, to: doc.path, active: doc.slug === page.slug })
}

const section = sectionOf(page.slug)
const crumbs: BreadcrumbItem[] = [
  { label: 'Docs', to: '/docs' },
  ...(section ? [{ label: SECTIONS[section]! }] : []),
  ...(page.slug ? [{ label: page.frontmatter.title }] : []),
]
</script>

<template>
  <div class="site">
    <SiteHeader current="docs" :path="page.path" menu="docs-nav" />
    <div class="docs">
      <aside id="docs-nav" class="docs-nav" popover>
        <Button class="site-menu docs-close" variant="ghost" tone="neutral" size="sm" popovertarget="docs-nav" popovertargetaction="hide">Close</Button>
        <NavigationMenu :items="groups" orientation="vertical" label="Docs" />
      </aside>
      <Page id="main" class="docs-page">
        <template #header>
          <Breadcrumb v-if="page.slug" :items="crumbs" />
          <h1 class="docs-title">{{ page.frontmatter.title }}</h1>
          <p class="docs-lead">{{ page.frontmatter.description }}</p>
        </template>
        <div class="docs-body">
          <nav v-if="page.toc.length" class="docs-toc" aria-labelledby="docs-toc-title">
            <p id="docs-toc-title">On this page</p>
            <ul>
              <li v-for="entry in page.toc" :key="entry.id" :data-depth="entry.depth">
                <Link :to="`#${entry.id}`">{{ entry.text }}</Link>
              </li>
            </ul>
          </nav>
          <article class="docs-prose">
            <template v-for="(part, i) in page.parts" :key="i">
              <Island v-if="'code' in part" when="idle" :component="CopyCommand" :props="{ command: part.code.trim() }" />
              <div v-else v-html="html(part)" />
            </template>
          </article>
          <nav v-if="page.prev || page.next" class="docs-pager" aria-label="Previous and next">
            <Link v-if="page.prev" :to="page.prev.path" rel="prev">
              <span>Previous</span>{{ page.prev.title }}
            </Link>
            <Link v-if="page.next" :to="page.next.path" rel="next">
              <span>Next</span>{{ page.next.title }}
            </Link>
          </nav>
        </div>
      </Page>
    </div>
    <SiteFooter />
  </div>
</template>
