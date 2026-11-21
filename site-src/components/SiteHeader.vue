<script setup vapor lang="ts">
// The site's header: the wordmark, then its sections. Plain links, so it
// works without JavaScript at every width; on a narrow screen Components and
// GitHub give way. A page with a sidebar names it in `menu`: under 768 px a
// Menu button then opens it, through the HTML popover attribute (the sidebar
// is a popover element), so that works without JavaScript too.
import { Button, Header, Link } from '@weft/ui'
import SiteMark from './SiteMark.vue'
import '../styles/site.css'

defineProps<{
  /** The section the page is in, drawn as current. */
  current?: 'docs' | 'components'
  /** The id of the page's sidebar, a `popover` element the narrow Menu button opens. */
  menu?: string
}>()
</script>

<template>
  <Header>
    <template #leading><SiteMark /></template>
    <nav class="site-nav" aria-label="Site">
      <Link to="/docs" standalone :active="current === 'docs' || undefined">Docs</Link>
      <Link to="/components" standalone class="site-wide" :active="current === 'components' || undefined">Components</Link>
      <Link to="/#agents" standalone>For agents</Link>
      <Link to="https://github.com/flakerimi/weft" standalone class="site-wide">GitHub</Link>
    </nav>
    <template v-if="menu" #actions>
      <Button class="site-menu" variant="soft" tone="neutral" size="sm" :popovertarget="menu" :aria-controls="menu">Menu</Button>
    </template>
  </Header>
</template>
