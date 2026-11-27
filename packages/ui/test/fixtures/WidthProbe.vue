<script setup vapor lang="ts">
import { useTemplateRef } from 'vue'
import { useWidth } from '../../src/adaptive/useWidth.ts'
const box = useTemplateRef<HTMLElement>('box')
const width = useWidth(box)
const widthId = width.id
</script>

<template><div ref="box" class="box" :data-width="width" :data-weft-w="widthId"></div></template>
