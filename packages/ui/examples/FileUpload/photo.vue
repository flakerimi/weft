<script setup vapor lang="ts">
import { ref } from 'vue'
import { FileUpload, FormField } from '@weft/ui'

// One image; on a phone the button goes straight to the front camera.
const avatar = ref<File | null>(null)
</script>

<template>
  <FormField label="Profile photo" name="avatar">
    <FileUpload v-model="avatar" accept="image/*" capture="user" />
  </FormField>
</template>
