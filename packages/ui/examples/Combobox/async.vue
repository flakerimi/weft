<script setup vapor lang="ts">
import { ref } from 'vue'
import { Combobox, FormField, type SelectItems } from '@weft/ui'

const city = ref<{ label: string; country: string } | undefined>()
const cities = [
  ['Amsterdam', 'Netherlands'], ['Athens', 'Greece'], ['Berlin', 'Germany'], ['Lisbon', 'Portugal'],
  ['Ljubljana', 'Slovenia'], ['Madrid', 'Spain'], ['Oslo', 'Norway'], ['Paris', 'France'],
  ['Prishtina', 'Kosovo'], ['Rome', 'Italy'], ['Tirana', 'Albania'], ['Vienna', 'Austria'],
].map(([label, country]) => ({ label: label!, country: country!, description: country }))

// A stand-in for a server search: answers after a moment, and stops when the signal aborts it.
function load(term: string, { signal }: { signal: AbortSignal }): Promise<SelectItems> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      const t = term.toLowerCase()
      resolve(cities.filter((c) => c.label.toLowerCase().includes(t) || c.country.toLowerCase().includes(t)))
    }, 400)
    signal.addEventListener('abort', () => {
      clearTimeout(timer)
      reject(signal.reason)
    })
  })
}
</script>

<template>
  <FormField label="City" description="Searched on the server as you type.">
    <Combobox v-model="city" :load="load" by="label" placeholder="Search cities" />
  </FormField>
</template>
