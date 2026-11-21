<script setup vapor lang="ts">
// A list of options to choose one or several from, always shown (WAI-ARIA
// APG Listbox). Focus stays on the list, or on its filter field when it has
// one, and aria-activedescendant names the active option, so the arrows,
// Home, End, Page Up and Page Down, and typing a label move without moving
// focus; Enter and Space choose (Space toggles with multiple). Disabled
// options are skipped. Past 200 options only the rows in view are drawn.
// Select, Combobox and CommandPalette draw their lists with it: they own the
// list's state (useListbox) and this draws theirs.
import { computed, onMounted, onScopeDispose, useTemplateRef, watch } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import Icon from '../basics/Icon.vue'
import Spinner from '../basics/Spinner.vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useControlAttrs, useField } from '../forms/field.ts'
import { formValue, type By, type SelectItem, type SelectItems } from './items.ts'
import { injectListbox, useListbox, type Choice, type CreateItem, type Row, type RowGroup } from './listbox.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/listbox.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The options: values, or objects with `label`, `value`, `description`, `icon`, `disabled`, `type: 'label' | 'separator'`; or groups of them. */
    items?: SelectItems
    /** The chosen value, bound with `v-model`; an array with `multiple`. */
    modelValue?: unknown
    /** The value it starts with when `v-model` is not bound. */
    defaultValue?: unknown
    /** Choose several: `v-model` is an array, and choosing toggles. */
    multiple?: boolean
    /** Read an object's value at this key; without it the object itself is the value. */
    valueKey?: string
    /** Read an object's label at this key. */
    labelKey?: string
    /** Read an object's description at this key. */
    descriptionKey?: string
    /** Compare values by this key, or with this function; by default in depth. */
    by?: By
    /** A search field above the list that filters it. */
    filter?: boolean
    /** The filter's text, bound with `v-model:search-term`. */
    searchTerm?: string
    /** The filter's placeholder. */
    placeholder?: string
    /** Search these keys of each item; by default its label. */
    filterFields?: string[]
    /** Show the items as given while searching: the app filters them (async search). */
    ignoreFilter?: boolean
    /** Offer the search as a new item (emits `create`): when nothing matches, or `'always'`; `{ position: 'top' }` puts it first. */
    createItem?: CreateItem
    /** Items are on their way: `aria-busy`, and a spinner in place of an empty list. */
    loading?: boolean
    /** Nothing can be chosen, and it takes no focus. */
    disabled?: boolean
    /** `sm`, `md` or `lg`; a FieldGroup's or FormField's size applies when not set. */
    size?: Size
    /** The arrows wrap round at the ends. */
    loop?: boolean
    /** Draw only the rows in view: by default past 200 options; an object sets `estimateSize` (px, the row height to start from; rows then grow to the tallest measured) and `overscan`. */
    virtualize?: boolean | { estimateSize?: number; overscan?: number }
    /** With multiple: `toggle`, choosing toggles; `replace`, the choice follows the arrows and a press (Ctrl toggles, Shift extends). */
    selectionBehavior?: 'toggle' | 'replace'
    /** A Lucide icon marking chosen options, in place of the drawn check. */
    selectedIcon?: string
    /** The list's id; inside a FormField the field's id is used. */
    id?: string
    /** The form field name its value submits under; a FormField's name applies when not set. */
    name?: string
    /** A value is required (aria-required). */
    required?: boolean
  }>(),
  {
    modelValue: undefined,
    defaultValue: undefined,
    searchTerm: undefined,
    labelKey: 'label',
    descriptionKey: 'description',
    virtualize: undefined,
    selectionBehavior: 'toggle',
  },
)

const emit = defineEmits<{
  /** Chosen: the value, or the array of values with `multiple`. */
  'update:modelValue': [value: unknown]
  /** The filter's text changed. */
  'update:searchTerm': [term: string]
  /** The create option was chosen, with the search as the new item's label. */
  create: [term: string]
  /** The active option changed: its item, or undefined when none is active. */
  highlight: [item: SelectItem | undefined]
}>()

