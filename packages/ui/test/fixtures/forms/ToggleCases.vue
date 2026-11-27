<script setup vapor lang="ts">
import { ref } from 'vue'
import Toggle from '../../../src/forms/Toggle.vue'
import ToggleGroup from '../../../src/forms/ToggleGroup.vue'
import FormField from '../../../src/forms/FormField.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

withDefaults(
  defineProps<{
    multiple?: boolean
    orientation?: 'horizontal' | 'vertical'
    loading?: boolean
    disabled?: boolean
    input?: Input
    dir?: 'ltr' | 'rtl'
  }>(),
  {},
)
const bold = ref(false)
const align = ref<string | number | null | (string | number)[]>(null)
const styles = ref<(string | number)[]>(['italic'])
const items = [
  { label: 'Left', value: 'left', icon: 'align-left' },
  { label: 'Center', value: 'center' },
  { label: 'Right', value: 'right', disabled: true },
  { label: 'Justify', value: 'justify' },
]
defineExpose({ bold, align, styles })
</script>

<template>
  <WeftProvider :input="input">
    <form :dir="dir" @submit.prevent>
      <Toggle v-model="bold" class="bold" label="Bold" name="bold" :loading="loading" :disabled="disabled" />
      <ToggleGroup v-if="!multiple" v-model="align" class="align" :items="items" name="align" aria-label="Alignment" :orientation="orientation" :disabled="disabled" />
      <ToggleGroup v-else v-model="styles" class="styles" multiple :items="[{ label: 'Bold', value: 'bold' }, { label: 'Italic', value: 'italic' }, { label: 'Underline', value: 'underline' }]" name="styles" aria-label="Styles" />
      <FormField label="Size" class="size-field"><ToggleGroup class="sizes" :items="[{ label: 'S', value: 's' }, { label: 'M', value: 'm' }]" /></FormField>
    </form>
  </WeftProvider>
</template>
