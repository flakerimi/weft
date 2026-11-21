<script setup vapor lang="ts">
// An island that loads a component on demand (a dynamic import) and draws
// it once it is in: never on the server.
import { onMounted, shallowRef, type Component } from 'vue'

const later = shallowRef<Component>()
onMounted(async () => {
  later.value = (await import('./Later.vue')).default
})
</script>

<template>
  <div class="on-demand"><component :is="later" v-if="later" /></div>
</template>
