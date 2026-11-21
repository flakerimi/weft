<script setup vapor lang="ts">
// The gallery's sidebar: every component under its group, the current one
// marked. As the docs' sidebar does, it is a popover the header's Menu
// button opens as a drawer on a narrow screen, with no script.
import { Button, NavigationMenu, type NavItem } from '@weft/ui'
import { groups } from '../../gallery/catalog.ts'

const props = defineProps<{
  /** The current component's name, if the page is one's. */
  current?: string
}>()

const items: NavItem[][] = [
  [{ label: 'All components', to: '/components', active: !props.current }],
  ...groups.map((g) => [
    { type: 'label' as const, label: g.label },
    ...g.components.map((c) => ({ label: c.name, to: c.path, active: c.name === props.current })),
  ]),
]
</script>

<template>
  <aside id="components-nav" class="docs-nav gallery-nav" popover>
    <Button class="site-menu docs-close" variant="ghost" tone="neutral" size="sm" popovertarget="components-nav" popovertargetaction="hide">Close</Button>
    <NavigationMenu :items="items" orientation="vertical" label="Components" />
  </aside>
</template>