defineSlots<{
  /** An option's whole content. */
  item?: (props: { item: SelectItem; index: number; selected: boolean }) => unknown
  /** Before an option's label, in place of its icon. */
  'item-leading'?: (props: { item: SelectItem; index: number }) => unknown
  /** An option's label. */
  'item-label'?: (props: { item: SelectItem; index: number }) => unknown
  /** After an option's label, before the check. */
  'item-trailing'?: (props: { item: SelectItem; index: number; selected: boolean }) => unknown
  /** The create option's label. */
  'create-item-label'?: (props: { item: string }) => unknown
  /** Shown when no option is (or matches): by default "No results". */
  empty?: (props: { searchTerm: string }) => unknown
  /** Shown in place of the list when its items failed to load (a Combobox's `load`): by default "Couldn’t load results". */
  error?: (props: { error: unknown }) => unknown
}>()

const host = injectListbox()
const weft = useWeft()
const touch = () => weft.input.value === 'touch'
const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)
// A list is not a labelable element: it names itself by the field's label.
// With a filter, the filter field is the labelled control.
if (!host && !props.filter) f.field?.namedByLabel()

const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? (props.multiple ? [] : undefined))
const term = useControllable(props, 'searchTerm', emit, '')
const ownId = useId('listbox')
const list =
  host ??
  useListbox(props, model, {
    id: props.filter ? `${f.id.value ?? ownId}-list` : (f.id.value ?? ownId),
    search: () => (props.filter ? term.value : ''),
    touch,
    label: () => {
      if (f.field) return { 'aria-labelledby': f.field.labelId }
      const label = own.value['aria-label'] as string | undefined
      return label ? { 'aria-label': label } : {}
    },
    onCreate: (t) => emit('create', t),
    onHighlight: (c) => emit('highlight', c?.kind === 'option' ? c.item : undefined),
  })

const listEl = useTemplateRef<HTMLElement>('listEl')
const filterEl = useTemplateRef<HTMLInputElement>('filterEl')
watch(listEl, (el) => (list.scroller.value = el ?? null), { immediate: true, flush: 'post' })

const disabled = computed(() => (host ? !!list.props.disabled : f.disabled.value))
const virtual = computed(() => list.virtual.virtual.value)
/** The element that has focus and so carries aria-activedescendant, when this list owns it. */
const focusesList = computed(() => (host ? list.ownsFocus() : !props.filter))
/** The attributes and description of a list that is its own control (not its owner's list). */
const ownControl = computed(() => focusesList.value && !host)
/** The rows drawn now: those in view, when virtual, and in each group drawn its heading, which labels it. */
const shown = computed(() => new Set(virtual.value ? list.virtual.indices.value : []))
const rowsOf = (g: RowGroup) => {
  if (!virtual.value) return g.rows
  const rows = g.rows.filter((r) => shown.value.has(r.row))
  const heading = g.rows[0]
  if (rows.length && heading?.kind === 'heading' && rows[0] !== heading) rows.unshift(heading)
  return rows
}
const empty = computed(() => list.options.value.length === 0)
const failed = computed(() => list.props.error !== undefined && list.props.error !== null)
const busy = computed(() => !!list.props.loading && !failed.value)
const values = computed(() => {
  if (host || !f.name.value) return []
  const v = model.value
  return (props.multiple ? (Array.isArray(v) ? v : []) : v === undefined || v === null ? [] : [v]).map(formValue)
})

const optionOf = (event: Event) => {
  const el = (event.target as Element).closest<HTMLElement>('[role="option"]')
  return el ? list.options.value.find((o) => o.id === el.id) : undefined
}
const focusOwner = () => (props.filter ? filterEl.value : listEl.value)

function onKeydown(event: KeyboardEvent) {
  if (disabled.value) return
  if (list.keydown(event, { edges: true, typeahead: true, space: true, range: true })) event.preventDefault()
}
function onFilterKeydown(event: KeyboardEvent) {
  if (disabled.value) return
  if (list.keydown(event, { edges: false, typeahead: false, space: false })) event.preventDefault()
}
function onFocus() {
  if (list.active.value < 0) list.highlight('selected')
}
function onFilterInput(event: Event) {
  term.value = (event.target as HTMLInputElement).value
  f.input()
}
function onBlur(event: FocusEvent) {
  const to = event.relatedTarget as Node | null
  if (!to || !(event.currentTarget as Element).closest('.weft-listbox')?.contains(to)) f.blur()
}
// A press on an option keeps focus where it is (the list, its filter, or
// the owner's field), and focuses the list when it has none yet.
function onMousedown(event: MouseEvent) {
  event.preventDefault()
  if (!host && !disabled.value) focusOwner()?.focus({ preventScroll: true })
}
function onPointermove(event: PointerEvent) {
  const o = optionOf(event)
  if (o && !o.disabled && o.id !== list.activeId.value) list.highlight(o.index, { scroll: false })
}
function onClick(event: MouseEvent) {
  const o = optionOf(event)
  if (!o || disabled.value) return
  if (list.activeId.value !== o.id) list.highlight(o.index, { scroll: false })
  list.press(o, event)
}

