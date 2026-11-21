<script setup vapor lang="ts">
// Text over several lines. With `autoresize` it grows with its text from
// `rows` lines up to `maxrows` (then scrolls), and shrinks back when text is
// removed. Inside a FormField it takes the field's id, description and error
// state. Attributes go to the <textarea>.
import { nextTick, onMounted, useTemplateRef, watch } from 'vue'
import { useControlAttrs, useField } from './field.ts'
import type { Size } from '../types.ts'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The text, bound with `v-model`. */
    modelValue?: string | null
    /** From `v-model.trim` and `.lazy` (update on change, not on every key). */
    modelModifiers?: { trim?: boolean; lazy?: boolean }
    /** Lines shown, and with autoresize the fewest it shrinks to. */
    rows?: number
    /** Grow with the text, from `rows` up to `maxrows`. */
    autoresize?: boolean
    /** With autoresize, the most lines it grows to before it scrolls; 0 for no limit. */
    maxrows?: number
    /** `sm`, `md` or `lg`: the text size and padding. */
    size?: Size
    /** Shown while it is empty; not a label. */
    placeholder?: string
    /** The form field name; a FormField's name applies when not set. */
    name?: string
    /** The textarea's id; inside a FormField the field's id is used. */
    id?: string
    /** Unavailable: natively disabled. */
    disabled?: boolean
    /** Shown but not editable. */
    readonly?: boolean
    /** The native `required`. */
    required?: boolean
  }>(),
  { modelValue: undefined, rows: 3, maxrows: 0 },
)

const emit = defineEmits<{
  /** The text changed (on each key, or on change with `.lazy`). */
  'update:modelValue': [value: string]
  /** The textarea lost focus. */
  blur: [event: FocusEvent]
  /** The textarea took focus. */
  focus: [event: FocusEvent]
  /** The native change: the text was committed. */
  change: [event: Event]
}>()

const f = useField(props)
const { own, describedBy } = useControlAttrs(f)
const area = useTemplateRef<HTMLTextAreaElement>('area')

/** Fit the height to the text, between `rows` and `maxrows` lines. */
function resize() {
  const el = area.value
  if (!el || !props.autoresize) return
  const style = getComputedStyle(el)
  const font = parseFloat(style.fontSize) || 16
  // line-height may come back in px, as a bare multiple, or as `normal`.
  const line = style.lineHeight.endsWith('px') ? parseFloat(style.lineHeight) : (parseFloat(style.lineHeight) || 1.2) * font
  const edges = (a: string, b: string) => (parseFloat(style.getPropertyValue(a)) || 0) + (parseFloat(style.getPropertyValue(b)) || 0)
  const padding = edges('padding-top', 'padding-bottom')
  const border = edges('border-top-width', 'border-bottom-width')
  // Back to `rows` first, so the text's own height shows when it shrinks.
  el.style.blockSize = 'auto'
  const max = props.maxrows > 0 ? props.maxrows * line + padding + border : Infinity
  const want = el.scrollHeight + border
  el.style.blockSize = `${Math.min(want, max)}px`
  el.style.overflowY = want > max ? 'auto' : 'hidden'
}

function commit(event: Event) {
  const raw = (event.target as HTMLTextAreaElement).value
  emit('update:modelValue', props.modelModifiers?.trim ? raw.trim() : raw)
  f.input()
}
function onInput(event: Event) {
  if (!props.modelModifiers?.lazy) commit(event)
  resize()
}
function onChange(event: Event) {
  if (props.modelModifiers?.lazy) commit(event)
  const el = event.target as HTMLTextAreaElement
  if (props.modelModifiers?.trim) el.value = el.value.trim()
  emit('change', event)
}
function onBlur(event: FocusEvent) {
  f.blur()
  emit('blur', event)
}

onMounted(resize)
watch(() => [props.modelValue, props.autoresize, props.rows, props.maxrows], () => nextTick(resize))

defineExpose({
  /** Focus the text. */
  focus: (options?: FocusOptions) => area.value?.focus(options),
  /** The <textarea> element. */
  textarea: area,
  /** Fit the height to the text now (after the textarea was shown, say). */
  resize,
})
</script>

<template>
  <textarea
    ref="area"
    class="weft-textarea"
    :class="$attrs.class"
    :style="$attrs.style"
    v-bind="own"
    :id="f.id.value"
    :name="f.name.value"
    :rows="rows"
    :value="modelValue ?? ''"
    :placeholder="placeholder"
    :disabled="f.disabled.value || undefined"
    :readonly="readonly || undefined"
    :required="required || undefined"
    :data-size="f.size.value"
    :data-invalid="f.invalid.value ? 'true' : undefined"
    :data-autoresize="autoresize ? 'true' : undefined"
    :aria-describedby="describedBy()"
    :aria-invalid="f.aria.value['aria-invalid']"
    :aria-required="f.aria.value['aria-required']"
    @input="onInput"
    @change="onChange"
    @blur="onBlur"
    @focus="emit('focus', $event)"
  ></textarea>
</template>
