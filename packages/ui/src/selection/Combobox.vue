<script setup vapor lang="ts">
// A text field that searches a list as you type and chooses from it (WAI-ARIA
// APG Combobox with a listbox popup and list autocomplete): Nuxt UI's
// InputMenu, and with `trigger="button"` its SelectMenu, a button whose list
// has a search field at the top. Typing opens and filters the list, the
// first match active; ArrowDown and ArrowUp open it and move, Page Up and
// Page Down move ten, Enter chooses, Escape closes and then clears the text,
// Alt+ArrowUp closes. Home and End stay with the text. Focus stays in the
// field, aria-activedescendant naming the active option. It can offer to
// create what was typed (`create-item`), choose several (`multiple`: tags in
// the field, Backspace removes the last), bind the text itself (`mode=
// "autocomplete"`), and load its items as you type (`load`, debounced, the
// latest answer kept). On touch the list is a modal bottom sheet with a
// search field at the top; the field itself only opens it. Inside a
// FormField it takes the field's id, label, description and error; with a
// name its value submits in hidden inputs.
import { computed, nextTick, onScopeDispose, ref, shallowRef, useTemplateRef, watch } from 'vue'
import Icon from '../basics/Icon.vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import type { Placement } from '../headless/useFloating.ts'
import { shareField, useControlAttrs, useField } from '../forms/field.ts'
import Input from '../forms/Input.vue'
import Popover from '../overlays/Popover.vue'
import { formValue, itemOf, labelOfValue, sameValue, type By, type SelectItem, type SelectItems } from './items.ts'
import { listProps, provideListbox, useListbox, type CreateItem } from './listbox.ts'
import Listbox from './Listbox.vue'
import { labelText, searchField, usePopup, whenFocusLeaves } from './popup.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/select.css'
import '../styles/tag.css'
import '../styles/combobox.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The options: values, or objects with `label`, `description`, `icon`, `disabled`, `type: 'label' | 'separator'`; or groups of them. */
    items?: SelectItems
    /** The chosen value, bound with `v-model`; an array with `multiple`; the text in autocomplete mode. */
    modelValue?: unknown
    /** The value it starts with when `v-model` is not bound. */
    defaultValue?: unknown
    /** Choose several: `v-model` is an array, shown as tags, and the list stays open. */
    multiple?: boolean
    /** Read an object's value at this key; without it the object itself is the value. */
    valueKey?: string
    /** Read an object's label at this key. */
    labelKey?: string
    /** Read an object's description at this key. */
    descriptionKey?: string
    /** Compare values by this key, or with this function; by default in depth. */
    by?: By
    /** `input`: the field is the search (Nuxt UI's InputMenu). `button`: a button, with the search at the top of the list (its SelectMenu). */
    trigger?: 'input' | 'button'
    /** `combobox`: `v-model` is the chosen item's value. `autocomplete`: it is the text, which the list completes. */
    mode?: 'combobox' | 'autocomplete'
    /** The search, bound with `v-model:search-term`. */
    searchTerm?: string
    /** Shown in the empty field, or on the button while nothing is chosen. */
    placeholder?: string
    /** The search field's placeholder, at the top of the list (a button's, and on touch). */
    searchPlaceholder?: string
    /** Search these keys of each item; by default its label. */
    filterFields?: string[]
    /** Show the items as given while searching: the app filters them (async search with `v-model:search-term`). */
    ignoreFilter?: boolean
    /** Offer what was typed as a new item (emits `create`): when nothing matches, or `'always'`; `{ position: 'top' }` puts it first. */
    createItem?: CreateItem
    /** Load the items for a search (when the list opens, and as you type, after `debounce` ms); the latest answer is shown as given, and the signal aborts a superseded one. */
    load?: (term: string, context: { signal: AbortSignal }) => Promise<SelectItems>
    /** ms of quiet typing before `load` runs. */
    debounce?: number
    /** Items are on their way: a spinner in the field, aria-busy, and "Loading…" in place of an empty list. */
    loading?: boolean
    /** A clear button while there is a value. */
    clear?: boolean
    /** Emptying the text clears the value too. */
    resetModelValueOnClear?: boolean
    /** Leaving the field puts back the chosen label (and a closed list forgets its search). */
    resetSearchTermOnBlur?: boolean
    /** Choosing forgets the search. */
    resetSearchTermOnSelect?: boolean
    /** Open the list when the field takes focus. */
    openOnFocus?: boolean
    /** Open the list on a press in the field. */
    openOnClick?: boolean
    /** `sm`, `md` or `lg`; a FieldGroup's or FormField's size applies when not set. md is 44 px tall on touch. */
    size?: Size
    /** A Lucide icon before the text. The leading slot replaces it. */
    icon?: string
    /** A Lucide icon in place of the chevron. */
    trailingIcon?: string
    /** A Lucide icon marking chosen options, in place of the drawn check. */
    selectedIcon?: string
    /** Cannot be opened or changed. */
    disabled?: boolean
    /** A value is required: aria-required on the field (not the native `required`, which would read the search). */
    required?: boolean
    /** The form field name its value submits under; a FormField's name applies when not set. */
    name?: string
    /** The field's id; inside a FormField the field's id is used, so its label points here. */
    id?: string
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** Where the list floats on pointer (start and end follow the text direction). */
    placement?: Placement
    /** The arrows wrap round at the ends. */
    loop?: boolean
    /** Draw only the rows in view: by default past 200 options; an object sets `estimateSize` (px, the row height to start from) and `overscan`. */
    virtualize?: boolean | { estimateSize?: number; overscan?: number }
  }>(),
  {
    modelValue: undefined,
    defaultValue: undefined,
    trigger: 'input',
    mode: 'combobox',
    searchTerm: undefined,
    labelKey: 'label',
    descriptionKey: 'description',
    debounce: 200,
    resetModelValueOnClear: true,
    resetSearchTermOnBlur: true,
    resetSearchTermOnSelect: true,
    open: undefined,
    defaultOpen: false,
    placement: 'bottom-start',
    virtualize: undefined,
  },
)

