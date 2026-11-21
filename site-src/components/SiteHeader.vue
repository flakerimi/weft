<script setup vapor lang="ts">
// The site's header: a skip link, the wordmark, then its sections. Plain
// links, so it works without JavaScript at every width; on a narrow screen
// Components and GitHub give way. GitHub is there only when the build says
// the repository is public (WEFT_REPO_PUBLIC=1). For agents is /skills: the
// skills, llms.txt and the catalogue. The section the page is in
// is marked aria-current="true"; its own front page, "page". A page with a sidebar names it in `menu`: under 768 px a
// Menu button then opens it, through the HTML popover attribute (the sidebar
// is a popover element), so that works without JavaScript too.
import { Button, Header, Link } from '@weft/ui'
import SiteMark from './SiteMark.vue'
import '../styles/site.css'

/** The site's sections, each by its front page. */
type Section = 'docs' | 'components' | 'agents'
const FRONT: Record<Section, string> = { docs: '/docs', components: '/components', agents: '/skills' }

const props = withDefaults(
  defineProps<{
    /** The section the page is in, drawn as current. */
    current?: Section
    /** The page's path, so the section's own front page is the current page. */
    path?: string
    /** The id of the page's sidebar, a `popover` element the narrow Menu button opens. */
    menu?: string
    /** Whether it stays at the top as the page scrolls; the landing page's scrolls away. */
    sticky?: boolean
  }>(),
  { sticky: true },
)

const repoPublic = __WEFT_REPO_PUBLIC__
/** aria-current for a section's link: "page" on its front page, "true" anywhere else in it. */
const currentOf = (section: Section) => (props.current !== section ? undefined : props.path === FRONT[section] ? 'page' : 'true')
</script>

<template>
  <a class="site-skip" href="#main">Skip to the content</a>
  <Header :sticky="sticky">
    <template #leading><SiteMark /></template>
    <nav class="site-nav" aria-label="Site">
      <Link to="/docs" standalone :aria-current="currentOf('docs')">Docs</Link>
      <Link to="/components" standalone class="site-wide" :aria-current="currentOf('components')">Components</Link>
      <Link to="/skills" standalone :aria-current="currentOf('agents')">For agents</Link>
      <Link v-if="repoPublic" to="https://github.com/flakerimi/weft" standalone class="site-wide">GitHub</Link>
    </nav>
    <template v-if="menu" #actions>
      <Button class="site-menu" variant="soft" tone="neutral" size="sm" :popovertarget="menu" :aria-controls="menu">Menu</Button>
    </template>
  </Header>
</template>
