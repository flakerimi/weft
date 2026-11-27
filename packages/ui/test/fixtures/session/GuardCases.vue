<script setup vapor lang="ts">
import { Guard } from '../../../src/index.ts'
defineProps<{ case: string; perm?: string; signIn?: string }>()
</script>

<template>
  <Guard v-if="$props.case === 'plain'" :perm="perm" :sign-in="signIn">
    <template #default="{ user }"><p class="secret">For {{ user.name }}</p></template>
  </Guard>
  <Guard v-else-if="$props.case === 'slots'" :perm="perm">
    <p class="secret">Secret</p>
    <template #denied><p class="mine-denied">Not for you</p></template>
    <template #loading><p class="mine-loading">Wait</p></template>
  </Guard>
</template>
