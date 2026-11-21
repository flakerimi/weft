<script setup vapor lang="ts">
// The sidebar of the pages for agents: the skills, each by its name, and the
// raw files an agent reads (llms.txt, the full text, the catalogue). As the
// docs' sidebar does, it is a popover the header's Menu button opens as a
// drawer on a narrow screen, with no script.
import { Button, NavigationMenu, type NavItem } from '@weft/ui'
import { useCollection } from 'virtual:weft/content'

const props = defineProps<{
  /** The current page's path. */
  current: string
}>()

const items: NavItem[][] = [
  [{ label: 'For agents', to: '/skills', active: props.current === '/skills' }],
  [{ type: 'label', label: 'Skills' }, ...useCollection('skills').map((s) => ({ label: s.title, to: s.path, active: s.path === props.current }))],
  [
    { type: 'label', label: 'Files' },
    { label: 'llms.txt', to: '/llms.txt' },
    { label: 'llms-full.txt', to: '/llms-full.txt' },
    { label: 'catalog.json', to: '/catalog.json' },
  ],
]
</script>

<template>
  <aside id="skills-nav" class="docs-nav" popover>
    <Button class="site-menu docs-close" variant="ghost" tone="neutral" size="sm" popovertarget="skills-nav" popovertargetaction="hide">Close</Button>
    <NavigationMenu :items="items" orientation="vertical" label="For agents" />
  </aside>
</template>
