<script setup lang="ts">
// The benchmark's island: bench/weft/components/Counter.vue without `vapor`.
// Its styles are in bench/shared/landing.css.
import { ref } from 'vue'

const props = defineProps<{ start: number }>()
const count = ref(props.start)
</script>

<template>
  <div class="counter">
    <button type="button" class="counter__button" aria-label="Decrease" @click="count--">−</button>
    <output class="counter__value" aria-live="polite">{{ count }}</output>
    <button type="button" class="counter__button" aria-label="Increase" @click="count++">+</button>
  </div>
</template>
