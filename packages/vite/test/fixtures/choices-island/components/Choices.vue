<script setup vapor lang="ts">
// Radios and selects whose state a person may change before the island
// hydrates (keep-typed.test.ts), in the orders and defaults that matter: a
// radio group whose default comes first, one with no default, a select whose
// model matches none of its options (the browser shows the first), one whose
// model matches, a multiple select, and a Weft RadioGroup whose default is first.
import { ref } from 'vue'
import { RadioGroup } from '@weft/ui'

const first = ref('a')
const none = ref<string | null>(null)
const blank = ref('')
const pick = ref('b')
const many = ref<string[]>(['y'])
const plan = ref<string | number | null>('free')
</script>

<template>
  <div class="choices">
    <input type="radio" name="first" value="a" v-model="first" /><input type="radio" name="first" value="b" v-model="first" /><input type="radio" name="first" value="c" v-model="first" />
    <input type="radio" name="none" value="a" v-model="none" /><input type="radio" name="none" value="b" v-model="none" />
    <select class="blank" v-model="blank"><option value="x">x</option><option value="y">y</option></select>
    <select class="pick" v-model="pick"><option value="a">a</option><option value="b">b</option><option value="c">c</option></select>
    <select class="many" multiple v-model="many"><option value="x">x</option><option value="y">y</option><option value="z">z</option></select>
    <RadioGroup v-model="plan" :items="['free', 'pro', 'team']" aria-label="Plan" />
    <output>{{ first }}|{{ none }}|{{ blank }}|{{ pick }}|{{ many.join(',') }}|{{ plan }}</output>
  </div>
</template>
