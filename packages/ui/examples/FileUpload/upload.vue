<script setup vapor lang="ts">
import { ref } from 'vue'
import { FileUpload, FormField, type Upload } from '@weft/ui'

// Each file is sent as it is added, with its progress; removing it while it uploads cancels it.
const files = ref<File[]>([])
const upload: Upload = (file, { signal, onProgress }) =>
  new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/files')
    xhr.upload.onprogress = (e) => e.lengthComputable && onProgress(e.loaded / e.total)
    xhr.onload = () => (xhr.status < 300 ? resolve(xhr.response) : reject(new Error(xhr.statusText)))
    xhr.onerror = () => reject(new Error('Network error'))
    signal.addEventListener('abort', () => xhr.abort())
    const body = new FormData()
    body.append('file', file)
    xhr.send(body)
  })
</script>

<template>
  <FormField label="Receipts">
    <FileUpload v-model="files" multiple :upload="upload" />
  </FormField>
</template>
