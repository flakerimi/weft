<script setup vapor lang="ts">
// Three overlays as a dialog, a menu opened from it and a tooltip opened from
// the menu would use the headless layer. The menu and the tooltip sit outside
// the dialog's element, as overlays rendered into the body do.
import { ref, useTemplateRef } from 'vue'
import { useDismiss } from '../../../src/headless/useDismiss.ts'
import { useFocusTrap } from '../../../src/headless/useFocusTrap.ts'
import { useScrollLock } from '../../../src/headless/useScrollLock.ts'

const dialog = ref(false)
const menu = ref(false)
const tip = ref(false)
const dialogEl = useTemplateRef<HTMLElement>('dialogEl')
const menuEl = useTemplateRef<HTMLElement>('menuEl')
const tipEl = useTemplateRef<HTMLElement>('tipEl')
const menuTrigger = useTemplateRef<HTMLElement>('menuTrigger')

useDismiss(dialogEl, { onDismiss: () => (dialog.value = false) })
useFocusTrap(dialogEl, { active: dialog })
useScrollLock(dialog)
useDismiss(menuEl, { onDismiss: () => (menu.value = false), ignore: () => [menuTrigger.value] })
useScrollLock(menu)
useDismiss(tipEl, { onDismiss: () => (tip.value = false), outside: false })
</script>

<template>
  <button class="open-dialog" @click="dialog = true">open</button>
  <div v-if="dialog" ref="dialogEl" class="dialog" role="dialog">
    <button ref="menuTrigger" class="open-menu" @click="menu = !menu">menu</button>
    <button class="other">other</button>
  </div>
  <div v-if="menu" ref="menuEl" class="menu" role="menu">
    <button class="item" role="menuitem" @focus="tip = true">item</button>
  </div>
  <div v-if="tip" ref="tipEl" class="tip" role="tooltip">tip</div>
</template>