// Long lists: rows start at an estimated height and grow to the tallest one
// drawn (a larger font, a taller item slot), so none overlaps the next.
function measure() {
  const el = listEl.value
  if (!virtual.value || !el) return
  let tallest = 0
  for (const row of el.querySelectorAll<HTMLElement>('.weft-listbox__option, .weft-listbox__heading')) {
    tallest = Math.max(tallest, row.getBoundingClientRect().height)
  }
  list.measure(tallest)
}
watch(() => [virtual.value && shown.value, list.rowHeight.value], measure, { flush: 'post' })
onMounted(measure)
// Rows can get smaller as well as taller (a smaller font, a wider list
// wrapping less): when the list's width or the page's fonts change, they go
// back to the estimate and are measured again.
let width = 0
let observer: ResizeObserver | undefined
const remeasure = () => virtual.value && list.remeasure()
watch(
  listEl,
  (el) => {
    observer?.disconnect()
    observer = undefined
    if (!el || typeof ResizeObserver === 'undefined') return
    width = el.getBoundingClientRect().width
    observer = new ResizeObserver((entries) => {
      const w = entries[0]?.contentRect.width ?? width
      if (w !== width) remeasure()
      width = w
    })
    observer.observe(el)
  },
  { immediate: true, flush: 'post' },
)
onMounted(() => document.fonts?.addEventListener?.('loadingdone', remeasure))
onScopeDispose(() => {
  observer?.disconnect()
  if (typeof document !== 'undefined') document.fonts?.removeEventListener?.('loadingdone', remeasure)
})

const isOption = (r: Row): r is Choice => r.kind === 'option' || r.kind === 'create'
const selected = (r: Row) => isOption(r) && list.isSelected(r)
const rowStyle = (r: Row) => (virtual.value ? { top: `${list.virtual.offset(r.row)}px` } : undefined)

defineExpose({
  /** Focus the list, or its filter field. */
  focus: () => focusOwner()?.focus(),
})
</script>

