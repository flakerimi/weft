<script setup vapor lang="ts">
import { onMounted, ref } from 'vue'
import { useParams } from '@weft/ui'

const id = ref('')
// In the browser only: at build the page has no address, and useParams() is {}.
onMounted(() => {
  id.value = useParams().id ?? ''
})
</script>

<template>
  <p class="product">Product {{ id }}</p>
</template>
