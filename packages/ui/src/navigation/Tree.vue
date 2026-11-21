<script setup vapor lang="ts">
// A hierarchy of nodes that open and close, such as files in folders
// (WAI-ARIA APG, Tree View). It draws the visible nodes as one flat list of
// treeitems, each stating its aria-level, aria-setsize and aria-posinset,
// and aria-expanded when it has children. The tree is one tab stop. Down
// and Up move over the visible nodes, Home and End to the first and last;
// Right opens a closed node, then moves to its first child; Left closes an
// open node, else moves to its parent (both mirrored right to left); `*`
// opens every sibling of the focused node; typing jumps to a node by its
// label. Enter or a press chooses a node and opens or closes it; Space
// chooses. With `multiple` (aria-multiselectable): Space and a press toggle
// a node, Shift with Down or Up moves and toggles, Ctrl+A chooses every
// visible node. Disabled nodes are reached but never chosen, opened or
// closed. Past 200 visible nodes it renders only those in view (a fixed
// row height, scrolling inside the tree). On touch each row is at least
// --weft-hit-min tall.
import { computed, nextTick, ref, useTemplateRef } from 'vue'
import Icon from '../basics/Icon.vue'
import { useWeft } from '../adaptive/context.ts'
import { focus, isRtl } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useTypeahead } from '../headless/useTypeahead.ts'
import { useVirtualList } from '../headless/useVirtualList.ts'
import type { Size } from '../types.ts'
import { defaultOpen, flatten, type TreeItem, type TreeRow } from './tree.ts'
import '../styles/base.css'
import '../styles/tree.css'

const props = withDefaults(
  defineProps<{
    /** The nodes at the top: `{ label, value, icon, children, defaultExpanded, disabled }`. */
    items?: TreeItem[]
    /** The chosen node's key (null when none is), or the chosen keys with `multiple`, bound with `v-model`. A node's key is its `value`, else its labels from the top joined with `/`. */
    modelValue?: string | string[] | null
    /** The chosen node or nodes when `v-model` is not bound. */
    defaultValue?: string | string[] | null
    /** Several nodes can be chosen. */
    multiple?: boolean
    /** Nodes can be chosen at all; false leaves only opening and closing (no aria-selected). */
    selectable?: boolean
    /** The keys of the open nodes, bound with `v-model:expanded`. */
    expanded?: string[]
    /** The open nodes when `v-model:expanded` is not bound; default those marked `defaultExpanded`. */
    defaultExpanded?: string[]
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** Nothing can be chosen, opened or closed. */
    disabled?: boolean
  }>(),
  { items: () => [], modelValue: undefined, defaultValue: undefined, selectable: true, expanded: undefined, defaultExpanded: undefined, size: 'md' },
)

const emit = defineEmits<{
  /** The chosen node or nodes changed. */
  'update:modelValue': [value: string | string[] | null]
  /** The open nodes changed. */
  'update:expanded': [keys: string[]]
  /** A node was chosen (or, with `multiple`, toggled). */
  select: [item: TreeItem]
  /** A node opened (true) or closed (false). */
  toggle: [item: TreeItem, open: boolean]
}>()

defineSlots<{
  /** A node's label, in place of `label`. */
  item?: (props: { item: TreeItem; level: number; expanded: boolean; selected: boolean }) => unknown
}>()

const weft = useWeft()
const id = useId('tree')
const root = useTemplateRef<HTMLElement>('root')
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? (props.multiple ? [] : null))
const open = useControllable(props, 'expanded', emit, () => props.defaultExpanded ?? defaultOpen(props.items))
const openSet = computed(() => new Set(open.value))
const rows = computed(() => flatten(props.items, openSet.value))
const chosen = computed(() => {
  const v = model.value
  return new Set(v == null ? [] : Array.isArray(v) ? v : [v])
})

/** Rows of a fixed height when windowed: the control height, or the hit size on touch. */
const ROW = { sm: 28, md: 32, lg: 40 } as const
// useWeft() reports the pointer until mounted, so a touch page hydrates the
// pointer height and the touch height follows.
const rowHeight = computed(() => (weft.input.value === 'touch' ? 44 : ROW[props.size]))
const focusKey = ref<string | null>(null)
const focusIndex = computed(() => {
  const at = focusKey.value === null ? -1 : rows.value.findIndex((r) => r.key === focusKey.value)
  if (at >= 0) return at
  const picked = rows.value.findIndex((r) => chosen.value.has(r.key))
  return picked >= 0 ? picked : 0
})
const list = useVirtualList(root, {
  count: () => rows.value.length,
  itemHeight: () => rowHeight.value,
  enabled: () => rows.value.length > 200,
  pinned: () => focusIndex.value,
})

const off = (row: TreeRow | undefined) => !row || props.disabled || !!row.item.disabled

/** Focus the row at `index` once it is rendered, scrolling it into view. */
async function focusAt(index: number) {
  const row = rows.value[index]
  if (!row) return
  focusKey.value = row.key
  list.scrollTo(index)
  await nextTick()
  focus(root.value?.querySelector<HTMLElement>(`[data-index="${index}"]`))
}

function setOpen(row: TreeRow, on: boolean) {
  if (off(row) || !row.branch || row.open === on) return
  open.value = on ? [...open.value, row.key] : open.value.filter((k) => k !== row.key)
  emit('toggle', row.item, on)
}

