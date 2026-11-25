<script>
  // The benchmark's island, as a Svelte component. Its styles are in
  // bench/shared/landing.css.
  let { start } = $props()
  let count = $state(start)
</script>

<div class="counter">
  <button type="button" class="counter__button" aria-label="Decrease" onclick={() => count--}>−</button>
  <output class="counter__value" aria-live="polite">{count}</output>
  <button type="button" class="counter__button" aria-label="Increase" onclick={() => count++}>+</button>
</div>
