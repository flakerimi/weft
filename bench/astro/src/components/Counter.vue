<script setup lang="ts">
// The benchmark's island: bench/weft/components/Counter.vue without `vapor`,
// since Astro's Vue integration renders with Vue's virtual DOM.
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

<style>
.counter { display: inline-flex; align-items: center; gap: 12px; }
.counter__button {
  width: 44px;
  height: 44px;
  font: inherit;
  font-size: 1.25em;
  color: #fff;
  background: #3b5bdb;
  border: 0;
  border-radius: 10px;
  cursor: pointer;
}
.counter__value { min-width: 3ch; text-align: center; font-size: 1.5em; font-variant-numeric: tabular-nums; }
</style>
