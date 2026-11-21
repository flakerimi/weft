<script setup vapor lang="ts">
// A button that opens a list to choose one value from, or several (WAI-ARIA
// APG Select-Only Combobox). Focus stays on the button while the list is
// open, aria-activedescendant naming the active option. Closed: ArrowDown,
// ArrowUp, Enter and Space open it on the chosen option (else the first),
// Home and End on the first and the last, typing on the option whose label
// starts so. Open: the arrows, Home, End, Page Up and Page Down move
// (skipping disabled options), typing jumps, Enter and Space choose and
// close (with multiple they toggle and it stays open), Tab chooses and
// moves on, Escape closes without choosing. On pointer the list floats
// under the button; on touch it is a modal bottom sheet with a search field
// at the top, which takes focus and filters. Inside a FormField it takes the
// field's id, label, description and error; with a name it submits its
// value in hidden inputs. Past 200 options only the rows in view are drawn.
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue'
import Icon from '../basics/Icon.vue'
import Spinner from '../basics/Spinner.vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { hideField, useControlAttrs, useField } from '../forms/field.ts'
import Popover from '../overlays/Popover.vue'
import type { Placement } from '../headless/useFloating.ts'
import { formValue, itemOf, labelOfValue, type By, type SelectItem, type SelectItems } from './items.ts'
import { listProps, provideListbox, useListbox } from './listbox.ts'
import Listbox from './Listbox.vue'
import { labelText, usePopup, whenFocusLeaves } from './popup.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/select.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The options: values, or objects with `label`, `value`, `description`, `icon`, `disabled`, `type: 'label' | 'separator'`; or groups of them. */
    items?: SelectItems
    /** The chosen value, bound with `v-model`; an array with `multiple`. */
    modelValue?: unknown
    /** The value it starts with when `v-model` is not bound. */
    defaultValue?: unknown
    /** Choose several: `v-model` is an array, choosing toggles, and the list stays open. */
    multiple?: boolean
    /** Read an object's value at this key; an empty string makes the object itself the value. */
    valueKey?: string
    /** Read an object's label at this key. */
    labelKey?: string
    /** Read an object's description at this key. */
    descriptionKey?: string
    /** Compare values by this key, or with this function; by default in depth. */
    by?: By
    /** Shown while nothing is chosen. */
    placeholder?: string
    /** The search field's placeholder, in the touch sheet. */
    searchPlaceholder?: string
    /** `sm`, `md` or `lg`; a FieldGroup's or FormField's size applies when not set. md is 44 px tall on touch. */
    size?: Size
    /** A Lucide icon before the value. The leading slot replaces it. */
    icon?: string
    /** A Lucide icon in place of the chevron. */
    trailingIcon?: string
    /** A Lucide icon marking chosen options, in place of the drawn check. */
    selectedIcon?: string
    /** Work in progress: a spinner in place of the chevron, and aria-busy. */
    loading?: boolean
    /** Cannot be opened or changed. */
    disabled?: boolean
    /** A value is required (aria-required). */
    required?: boolean
    /** The form field name its value submits under; a FormField's name applies when not set. */
    name?: string
    /** The button's id; inside a FormField the field's id is used, so its label points here. */
    id?: string
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** Where the list floats on pointer (start and end follow the text direction). */
    placement?: Placement
    /** The arrows wrap round at the ends. */
    loop?: boolean
    /** Draw only the rows in view; by default past 200 options. */
    virtualize?: boolean
  }>(),
  {
    modelValue: undefined,
    defaultValue: undefined,
    valueKey: 'value',
    labelKey: 'label',
    descriptionKey: 'description',
    open: undefined,
    defaultOpen: false,
    placement: 'bottom-start',
    virtualize: undefined,
  },
)

const emit = defineEmits<{
  /** Chosen: the value, or the array of values with `multiple`. */
  'update:modelValue': [value: unknown]
  /** Opened or closed. */
  'update:open': [open: boolean]
  /** The value changed by a choice. */
  change: [value: unknown]
  /** Focus left the button and its list. */
  blur: [event: FocusEvent]
  /** The button took focus. */
  focus: [event: FocusEvent]
}>()

