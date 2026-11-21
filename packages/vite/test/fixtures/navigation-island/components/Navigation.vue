<script setup vapor lang="ts">
// Every navigation and disclosure component in one island: the server
// renders the tab list and the selected panel, the path, the pages, the
// steps, the sections with their hidden content, the tree's rows, the
// menus and the app shell, none of the dropdowns or sheets that open; the
// browser hydrates that markup in place (navigation-island.test.ts), then
// they work.
import { ref } from 'vue'
import {
  Accordion,
  AppShell,
  Breadcrumb,
  Collapsible,
  Footer,
  Header,
  NavigationMenu,
  Pagination,
  Stepper,
  Tabs,
  Tree,
  type NavItem,
} from '@weft/ui'

const tab = ref<string | number>('one')
const page = ref(3)
const step = ref<string | number>(0)
const section = ref<string | string[] | null>(null)
const open = ref(false)
const node = ref<string | string[] | null>(null)
const nav: NavItem[] = [
  { label: 'Guide', to: '/guide', icon: 'book-open' },
  { label: 'More', icon: 'box', children: [{ label: 'Blog', to: '/blog' }] },
]
</script>

<template>
  <AppShell :items="nav" title="Acme" :persist="false">
    <Header title="Acme" :sticky="false"><NavigationMenu :items="nav" label="Top" /></Header>
    <Breadcrumb :items="[{ label: 'Home', to: '/' }, { label: 'Guide' }]" />
    <Tabs v-model="tab" :items="[{ label: 'One', value: 'one', content: 'First' }, { label: 'Two', value: 'two', content: 'Second' }]" aria-label="Views" />
    <Pagination v-model:page="page" :total="100" />
    <Stepper v-model="step" :items="[{ title: 'Start' }, { title: 'Finish' }]" />
    <Accordion v-model="section" :items="[{ label: 'Shipping', value: 'ship', content: 'Two days' }, { label: 'Returns', value: 'back', content: 'Thirty days' }]" />
    <Collapsible v-model:open="open" label="Details"><template #content>Hidden details</template></Collapsible>
    <Tree v-model="node" :items="[{ label: 'src', value: 'src', children: [{ label: 'app.ts', value: 'app' }] }, { label: 'README', value: 'readme' }]" aria-label="Files" />
    <Footer><span>Footer</span></Footer>
    <output>{{ tab }}|{{ page }}|{{ step }}|{{ section }}|{{ open }}|{{ node }}</output>
  </AppShell>
</template>
