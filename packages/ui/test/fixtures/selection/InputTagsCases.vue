<script setup vapor lang="ts">
import { ref } from 'vue'
import FormField from '../../../src/forms/FormField.vue'
import InputTags from '../../../src/selection/InputTags.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = defineProps<{
  input?: Input
  dir?: 'ltr' | 'rtl'
  initial?: string[]
  max?: number
  addOnPaste?: boolean
  addOnBlur?: boolean
  addOnTab?: boolean
  duplicate?: boolean
  disabled?: boolean
  upper?: boolean
  /** Draw each tag's text through the `item-label` slot, or its deprecated old name `item-text`. */
  custom?: 'item-label' | 'item-text'
}>()
const tags = ref<unknown[]>(props.initial ?? [])
const log = ref<string[]>([])
const submitted = ref(0)
defineExpose({ tags, log, submitted })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <form @submit.prevent="submitted++">
        <FormField label="Tags" name="tags" hint="Comma or Enter">
          <InputTags
            v-model="tags"
            placeholder="Add a tag"
            :max="max"
            :add-on-paste="addOnPaste"
            :add-on-blur="addOnBlur"
            :add-on-tab="addOnTab"
            :duplicate="duplicate"
            :disabled="disabled"
            :convert-value="upper ? (t: string) => t.toUpperCase() : undefined"
            :display-value="upper ? (v: unknown) => `#${v}` : undefined"
            @add-tag="(v: unknown) => log.push(`add:${v}`)"
            @remove-tag="(v: unknown) => log.push(`remove:${v}`)"
            @invalid="(v: unknown) => log.push(`invalid:${v}`)"
          >
            <template v-if="custom === 'item-label'" #item-label="{ item, index }"><b class="custom">{{ index }}:{{ item }}</b></template>
            <template v-if="custom === 'item-text'" #item-text="{ item, index }"><i class="custom">{{ index }}={{ item }}</i></template>
          </InputTags>
        </FormField>
      </form>
      <button class="after">after</button>
    </div>
  </WeftProvider>
</template>
