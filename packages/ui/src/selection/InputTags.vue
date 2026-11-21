<script setup vapor lang="ts">
// A text field that turns what is typed into tags: Enter or the delimiter
// (a comma) adds the text as a tag, and so can a paste, Tab and leaving the
// field when asked. The tags are a list before the text, each with a ×.
// Backspace in an empty field, or ArrowLeft at its start (ArrowRight right
// to left), moves focus onto the last tag; the arrows move between tags and
// back to the field, and Backspace or Delete takes out the focused tag.
// Duplicates and tags past `max` are refused (`invalid`). Inside a FormField
// the text field takes the field's id, description, error and required
// mark; with a name the tags submit in hidden inputs, one each.
import { computed, nextTick, ref, useTemplateRef } from 'vue'
import Icon from '../basics/Icon.vue'
import { isRtl } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { shareField, useField } from '../forms/field.ts'
import Input from '../forms/Input.vue'
import { formValue, sameValue } from './items.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/tag.css'
import '../styles/input-tags.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The tags, bound with `v-model`. */
    modelValue?: unknown[]
    /** The tags it starts with when `v-model` is not bound. */
    defaultValue?: unknown[]
    /** Shown in the empty field. */
    placeholder?: string
    /** Typing this adds the text before it as a tag; a string or a pattern. */
    delimiter?: string | RegExp
    /** A paste is split at the delimiter into tags. */
    addOnPaste?: boolean
    /** Tab adds the text as a tag (and stays). */
    addOnTab?: boolean
    /** Leaving the field adds the text as a tag. */
    addOnBlur?: boolean
    /** The same tag may be added twice. */
    duplicate?: boolean
    /** At most this many tags. */
    max?: number
    /** At most this many characters in a tag. */
    maxLength?: number
    /** Turn the typed text into the tag's value. */
    convertValue?: (text: string) => unknown
    /** The text a tag shows for its value. */
    displayValue?: (value: unknown) => string
    /** `sm`, `md` or `lg`; a FieldGroup's or FormField's size applies when not set. */
    size?: Size
    /** A Lucide icon before the tags. The leading slot replaces it. */
    icon?: string
    /** Work in progress: a spinner at the end, and aria-busy. */
    loading?: boolean
    /** Tags cannot be added or taken out. */
    disabled?: boolean
    /** The form field name its tags submit under; a FormField's name applies when not set. */
    name?: string
    /** The text field's id; inside a FormField the field's id is used. */
    id?: string
  }>(),
  { modelValue: undefined, defaultValue: undefined, delimiter: ',' },
)

const emit = defineEmits<{
  /** The tags changed. */
  'update:modelValue': [value: unknown[]]
  /** A tag was added, with its value. */
  addTag: [value: unknown]
  /** A tag was taken out, with its value. */
  removeTag: [value: unknown]
  /** A tag was refused (a duplicate, or past `max`), with its value. */
  invalid: [value: unknown]
  /** The tags changed by an addition or a removal. */
  change: [value: unknown[]]
  /** Focus left the field and its tags. */
  blur: [event: FocusEvent]
  /** The field took focus. */
  focus: [event: FocusEvent]
}>()

defineSlots<{
  /** Before the tags, in place of `icon`. */
  leading?: () => unknown
  /** After the text. */
  trailing?: () => unknown
  /** A tag's text. */
  'item-text'?: (props: { item: unknown; index: number }) => unknown
}>()

const f = useField(props)
// The Input inside is the field's control, but its text is a draft.
shareField(f.field)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? [])
const draft = ref('')
const wrap = useTemplateRef<HTMLElement>('wrap')
const tags = computed(() => (Array.isArray(model.value) ? model.value : []))
const display = (v: unknown) => (props.displayValue ? props.displayValue(v) : String(v))
const box = () => wrap.value?.closest('.weft-input-tags') ?? null
const fieldEl = () => box()?.querySelector<HTMLInputElement>('input.weft-input__field') ?? null
const tagEls = () => [...(box()?.querySelectorAll<HTMLElement>('.weft-tag') ?? [])]
const focusTag = (i: number) => tagEls()[i]?.focus()
const focusField = () => fieldEl()?.focus()

function add(text: string): boolean {
  const t = text.trim()
  if (!t || f.disabled.value) return false
  const value = props.convertValue ? props.convertValue(t) : t
  if ((props.max !== undefined && tags.value.length >= props.max) || (!props.duplicate && tags.value.some((v) => sameValue(v, value)))) {
    emit('invalid', value)
    return false
  }
  model.value = [...tags.value, value]
  emit('addTag', value)
  emit('change', model.value)
  f.input()
  return true
}
function removeAt(i: number) {
  if (f.disabled.value) return
  const value = tags.value[i]
  model.value = tags.value.filter((_, j) => j !== i)
  emit('removeTag', value)
  emit('change', model.value)
  f.input()
}
const parts = (text: string) => text.split(props.delimiter)