function choose(row: TreeRow, mode: 'set' | 'toggle') {
  if (off(row) || !props.selectable) return
  if (props.multiple) {
    const now = [...chosen.value]
    const has = chosen.value.has(row.key)
    model.value = mode === 'toggle' && has ? now.filter((k) => k !== row.key) : has ? now : [...now, row.key]
  } else model.value = row.key
  emit('select', row.item)
}

function activate(row: TreeRow) {
  choose(row, props.multiple ? 'toggle' : 'set')
  if (row.branch) setOpen(row, !row.open)
}

function onClick(index: number, event: MouseEvent) {
  const row = rows.value[index]
  if (!row) return
  focusKey.value = row.key
  focus(event.currentTarget as HTMLElement)
  activate(row)
}

const typeahead = useTypeahead(
  () => rows.value.map((r) => r.item.label),
  (i) => void focusAt(i),
)

function onKeydown(event: KeyboardEvent) {
  if (event.defaultPrevented || event.altKey || event.metaKey) return
  const at = focusIndex.value
  const row = rows.value[at]
  if (!row) return
  const last = rows.value.length - 1
  const rtl = isRtl(root.value)
  const forward = rtl ? 'ArrowLeft' : 'ArrowRight'
  const back = rtl ? 'ArrowRight' : 'ArrowLeft'
  const handled = () => event.preventDefault()
  if (event.ctrlKey) {
    if (event.key.toLowerCase() === 'a' && props.multiple && props.selectable && !props.disabled) {
      handled()
      model.value = rows.value.filter((r) => !r.item.disabled).map((r) => r.key)
    }
    return
  }
  switch (event.key) {
    case 'ArrowDown':
    case 'ArrowUp': {
      handled()
      const to = event.key === 'ArrowDown' ? Math.min(at + 1, last) : Math.max(at - 1, 0)
      void focusAt(to)
      const target = rows.value[to]
      if (event.shiftKey && props.multiple && target && to !== at) choose(target, 'toggle')
      return
    }
    case 'Home':
      handled()
      void focusAt(0)
      return
    case 'End':
      handled()
      void focusAt(last)
      return
    case forward:
      handled()
      if (row.branch && !row.open) setOpen(row, true)
      else if (row.open) void focusAt(at + 1)
      return
    case back:
      handled()
      if (row.open) setOpen(row, false)
      else if (row.parent >= 0) void focusAt(row.parent)
      return
    case 'Enter':
      handled()
      activate(row)
      return
    case '*': {
      handled()
      const siblings = rows.value.filter((r) => r.parent === row.parent && r.branch && !r.open && !off(r))
      if (!siblings.length) return
      open.value = [...open.value, ...siblings.map((r) => r.key)]
      for (const r of siblings) emit('toggle', r.item, true)
      return
    }
  }
  if (typeahead.onKeydown(event, at)) {
    handled()
    return
  }
  if (event.key === ' ') {
    handled()
    choose(row, props.multiple ? 'toggle' : 'set')
  }
}

function onFocusin(event: FocusEvent) {
  const el = (event.target as Element).closest<HTMLElement>('[role="treeitem"]')
  const row = el ? rows.value[Number(el.dataset.index)] : undefined
  if (row) focusKey.value = row.key
}

const rowStyle = (row: TreeRow, index: number) =>
  list.virtual.value
    ? { '--weft-tree-level': String(row.level - 1), top: `${list.offset(index)}px` }
    : { '--weft-tree-level': String(row.level - 1) }
</script>

<template>
  <div
    ref="root"
    class="weft-tree"
    role="tree"
    :aria-multiselectable="multiple ? 'true' : undefined"
    :aria-disabled="disabled ? 'true' : undefined"
    :data-size="size"
    :data-virtual="list.virtual.value ? 'true' : undefined"
    :style="{ '--weft-tree-row': `${rowHeight}px` }"
    @keydown="onKeydown"
    @focusin="onFocusin"
  >
    <div v-if="list.virtual.value" class="weft-tree__spacer" aria-hidden="true" :style="{ blockSize: `${list.total.value}px` }"></div>
    <div
      v-for="i in list.indices.value"
      :id="`${id}-${i}`"
      :key="rows[i]!.key"
      class="weft-tree__item"
      role="treeitem"
      :data-index="i"
      :tabindex="i === focusIndex ? 0 : -1"
      :aria-level="rows[i]!.level"
      :aria-setsize="rows[i]!.setsize"
      :aria-posinset="rows[i]!.posinset"
      :aria-expanded="rows[i]!.branch ? (rows[i]!.open ? 'true' : 'false') : undefined"
      :aria-selected="selectable ? (chosen.has(rows[i]!.key) ? 'true' : 'false') : undefined"
      :aria-disabled="rows[i]!.item.disabled ? 'true' : undefined"
      :style="rowStyle(rows[i]!, i)"
      @click="onClick(i, $event)"
    >
      <span class="weft-tree__toggle" aria-hidden="true">
        <svg v-if="rows[i]!.branch" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6" /></svg>
      </span>
      <Icon v-if="rows[i]!.item.icon" :name="rows[i]!.item.icon!" class="weft-tree__icon" />
      <span class="weft-tree__label"><slot name="item" :item="rows[i]!.item" :level="rows[i]!.level" :expanded="rows[i]!.open" :selected="chosen.has(rows[i]!.key)">{{ rows[i]!.item.label }}</slot></span>
    </div>
  </div>
</template>
