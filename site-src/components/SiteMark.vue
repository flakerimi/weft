<script setup vapor lang="ts">
// The owner's wordmark, public/weft.svg (docs/design/weft-wordmark.svg as it
// is), as a link home. CSS draws it through the file as a mask: its letters
// in the text colour, so it follows light and dark, and its thread in
// saffron. The file is fetched once and cached; the page carries none of it.
import '../styles/site.css'
</script>

<template>
  <a class="site-mark" href="/" aria-label="Weft home" />
</template>
