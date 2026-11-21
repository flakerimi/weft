<script setup vapor lang="ts">
// The page for an address that has none: the site's header and footer, and
// the ways back in. Like every page with no island, it ships no script.
import { Link, Page, useHead } from '@weft/ui'
import '../styles/fonts.css'
import '../styles/not-found.css'
import SiteFooter from '../components/SiteFooter.vue'
import SiteHeader from '../components/SiteHeader.vue'

useHead({ title: 'Not found — Weft', description: 'Nothing lives at this address.' })
</script>

<template>
  <div class="site">
    <SiteHeader />
    <Page id="main" class="not-found">
      <template #header>
        <h1 class="not-found__title">Not found</h1>
        <p class="not-found__lead">Nothing lives at this address. It may have moved, or the link may be wrong.</p>
      </template>
      <ul class="not-found__links">
        <li><Link to="/">The front page</Link></li>
        <li><Link to="/docs">The docs</Link>, from installing Weft to serving a site from Go</li>
        <li><Link to="/components">The components</Link>, each with its examples and API</li>
        <li><Link to="/skills">For agents</Link>: skills, llms.txt and the catalogue</li>
      </ul>
    </Page>
    <SiteFooter />
  </div>
</template>
