<script setup vapor lang="ts">
import { Link } from '../../../src/index.ts'
// external and active stay unset unless a test sets them, as on a real Link.
withDefaults(
  defineProps<{ to?: string; href?: string; rel?: string; target?: string; external?: boolean; active?: boolean; disabled?: boolean; onPress?: (e: MouseEvent) => void }>(),
  { external: undefined, active: undefined },
)
</script>

<template>
  <Link :to="to" :href="href" :rel="rel" :target="target" :external="external" :active="active" :disabled="disabled" @click="onPress?.($event)">Docs</Link>
</template>
