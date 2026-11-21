<script setup vapor lang="ts">
import { ref } from 'vue'
import FormField from '../../../src/forms/FormField.vue'
import FileUpload from '../../../src/forms/FileUpload.vue'
import type { FileRejection, Upload } from '../../../src/forms/files.ts'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = withDefaults(defineProps<{
  input?: Input
  initial?: File[] | File | null
  multiple?: boolean
  accept?: string
  capture?: 'user' | 'environment'
  maxSize?: number
  maxFiles?: number
  upload?: Upload
  disabled?: boolean
  description?: string
  locale?: string
}>(), { initial: undefined })
const input = ref<Input | undefined>(props.input)
const value = ref<File[] | File | null>(props.initial ?? (props.multiple ? [] : null))
const rejected = ref<FileRejection[]>([])
const uploaded = ref<[string, unknown][]>([])
const failed = ref<string[]>([])
const changes = ref(0)
const submitted = ref<string[]>([])
function onSubmit(event: Event) {
  event.preventDefault()
  submitted.value = new FormData(event.target as HTMLFormElement).getAll('files').map((f) => (f as File).name)
}
defineExpose({ input, value, rejected, uploaded, failed, changes, submitted })
</script>

<template>
  <WeftProvider :input="input" :locale="locale">
    <form @submit="onSubmit">
      <FormField label="Attachments" name="files" description="Up to 5 MB each">
        <FileUpload
          v-model="value"
          :multiple="multiple"
          :accept="accept"
          :capture="capture"
          :max-size="maxSize"
          :max-files="maxFiles"
          :upload="upload"
          :disabled="disabled"
          :description="description"
          @reject="rejected.push(...$event)"
          @uploaded="(file, result) => uploaded.push([file.name, result])"
          @error="(file) => failed.push(file.name)"
          @change="changes++"
        />
      </FormField>
      <button class="submit" type="submit">Send</button>
    </form>
  </WeftProvider>
</template>
