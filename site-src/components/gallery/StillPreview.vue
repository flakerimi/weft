<script setup vapor lang="ts">
// A part of the page drawn once by the server and never hydrated: an index
// thumbnail. It gives its slot an id scope of its own, as an island does,
// so ids inside stay unique on the page, and an Island inside draws its
// component in place rather than becoming an island: a thumbnail ships no
// script, whatever its example holds.
import { provide } from 'vue'
import { ID_SCOPE, type IdScope } from '@weft/ui'

const props = defineProps<{
  /** The scope's prefix, unique on the page: its ids are <name>-<scope>-<n>. */
  scope: string
}>()
provide<IdScope>(ID_SCOPE, { prefix: props.scope, next: 0 })
</script>

<template>
  <slot />
</template>
