<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, InputTags } from '@weft/ui'

const keywords = ref(['vapor', 'islands'])
</script>

<template>
  <FormField label="Keywords" hint="Enter or comma" name="keywords">
    <InputTags v-model="keywords" placeholder="Add a keyword" add-on-paste add-on-blur :max="8" />
  </FormField>
</template>