const emit = defineEmits<{
  /** Chosen: the value, the array of values with `multiple`, or the text in autocomplete mode. */
  'update:modelValue': [value: unknown]
  /** Opened or closed. */
  'update:open': [open: boolean]
  /** The search changed. */
  'update:searchTerm': [term: string]
  /** The value changed by a choice, a tag's removal or the clear button. */
  change: [value: unknown]
  /** The create option was chosen, with what was typed. */
  create: [term: string]
  /** The clear button was pressed. */
  clear: []
  /** A tag was removed (multiple), with its value. */
  'remove-tag': [value: unknown]
  /** The active option changed: its item, or undefined. */
  highlight: [item: SelectItem | undefined]
  /** `load` failed (not when aborted). */
  error: [error: unknown]
  /** Focus left the field and its list. */
  blur: [event: FocusEvent]
  /** The field took focus. */
  focus: [event: FocusEvent]
}>()

defineSlots<{
  /** The value as shown on the button (`trigger="button"`). */
  default?: (props: { modelValue: unknown; item: SelectItem | undefined; open: boolean }) => unknown
  /** Before the text, in place of `icon`. */
  leading?: () => unknown
  /** After the text, in place of the chevron. */
  trailing?: (props: { open: boolean }) => unknown
  /** A tag's text, with `multiple`. */
  tag?: (props: { item: SelectItem | undefined; value: unknown }) => unknown
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
  /** Shown when there is no option, or none matches the search. */
  empty?: (props: { searchTerm: string }) => unknown
  /** Shown in place of the list when `load` failed, with its error: by default "Couldn’t load results". */
  error?: (props: { error: unknown }) => unknown
}>()

const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)
// The Input inside is the field's control, but its text is the search.
shareField(f.field, { required: () => !!props.required })