function onDraft(value: string | number | null) {
  const text = String(value ?? '')
  const pieces = parts(text)
  if (pieces.length > 1) {
    // Everything before the last delimiter becomes tags; the rest stays to type on.
    const rest = pieces.pop()!
    const refused = pieces.filter((p) => p.trim() && !add(p))
    draft.value = refused.length ? [...refused, rest].join(typeof props.delimiter === 'string' ? props.delimiter : ',') : rest
  } else draft.value = text
  showDraft()
}
/** The field shows the draft now: it may hold typed text the render has not caught up with. */
function showDraft() {
  const el = fieldEl()
  if (el && el.value !== draft.value) el.value = draft.value
}
function commit(): boolean {
  if (!add(draft.value)) return false
  draft.value = ''
  showDraft()
  return true
}

function onKeydown(event: KeyboardEvent) {
  if (f.disabled.value) return
  const el = event.target as HTMLInputElement
  if (event.key === 'Enter') {
    if (draft.value.trim()) {
      event.preventDefault()
      commit()
    }
    return
  }
  if (event.key === 'Tab' && !event.shiftKey && props.addOnTab && draft.value.trim()) {
    if (commit()) event.preventDefault()
    return
  }
  if (!tags.value.length || el.selectionStart !== 0 || el.selectionEnd !== 0) return
  const back = isRtl(el) ? 'ArrowRight' : 'ArrowLeft'
  if ((event.key === 'Backspace' && draft.value === '') || event.key === back) {
    event.preventDefault()
    focusTag(tags.value.length - 1)
  }
}
function onTagKeydown(event: KeyboardEvent, i: number) {
  if (f.disabled.value) return
  const rtl = isRtl(event.currentTarget as Element)
  const back = rtl ? 'ArrowRight' : 'ArrowLeft'
  const forward = rtl ? 'ArrowLeft' : 'ArrowRight'
  const k = event.key
  if (k === 'Backspace' || k === 'Delete') {
    event.preventDefault()
    removeAt(i)
    const left = tags.value.length
    // Backspace goes to the tag before, Delete to the one that took its place.
    const next = k === 'Backspace' ? i - 1 : i
    void nextTick(() => (left && next >= 0 && next < left ? focusTag(next) : left && k === 'Backspace' ? focusTag(0) : focusField()))
  } else if (k === back) {
    event.preventDefault()
    if (i > 0) focusTag(i - 1)
  } else if (k === forward) {
    event.preventDefault()
    if (i < tags.value.length - 1) focusTag(i + 1)
    else focusField()
  } else if (k === 'Escape') focusField()
}
function onPaste(event: ClipboardEvent) {
  if (!props.addOnPaste || f.disabled.value) return
  const text = event.clipboardData?.getData('text') ?? ''
  const pieces = parts(text).filter((p) => p.trim())
  if (pieces.length <= 1 && !draft.value) return
  event.preventDefault()
  for (const p of pieces) add(p)
}
function onRemove(i: number) {
  removeAt(i)
  void nextTick(focusField)
}
/** Focus leaving the field and its tags altogether. */
function leaving(event: FocusEvent) {
  const to = event.relatedTarget as Node | null
  return !to || !box()?.contains(to)
}
function onBlur(event: FocusEvent) {
  if (!leaving(event)) return
  if (props.addOnBlur) commit()
  f.blur()
  emit('blur', event)
}
function onTagBlur(event: FocusEvent) {
  if (!leaving(event)) return
  f.blur()
  emit('blur', event)
}
</script>

<template>
  <Input
    v-bind="$attrs"
    class="weft-input-tags"
    :model-value="draft"
    :id="f.id.value"
    :size="f.size.value"
    :disabled="f.disabled.value"
    :placeholder="placeholder"
    :loading="loading"
    :maxlength="maxLength"
    @update:model-value="onDraft"
    @keydown="onKeydown"
    @paste="onPaste"
    @blur="onBlur"
    @focus="emit('focus', $event)"
  >
    <template #leading>
      <span ref="wrap" class="weft-input-tags__start">
        <slot name="leading"><Icon v-if="icon" :name="icon" /></slot>
      </span>
      <span v-if="tags.length" class="weft-input-tags__list" role="list">
        <span
          v-for="(v, i) in tags"
          :key="`${i}:${formValue(v)}`"
          class="weft-tag"
          role="listitem"
          tabindex="-1"
          :data-size="f.size.value"
          @keydown="onTagKeydown($event, i)"
          @blur="onTagBlur"
        >
          <span class="weft-tag__text"><slot name="item-text" :item="v" :index="i">{{ display(v) }}</slot></span>
          <button
            type="button"
            class="weft-tag__remove"
            tabindex="-1"
            :aria-label="`Remove ${display(v)}`"
            :disabled="f.disabled.value || undefined"
            @click="onRemove(i)"
          ><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg></button>
        </span>
      </span>
    </template>
    <template v-if="$slots.trailing" #trailing><slot name="trailing" /></template>
  </Input>
  <input v-for="(v, i) in f.name.value ? tags : []" :key="i" type="hidden" :name="f.name.value" :value="formValue(v)" />
</template>