<template>
  <div
    class="weft-listbox"
    v-bind="host ? undefined : box"
    :data-size="host ? list.props.size : f.size.value"
    :data-hosted="host ? 'true' : undefined"
    :data-touch="list.touch() ? 'true' : undefined"
    :data-invalid="!host && f.invalid.value ? 'true' : undefined"
    :data-disabled="disabled ? 'true' : undefined"
  >
    <span v-if="!host && filter" class="weft-input weft-listbox__filter" :data-size="f.size.value">
      <svg class="weft-listbox__glyph weft-listbox__search" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" /></svg>
      <input
        ref="filterEl"
        class="weft-input__field"
        type="text"
        role="combobox"
        autocomplete="off"
        aria-expanded="true"
        aria-autocomplete="list"
        v-bind="own"
        :id="f.id.value ?? ownId"
        :value="term"
        :placeholder="placeholder ?? 'Search…'"
        :disabled="disabled || undefined"
        :aria-controls="list.id"
        :aria-activedescendant="list.activeId.value"
        :aria-describedby="describedBy()"
        :aria-invalid="f.aria.value['aria-invalid']"
        :aria-required="required || f.aria.value['aria-required'] ? 'true' : undefined"
        @input="onFilterInput"
        @keydown="onFilterKeydown"
        @blur="onBlur"
      />
    </span>
    <div
      ref="listEl"
      class="weft-listbox__list"
      role="listbox"
      v-bind="ownControl ? own : undefined"
      :id="list.id"
      :tabindex="focusesList ? (disabled ? -1 : 0) : undefined"
      :aria-activedescendant="focusesList ? list.activeId.value : undefined"
      :aria-labelledby="list.label()['aria-labelledby']"
      :aria-label="list.label()['aria-label']"
      :aria-multiselectable="list.props.multiple ? 'true' : undefined"
      :aria-busy="busy ? 'true' : undefined"
      :aria-disabled="disabled ? 'true' : undefined"
      :aria-describedby="ownControl ? describedBy() : undefined"
      :aria-invalid="ownControl ? f.aria.value['aria-invalid'] : undefined"
      :aria-required="ownControl && (required || f.aria.value['aria-required']) ? 'true' : undefined"
      :data-virtual="virtual ? 'true' : undefined"
      :style="virtual ? { '--weft-listbox-row': `${list.rowHeight.value}px` } : undefined"
      @keydown="focusesList && onKeydown($event)"
      @focus="focusesList && onFocus()"
      @blur="ownControl && onBlur($event)"
      @mousedown="onMousedown"
      @pointermove="onPointermove"
      @click="onClick"
    >
      <div v-if="virtual" class="weft-listbox__spacer" aria-hidden="true" :style="{ blockSize: `${list.virtual.total.value}px` }"></div>
      <template v-for="g in list.groups.value" :key="g.key">
        <div
          v-if="rowsOf(g).length"
          class="weft-listbox__group"
          :role="g.labelledBy ? 'group' : 'none'"
          :aria-labelledby="g.labelledBy"
        >
          <template v-for="r in rowsOf(g)" :key="r.key">
            <div v-if="r.kind === 'heading'" :id="r.id" class="weft-listbox__heading" role="presentation" :style="rowStyle(r)">{{ r.label }}</div>
            <div v-else-if="r.kind === 'separator'" class="weft-listbox__separator" aria-hidden="true"></div>
            <div
              v-else
              :id="r.id"
              class="weft-listbox__option"
              :class="{ 'weft-listbox__create': r.kind === 'create' }"
              role="option"
              :data-index="r.row"
              :aria-selected="selected(r) ? 'true' : 'false'"
              :aria-disabled="r.disabled ? 'true' : undefined"
              :aria-setsize="virtual ? r.setSize : undefined"
              :aria-posinset="virtual ? r.posInSet : undefined"
              :data-active="list.activeId.value === r.id ? 'true' : undefined"
              :style="rowStyle(r)"
            >
              <slot v-if="r.kind === 'option'" name="item" :item="r.item" :index="r.index" :selected="selected(r)">
                <span v-if="$slots['item-leading'] || r.icon" class="weft-listbox__leading">
                  <slot name="item-leading" :item="r.item" :index="r.index"><Icon v-if="r.icon" :name="r.icon" /></slot>
                </span>
                <span class="weft-listbox__label">
                  <span class="weft-listbox__text"><slot name="item-label" :item="r.item" :index="r.index">{{ r.label }}</slot></span>
                  <span v-if="r.description" class="weft-listbox__description">{{ r.description }}</span>
                </span>
                <span v-if="$slots['item-trailing']" class="weft-listbox__trailing"><slot name="item-trailing" :item="r.item" :index="r.index" :selected="selected(r)" /></span>
                <span v-if="list.checks && selected(r)" class="weft-listbox__check" aria-hidden="true">
                  <Icon v-if="selectedIcon" :name="selectedIcon" />
                  <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
                </span>
              </slot>
              <template v-else>
                <span class="weft-listbox__leading" aria-hidden="true">
                  <svg class="weft-listbox__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14" /><path d="M12 5v14" /></svg>
                </span>
                <span class="weft-listbox__label"><span class="weft-listbox__text"><slot name="create-item-label" :item="r.label">Create “{{ r.label }}”</slot></span></span>
              </template>
            </div>
          </template>
        </div>
      </template>
    </div>
    <div class="weft-listbox__status" role="status">
      <template v-if="failed">
        <slot name="error" :error="list.props.error">Couldn’t load results</slot>
      </template>
      <template v-else-if="empty">
        <template v-if="busy"><Spinner /> <span>Loading…</span></template>
        <slot v-else name="empty" :search-term="list.search()">No results</slot>
      </template>
    </div>
    <input v-for="(v, i) in values" :key="i" type="hidden" :name="f.name.value" :value="v" />
  </div>
</template>