const autocomplete = computed(() => props.mode === 'autocomplete')
const buttonMode = computed(() => props.trigger === 'button')
const model = useControllable(props, 'modelValue', emit, () =>
  props.defaultValue ?? (props.multiple ? [] : autocomplete.value ? '' : undefined),
)
const term = useControllable(props, 'searchTerm', emit, '')
const popup = usePopup(props, emit, () => f.disabled.value)
const { open, sheet } = popup
/** The search is typed in a field at the top of the list: a button's, and every touch sheet. */
const searchInPopup = computed(() => buttonMode.value || sheet.value)
/** In the field: the user has typed since the list last chose, so the field shows the search. */
const typing = ref(false)
const listId = useId('combobox-list')
const wrap = useTemplateRef<HTMLElement>('wrap')
const button = useTemplateRef<HTMLButtonElement>('button')
const valueKey = () => props.valueKey || undefined
const opts = () => ({ valueKey: valueKey(), labelKey: props.labelKey, by: props.by })

// Items from `load`: the latest answer, while a newer one is not on its way.
const loaded = shallowRef<SelectItems>([])
const pending = ref(false)
const loadError = shallowRef<unknown>(undefined)
let controller: AbortController | null = null
let timer: ReturnType<typeof setTimeout> | undefined
const items = computed(() => (props.load ? loaded.value : props.items))
const busy = computed(() => !!props.loading || pending.value)

/** What the list searches for. */
const search = computed(() => {
  if (autocomplete.value) return String(model.value ?? '')
  return searchInPopup.value || typing.value || props.multiple ? term.value : ''
})

// In autocomplete mode the list chooses text, not a value.
const listModel = computed({
  get: () => (autocomplete.value ? undefined : model.value),
  set: (v: unknown) => {
    if (!autocomplete.value) model.value = v
  },
})
const list = useListbox(
  listProps(props, {
    items: () => items.value,
    ignoreFilter: () => !!props.load || !!props.ignoreFilter,
    loading: () => busy.value,
    error: () => loadError.value,
    valueKey,
    disabled: () => f.disabled.value,
    size: () => f.size.value,
  }),
  listModel,
  {
    id: listId,
    search: () => search.value,
    touch: () => sheet.value,
    label: () => {
      if (labelText(f.field)) return { 'aria-labelledby': f.field!.labelId }
      return { 'aria-label': (own.value['aria-label'] as string | undefined) ?? props.placeholder }
    },
    onChoose: (choice, _event, listChanged) => {
      let changed = listChanged
      if (choice.kind === 'option' && autocomplete.value) {
        changed = model.value !== choice.label
        model.value = choice.label
        term.value = choice.label
      }
      typing.value = false
      if (props.resetSearchTermOnSelect && !autocomplete.value) term.value = ''
      if (choice.kind === 'option' && changed) {
        emit('change', model.value)
        f.input()
      }
      if (!props.multiple) popup.hide()
    },
    onCreate: (t) => emit('create', t),
    onHighlight: (c) => emit('highlight', c?.kind === 'option' ? c.item : undefined),
  },
)
provideListbox(list)

const values = computed<unknown[]>(() => {
  const v = model.value
  if (props.multiple) return Array.isArray(v) ? v : []
  return v === undefined || v === null || v === '' ? [] : [v]
})
const labelFor = (v: unknown) => labelOfValue(items.value, v, opts())
const chosenLabel = computed(() => (props.multiple ? '' : labelFor(model.value)))
const selectedItem = computed(() => (props.multiple ? undefined : itemOf(items.value, model.value, opts())))
/** The text in the field. */
const text = computed(() => {
  if (autocomplete.value) return String(model.value ?? '')
  if (props.multiple) return term.value
  return typing.value ? term.value : chosenLabel.value
})
const hasValue = computed(() => values.value.length > 0)
const sheetTitle = computed(() =>
  open.value && (sheet.value || buttonMode.value) ? (labelText(f.field) ?? (own.value['aria-label'] as string | undefined) ?? props.placeholder) : undefined,
)
const fieldEl = () => wrap.value?.querySelector<HTMLInputElement>('input.weft-input__field') ?? null
const anchor = computed(() => (buttonMode.value ? button.value : (wrap.value?.querySelector<HTMLElement>('.weft-input') ?? null)) ?? undefined)
const focusTrigger = () => (buttonMode.value ? button.value : fieldEl())?.focus({ preventScroll: true })