defineSlots<{
  /** The value as shown on the button. */
  default?: (props: { modelValue: unknown; item: SelectItem | undefined; open: boolean }) => unknown
  /** Before the value, in place of `icon`. */
  leading?: (props: { modelValue: unknown; open: boolean }) => unknown
  /** After the value, in place of the chevron. */
  trailing?: (props: { open: boolean }) => unknown
  /** An option's whole content. */
  item?: (props: { item: SelectItem; index: number; selected: boolean }) => unknown
  /** Before an option's label, in place of its icon. */
  'item-leading'?: (props: { item: SelectItem; index: number }) => unknown
  /** An option's label. */
  'item-label'?: (props: { item: SelectItem; index: number }) => unknown
  /** After an option's label, before the check. */
  'item-trailing'?: (props: { item: SelectItem; index: number; selected: boolean }) => unknown
  /** Shown when there is no option, or none matches the search. */
  empty?: (props: { searchTerm: string }) => unknown
}>()

const f = useField(props)
// The search field in the sheet is not the field's control.
hideField()
const { own, box, describedBy } = useControlAttrs(f)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? (props.multiple ? [] : undefined))
const popup = usePopup(props, emit, () => f.disabled.value)
const { open, sheet } = popup
const search = ref('')
const listId = useId('select-list')
const button = useTemplateRef<HTMLButtonElement>('button')
const valueKey = () => props.valueKey || undefined

const list = useListbox(
  listProps(props, { valueKey, disabled: () => f.disabled.value, size: () => f.size.value }),
  model,
  {
    id: listId,
    search: () => (sheet.value ? search.value : ''),
    touch: () => sheet.value,
    label: () => {
      if (labelText(f.field)) return { 'aria-labelledby': f.field!.labelId }
      return { 'aria-label': (own.value['aria-label'] as string | undefined) ?? props.placeholder }
    },
    onChoose: () => {
      emit('change', model.value)
      f.input()
      if (!props.multiple) popup.hide()
    },
  },
)
provideListbox(list)

const values = computed<unknown[]>(() => {
  const v = model.value
  return props.multiple ? (Array.isArray(v) ? v : []) : v === undefined || v === null || v === '' ? [] : [v]
})
const opts = () => ({ valueKey: valueKey(), labelKey: props.labelKey, by: props.by })
const display = computed(() => values.value.map((v) => labelOfValue(props.items, v, opts())).filter(Boolean).join(', '))
const selectedItem = computed(() => (props.multiple ? undefined : itemOf(props.items, model.value, opts())))
const sheetTitle = computed(() => (open.value && sheet.value ? (labelText(f.field) ?? (own.value['aria-label'] as string | undefined) ?? props.placeholder) : undefined))

/** Where the active option starts when the list opens. */
let start: 'selected' | 'first' | 'last' = 'selected'
/** A key typed on the closed button, for the list to jump with once it is open. */
let typed: string | null = null
function show(at: typeof start) {
  start = at
  popup.show()
}
watch(open, (on) => {
  if (on) {
    list.highlight(start)
    start = 'selected'
    // A key typed on the closed button jumps once the list is there.
    if (typed) list.keydown(new KeyboardEvent('keydown', { key: typed }), { edges: true, typeahead: true, space: false })
    typed = null
  } else {
    search.value = ''
    list.resetTypeahead()
  }
})
whenFocusLeaves(open, listId, () => button.value, () => f.blur())
// Touch to pointer while open: the search field goes, so focus comes back
// to the button, which carries the active option again. Once the render
// that removed the field is done and the sheet's focus trap has let go:
// before, focus moving out would read to the Popover as focus leaving it,
// and to the trap as focus to bring back.
watch(sheet, (touch) => {
  if (touch || !open.value) return
  search.value = ''
  void nextTick(() => open.value && button.value?.focus({ preventScroll: true }))
})

