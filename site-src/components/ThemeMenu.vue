<script setup vapor lang="ts">
// The header's Theme menu: Weft's Button opens a popover (the HTML popover
// attribute, as the narrow Menu button opens the sidebar) of the site's six
// themes, each a Button. The server draws the button and the popover, empty
// and hidden. Its pre-paint snippet (themes/menu.ts) applies the kept choice
// before the first paint; the menu's module, which the page fetches when the
// menu is first used, written beside the pages with useFile, fills the
// popover with a copy of the button per theme, its CSS with them, and does
// what a press asks. So the menu costs every page's first view little: page
// budgets are never raised. Each theme but weft is a stylesheet of its own,
// imported by its URL, which the page build writes under a hashed name.
// Without JavaScript the pages are Weft and an open menu shows nothing.
import { Button, usePrePaint } from '@weft/ui'
import { useFile } from 'virtual:weft/files'
import neutral from '/.site-themes/neutral.css?url'
import lisa from '/.site-themes/lisa.css?url'
import synthwave from '/.site-themes/synthwave.css?url'
import editor from '/.site-themes/editor.css?url'
import sandstorm from '/.site-themes/sandstorm.css?url'
import { SITE_THEMES, THEME_MODULE, themeModule, themeSnippet } from '../themes/menu.ts'
import menuCSS from '../styles/theme-menu.css?inline'

const urls: Record<(typeof SITE_THEMES)[number]['name'], string> = { neutral, lisa, synthwave, editor, sandstorm }
usePrePaint('site:theme', themeSnippet())
useFile(THEME_MODULE, themeModule(urls, menuCSS))
</script>

<template>
  <Button class="site-theme" variant="ghost" tone="neutral" size="sm" popovertarget="site-themes">Theme</Button>
  <div id="site-themes" class="site-themes" popover hidden role="group" aria-label="Theme" />
</template>
