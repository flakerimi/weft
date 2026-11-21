<script setup vapor lang="ts">
import { ref } from 'vue'
import { FileUpload, FormField } from '@weft/ui'

// Drop, paste or browse on pointer; one button for the camera, photos or files on touch.
const files = ref<File[]>([])
</script>

<template>
  <FormField label="Attachments" name="attachments">
    <FileUpload v-model="files" multiple accept="image/*,.pdf" :max-size="5_000_000" description="Images or PDFs, up to 5 MB each" />
  </FormField>
</template>
