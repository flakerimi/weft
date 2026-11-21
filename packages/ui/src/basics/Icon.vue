<script setup vapor lang="ts">
// A Lucide icon by name, drawn as an inline svg in the text colour. The build
// bundles only the icons it finds used (see icons.ts); an unknown name fails
// the build there, with a sentence naming the file.
import { computed } from 'vue'
import { drawIcon } from './icons.ts'
import '../styles/base.css'
import '../styles/icon.css'

const props = defineProps<{
  /** The Lucide icon's name, as on lucide.dev: `trash`, `chevron-down`. */
  name: string
  /** `sm` 16 px, `md` 20 px, `lg` 24 px; without it the icon is 1.25em, sized with the text. */
  size?: 'sm' | 'md' | 'lg'
  /** What the icon means, for an icon that stands alone; without it the icon is decorative and hidden from assistive technology. */
  label?: string
}>()

const body = computed(() => drawIcon(props.name))
</script>

<template>
  <svg
    class="weft-icon"
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    :data-size="size"
    :role="label ? 'img' : undefined"
    :aria-label="label || undefined"
    :aria-hidden="label ? undefined : 'true'"
    v-html="body"
  ></svg>
</template>
