<script setup vapor lang="ts">
import { ref } from 'vue'
import { FileUpload, FormField } from '@weft/ui'

// A gallery: image tiles inside the zone, each new choice in the dialog starting the set again.
const photos = ref<File[]>([])
</script>

<template>
  <FormField label="Photos" name="photos">
    <FileUpload v-model="photos" multiple accept="image/*" layout="grid" position="inside" reset />
  </FormField>
</template>