function runLoad(t: string) {
  if (!props.load) return
  cancelTimer()
  controller?.abort()
  const mine = new AbortController()
  controller = mine
  pending.value = true
  loadError.value = undefined
  Promise.resolve()
    .then(() => props.load!(t, { signal: mine.signal }))
    .then(
      (result) => {
        if (controller !== mine) return
        loaded.value = result ?? []
        pending.value = false
      },
      (error: unknown) => {
        if (controller !== mine) return
        pending.value = false
        if (mine.signal.aborted) return
        // The results shown were for another search: none are kept.
        loaded.value = []
        loadError.value = error ?? new Error('load failed')
        emit('error', error)
      },
    )
}
const cancelTimer = () => {
  if (timer !== undefined) clearTimeout(timer)
  timer = undefined
}
/** No load waiting or on its way: the debounce timer goes, the request in flight is aborted. */
function stopLoading() {
  cancelTimer()
  controller?.abort()
  controller = null
  pending.value = false
}
// Load when the list opens, then for each new search, after `debounce` ms
// of quiet; busy from the keystroke on, so the results shown, which are for
// the search before, cannot be chosen meanwhile.
// Closing stops it all, so a closed Combobox is never busy.
watch([open, search], ([on, t], [wasOn]) => {
  if (!on || !props.load) return stopLoading()
  cancelTimer()
  if (!wasOn || props.debounce <= 0) return runLoad(t)
  pending.value = true
  loadError.value = undefined
  timer = setTimeout(() => runLoad(t), props.debounce)
})
onScopeDispose(stopLoading)

let start: 'selected' | 'last' | null = 'selected'
function show(at: typeof start) {
  start = at
  popup.show()
}
watch(open, (on) => {
  if (on) {
    if (start) list.highlight(start)
    start = 'selected'
    return
  }
  list.resetTypeahead()
  // The search at the top of the list goes with it.
  if (searchInPopup.value && !autocomplete.value && props.resetSearchTermOnBlur) {
    term.value = ''
    typing.value = false
  }
})
whenFocusLeaves(open, listId, () => (buttonMode.value ? button.value : fieldEl()), () => f.blur())
// Touch to pointer while open: the sheet's search field goes, and focus
// comes back to the field (or button) once the sheet's trap has let go; the
// search carries on in the field.
watch(sheet, (touch) => {
  if (!open.value) return
  if (!touch && !buttonMode.value) typing.value = term.value !== ''
  if (!touch) void nextTick(() => open.value && focusTrigger())
})

function setText(value: string, opening: boolean) {
  term.value = value
  typing.value = true
  if (autocomplete.value) model.value = value
  else if (value === '' && !props.multiple && props.resetModelValueOnClear && hasValue.value) {
    model.value = null
    emit('change', null)
    f.input()
  }
  // Only for the field a person is typing in: text put back into a field
  // they have left (what they typed before the page was live) opens nothing.
  if (opening && !open.value && value !== '' && document.activeElement === fieldEl()) show(null)
}
const onType = (value: string | number | null) => setText(String(value ?? ''), true)

function removeValue(v: unknown) {
  if (f.disabled.value) return
  model.value = values.value.filter((x) => !sameValue(x, v, props.by))
  emit('remove-tag', v)
  emit('change', model.value)
  f.input()
}
function onClear() {
  model.value = props.multiple ? [] : autocomplete.value ? '' : null
  term.value = ''
  typing.value = false
  emit('clear')
  emit('change', model.value)
  f.input()
  focusTrigger()
}

function onFieldKeydown(event: KeyboardEvent) {
  if (f.disabled.value) return
  const k = event.key
  if (sheet.value) {
    // On touch the field only opens the sheet.
    if (k === 'ArrowDown' || k === 'Enter' || k === ' ') {
      event.preventDefault()
      show('selected')
    }
    return
  }
  if (k === 'Backspace' && props.multiple && term.value === '' && values.value.length) {
    event.preventDefault()
    removeValue(values.value[values.value.length - 1])
    return
  }
  if (!open.value) {
    if (k === 'ArrowDown' || k === 'ArrowUp') {
      event.preventDefault()
      show(event.altKey ? null : k === 'ArrowUp' ? 'last' : 'selected')
    } else if (k === 'Escape' && text.value !== '') {
      event.preventDefault()
      setText('', false)
    }
    return
  }
  if (k === 'Tab') return popup.hide()
  if (k === 'ArrowUp' && event.altKey) {
    event.preventDefault()
    return popup.hide()
  }
  listKeydown(event)
}
/**
 * A key for the open list. Enter never submits the form from it, and
 * chooses nothing while its own `load` is on its way (the options shown are
 * for the search before); the `loading` prop alone does not hold it back.
 */
function listKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && pending.value) return event.preventDefault()
  if (list.keydown(event, { edges: false, typeahead: false, space: false }) || event.key === 'Enter') event.preventDefault()
}
const searchAttrs = searchField(list, {
  value: () => (autocomplete.value ? text.value : term.value),
  onInput: (value) => setText(value, false),
  placeholder: () => props.searchPlaceholder,
  onKeydown: listKeydown,
})
function onButtonKeydown(event: KeyboardEvent) {
  if (f.disabled.value || open.value) return
  const k = event.key
  if (k === 'ArrowDown' || k === 'ArrowUp') {
    event.preventDefault()
    show(k === 'ArrowUp' ? 'last' : 'selected')
  } else if (k.length === 1 && k !== ' ' && !event.ctrlKey && !event.metaKey && !event.altKey) {
    // Typing on the button starts the search.
    event.preventDefault()
    term.value = k
    show(null)
  }
}
function onFieldClick() {
  if (sheet.value) popup.press()
  else if (props.openOnClick) show('selected')
}
function onFieldFocus(event: FocusEvent) {
  if (props.openOnFocus && !sheet.value) show('selected')
  emit('focus', event)
}
/** Focus leaving the field (or button) for somewhere other than its list. */
function onBlur(event: FocusEvent) {
  const to = event.relatedTarget as Node | null
  const panel = document.getElementById(listId)?.closest('.weft-popover')
  if (to && panel?.contains(to)) return
  if (!sheet.value && !buttonMode.value) {
    popup.hide()
    if (props.resetSearchTermOnBlur && !autocomplete.value) {
      typing.value = false
      term.value = ''
    }
  }
  f.blur()
  emit('blur', event)
}
function onToggle() {
  focusTrigger()
  popup.press()
}
function onOpenChange(on: boolean) {
  open.value = on
}
/** A press in the floating list's padding keeps focus in the field. */
function onPanelDown(event: MouseEvent) {
  if (!sheet.value && !buttonMode.value) event.preventDefault()
}
</script>