function onKeydown(event: KeyboardEvent) {
  if (f.disabled.value) return
  if (!open.value) {
    const k = event.key
    if (k === 'ArrowDown' || k === 'ArrowUp' || k === 'Enter' || k === ' ') show('selected')
    else if (k === 'Home') show('first')
    else if (k === 'End') show('last')
    else if (k.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
      typed = k
      show('selected')
    } else return
    event.preventDefault()
    return
  }
  if (event.key === 'Tab') {
    const o = list.activeOption.value
    if (o && !props.multiple && !sheet.value) list.choose(o, event)
    popup.hide()
    return
  }
  if (event.key === 'ArrowUp' && event.altKey) {
    event.preventDefault()
    const o = list.activeOption.value
    if (o) list.choose(o, event)
    popup.hide()
    return
  }
  if (list.keydown(event, { edges: true, typeahead: true, space: true })) event.preventDefault()
}
function onSearchKeydown(event: KeyboardEvent) {
  if (list.keydown(event, { edges: false, typeahead: false, space: false })) event.preventDefault()
}
function onSearchInput(event: Event) {
  search.value = (event.target as HTMLInputElement).value
}
/** Focus leaving the button for somewhere other than its list. */
function onBlur(event: FocusEvent) {
  const to = event.relatedTarget as Node | null
  const panel = document.getElementById(listId)?.closest('.weft-popover')
  if (to && panel?.contains(to)) return
  if (!sheet.value) popup.hide()
  f.blur()
  emit('blur', event)
}
function onOpenChange(on: boolean) {
  open.value = on
}
/** A press in the floating list's padding keeps focus on the button. */
function onPanelDown(event: MouseEvent) {
  if (!sheet.value) event.preventDefault()
}
</script>

<template>
  <span class="weft-select">
    <button
      ref="button"
      type="button"
      class="weft-input weft-select__trigger"
      role="combobox"
      aria-haspopup="listbox"
      v-bind="{ ...box, ...own }"
      :id="f.id.value"
      :disabled="f.disabled.value || undefined"
      :data-size="f.size.value"
      :data-invalid="f.invalid.value ? 'true' : undefined"
      :data-disabled="f.disabled.value ? 'true' : undefined"
      :data-open="open ? 'true' : undefined"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-controls="open ? listId : undefined"
      :aria-activedescendant="open && !sheet ? list.activeId.value : undefined"
      :aria-describedby="describedBy()"
      :aria-invalid="f.aria.value['aria-invalid']"
      :aria-required="required || f.aria.value['aria-required'] ? 'true' : undefined"
      :aria-busy="loading ? 'true' : undefined"
      @click="popup.press()"
      @keydown="onKeydown"
      @blur="onBlur"
      @focus="emit('focus', $event)"
    >
      <span v-if="$slots.leading || icon" class="weft-input__leading">
        <slot name="leading" :model-value="model" :open="open"><Icon v-if="icon" :name="icon" /></slot>
      </span>
      <span class="weft-select__value" :data-placeholder="display ? undefined : 'true'">
        <slot :model-value="model" :item="selectedItem" :open="open">{{ display || placeholder || ' ' }}</slot>
      </span>
      <span class="weft-input__trailing">
        <Spinner v-if="loading" />
        <slot v-else name="trailing" :open="open">
          <Icon v-if="trailingIcon" :name="trailingIcon" />
          <svg v-else class="weft-select__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
        </slot>
      </span>
    </button>
    <input v-for="(v, i) in f.name.value ? values : []" :key="i" type="hidden" :name="f.name.value" :value="formValue(v)" />
    <Popover
      :open="open"
      :anchor="button ?? undefined"
      :auto-focus="false"
      :placement="placement"
      :offset="4"
      :title="sheetTitle"
      :role="sheet ? 'dialog' : 'presentation'"
      class="weft-select__popup"
      @mousedown="onPanelDown"
      @update:open="onOpenChange"
    >
      <template #content>
        <span v-if="sheet" class="weft-input weft-select__search" :data-size="f.size.value">
          <svg class="weft-select__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" /></svg>
          <input
            class="weft-input__field"
            type="text"
            role="combobox"
            autocomplete="off"
            aria-expanded="true"
            aria-autocomplete="list"
            :aria-label="searchPlaceholder ?? 'Search'"
            :placeholder="searchPlaceholder ?? 'Search…'"
            :value="search"
            :aria-controls="listId"
            :aria-activedescendant="list.activeId.value"
            @input="onSearchInput"
            @keydown="onSearchKeydown"
          />
        </span>
        <Listbox :selected-icon="selectedIcon">
          <template v-if="$slots.item" #item="p"><slot name="item" v-bind="p" /></template>
          <template v-if="$slots['item-leading']" #item-leading="p"><slot name="item-leading" v-bind="p" /></template>
          <template v-if="$slots['item-label']" #item-label="p"><slot name="item-label" v-bind="p" /></template>
          <template v-if="$slots['item-trailing']" #item-trailing="p"><slot name="item-trailing" v-bind="p" /></template>
          <template v-if="$slots.empty" #empty="p"><slot name="empty" v-bind="p" /></template>
        </Listbox>
      </template>
    </Popover>
  </span>
</template>
