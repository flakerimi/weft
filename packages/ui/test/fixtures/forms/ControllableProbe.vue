<script setup vapor lang="ts">
// useControllable given a real defineEmits overload set, with update:modelValue
// neither first nor last, in a Vapor component: it must type-check (vue-tsc runs
// over this file) and emit through v-model at run time.
import { expectTypeOf } from 'vitest'
import type { Ref } from 'vue'
import { useControllable } from '../../../src/headless/useControllable.ts'

const props = withDefaults(defineProps<{ modelValue?: number; open?: boolean }>(), { modelValue: undefined, open: undefined })
const emit = defineEmits<{
  change: [event: Event]
  'update:modelValue': [value: number]
  'update:open': [open: boolean]
  close: []
}>()

const count = useControllable(props, 'modelValue', emit, 0)
const open = useControllable(props, 'open', emit, false)
expectTypeOf(count).toEqualTypeOf<Ref<number>>()
expectTypeOf(open).toEqualTypeOf<Ref<boolean>>()
// A default of the wrong type, or a key with no update event, is refused.
// @ts-expect-error the default must be a number
useControllable(props, 'modelValue', emit, 'zero')
// @ts-expect-error there is no update:other event
useControllable({ other: undefined as number | undefined }, 'other', emit, 0)
defineExpose({ bump: () => count.value++, toggle: () => (open.value = !open.value) })
</script>

<template><output>{{ count }} {{ open }}</output></template>