<template>
  <span ref="wrap" class="weft-combobox" :data-multiple="multiple && !buttonMode ? 'true' : undefined" :data-open="open ? 'true' : undefined">
    <button
      v-if="buttonMode"
      ref="button"
      type="button"
      class="weft-input weft-select__trigger weft-combobox__trigger"
      aria-haspopup="dialog"
      v-bind="{ ...box, ...own }"
      :id="f.id.value"
      :disabled="f.disabled.value || undefined"
      :data-size="f.size.value"
      :data-invalid="f.invalid.value ? 'true' : undefined"
      :data-disabled="f.disabled.value ? 'true' : undefined"
      :data-open="open ? 'true' : undefined"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-describedby="describedBy()"
      :aria-invalid="f.aria.value['aria-invalid']"
      :aria-required="required || f.aria.value['aria-required'] ? 'true' : undefined"
      :aria-busy="busy ? 'true' : undefined"
      @click="popup.press()"
      @keydown="onButtonKeydown"
      @blur="onBlur"
      @focus="emit('focus', $event)"
    >
      <span v-if="$slots.leading || icon" class="weft-input__leading"><slot name="leading"><Icon v-if="icon" :name="icon" /></slot></span>
      <span class="weft-select__value" :data-placeholder="hasValue ? undefined : 'true'">
        <slot :model-value="model" :item="selectedItem" :open="open">{{ values.map(labelFor).join(', ') || placeholder || ' ' }}</slot>
      </span>
      <span class="weft-input__trailing">
        <slot name="trailing" :open="open">
          <Icon v-if="trailingIcon" :name="trailingIcon" />
          <svg v-else class="weft-select__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
        </slot>
      </span>
    </button>
    <Input
      v-else
      v-bind="required ? { ...$attrs, 'aria-required': 'true' } : $attrs"
      :model-value="text"
      :id="f.id.value"
      :size="f.size.value"
      :disabled="f.disabled.value"
      :readonly="sheet || undefined"
      :placeholder="multiple && hasValue ? undefined : placeholder"
      :icon="icon"
      :loading="busy"
      role="combobox"
      autocomplete="off"
      aria-autocomplete="list"
      aria-haspopup="listbox"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-controls="open ? listId : undefined"
      :aria-activedescendant="open && !sheet ? list.activeId.value : undefined"
      @update:model-value="onType"
      @keydown="onFieldKeydown"
      @click="onFieldClick"
      @focus="onFieldFocus"
      @blur="onBlur"
    >
      <template v-if="(multiple && hasValue) || $slots.leading" #leading>
        <slot name="leading"><Icon v-if="icon" :name="icon" /></slot>
        <span v-for="(v, i) in multiple ? values : []" :key="i" class="weft-tag" :data-size="f.size.value">
          <span class="weft-tag__text"><slot name="tag" :item="itemOf(items, v, opts())" :value="v">{{ labelFor(v) }}</slot></span>
          <button
            type="button"
            class="weft-tag__remove"
            tabindex="-1"
            :aria-label="`Remove ${labelFor(v)}`"
            :disabled="f.disabled.value || undefined"
            @mousedown.prevent
            @click="removeValue(v)"
          ><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg></button>
        </span>
      </template>
      <template #trailing>
        <button
          v-if="clear && hasValue && !f.disabled.value"
          type="button"
          class="weft-combobox__button weft-combobox__clear"
          tabindex="-1"
          aria-label="Clear"
          @mousedown.prevent
          @click="onClear"
        ><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg></button>
        <slot name="trailing" :open="open">
          <button
            type="button"
            class="weft-combobox__button weft-combobox__toggle"
            tabindex="-1"
            aria-label="Show options"
            :disabled="f.disabled.value || undefined"
            @mousedown.prevent
            @click="onToggle"
          >
            <Icon v-if="trailingIcon" :name="trailingIcon" />
            <svg v-else class="weft-select__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
          </button>
        </slot>
      </template>
    </Input>
    <input v-for="(v, i) in f.name.value ? values : []" :key="i" type="hidden" :name="f.name.value" :value="formValue(v)" />
    <Popover
      :open="open"
      :anchor="anchor"
      :auto-focus="buttonMode"
      :placement="placement"
      :offset="4"
      :title="sheetTitle"
      :role="buttonMode || sheet ? 'dialog' : 'presentation'"
      class="weft-select__popup weft-combobox__popup"
      @mousedown="onPanelDown"
      @update:open="onOpenChange"
    >
      <template #content>
        <span v-if="searchInPopup" class="weft-input weft-select__search" :data-size="f.size.value">
          <svg class="weft-select__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" /></svg>
          <input v-bind="searchAttrs" />
        </span>
        <Listbox :selected-icon="selectedIcon">
          <template v-if="$slots.item" #item="p"><slot name="item" v-bind="p" /></template>
          <template v-if="$slots['item-leading']" #item-leading="p"><slot name="item-leading" v-bind="p" /></template>
          <template v-if="$slots['item-label']" #item-label="p"><slot name="item-label" v-bind="p" /></template>
          <template v-if="$slots['item-trailing']" #item-trailing="p"><slot name="item-trailing" v-bind="p" /></template>
          <template v-if="$slots['create-item-label']" #create-item-label="p"><slot name="create-item-label" v-bind="p" /></template>
          <template v-if="$slots.empty" #empty="p"><slot name="empty" v-bind="p" /></template>
          <template v-if="$slots.error" #error="p"><slot name="error" v-bind="p" /></template>
        </Listbox>
      </template>
    </Popover>
  </span>
</template>
