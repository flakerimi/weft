<script setup vapor lang="ts">
// A form with one text field and no button: Enter in the field submits it.
import { reactive, ref } from 'vue'
import { Form, Input } from '../../../src/index.ts'

const state = reactive({ q: '' })
const sent = ref<string[]>([])
defineExpose({ sent })
</script>

<template>
  <Form :state="state" @submit="sent.push($event.q)"><Input v-model="state.q" aria-label="Search" /></Form>
</template>
