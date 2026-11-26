<script setup vapor lang="ts">
// Two forms on one page: one in an island, one drawn by the page.
import { Button, Form, Island } from '@weft/ui'
import Login from '../components/Login.vue'
</script>

<template>
  <main>
    <Island when="load" :component="Login" />
    <Form :state="{}"><Button type="submit" label="Search" /></Form>
  </main>
</template>
