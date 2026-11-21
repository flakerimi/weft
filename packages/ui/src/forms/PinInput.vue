<script setup vapor lang="ts">
// A short code typed one character per box: a group of single-character
// inputs whose value is the joined text. Typing a character fills the box and
// moves to the next; Backspace clears the box, or the one before when it is
// empty; Delete clears the box; Left and Right move (swapped in a
// right-to-left document), Home and End go to the first and last box. A
// pasted (or autofilled) code fills the boxes from the one it lands in.
// `complete` fires once every box is filled.
import { computed, ref, watch } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import { isRtl } from '../headless/dom.ts'
import { useId } from '../headless/useId.ts'
import { leavesGroup } from './dom.ts'
import { useField } from './field.ts'
import type { Size } from '../types.ts'

const props = withDefaults(
  defineProps<{
    /** The code, bound with `v-model`. */
    modelValue?: string
    /** The code to start with when `v-model` is not bound. */
    defaultValue?: string
    /** How many boxes. */
    length?: number
    /** `number` takes digits only and brings up a number pad; `text` takes letters and digits. */
    type?: 'text' | 'number'
    /** Hide what is typed, as a password field does. */
    mask?: boolean
    /** A one-time code: lets the browser fill it from a message (autocomplete="one-time-code"). */
    otp?: boolean
    /** Shown in each empty box. */
    placeholder?: string
    /** The form field name: the joined code posts under it. */
    name?: string
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** Unavailable: every box natively disabled. */
    disabled?: boolean
    /** The native `required`, on each box. */
    required?: boolean
    /** A box's name for assistive technology; `{n}` is its number and `{count}` how many there are. */
    boxLabel?: string
  }>(),
  { modelValue: undefined, defaultValue: '', length: 5, type: 'text', boxLabel: 'Character {n} of {count}' },
)

const emit = defineEmits<{
  /** A box changed: the joined code. */
  'update:modelValue': [value: string]
  /** Every box is filled: the code. */
  complete: [value: string]
}>()

const f = useField(props)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const base = useId('pin')
const accepts = (ch: string) => (props.type === 'number' ? /^[0-9]$/.test(ch) : /^[\p{L}\p{N}]$/u.test(ch))

// The boxes; a gap stays a gap (the joined value closes it up, but the boxes
// are only redrawn from the value when it changes from outside).
const split = (code: string) => Array.from({ length: props.length }, (_, i) => Array.from(code)[i] ?? '')
const chars = ref<string[]>(split(model.value))
watch([model, () => props.length], ([code]) => {
  if (code !== chars.value.join('') || chars.value.length !== props.length) chars.value = split(code)
})

const boxes: HTMLInputElement[] = []
const focusBox = (i: number) => {
  const el = boxes[Math.max(0, Math.min(props.length - 1, i))]
  el?.focus()
  el?.select()
}

function write(next: string[]) {
  chars.value = next
  const code = next.join('')
  model.value = code
  f.input()
  if (next.every((c) => c !== '')) emit('complete', code)
}

/** Put `text`'s acceptable characters into the boxes from `i`; returns the box after the last one filled. */
function fill(i: number, text: string): number {
  const next = [...chars.value]
  let at = i
  for (const ch of Array.from(text)) {
    if (at >= props.length) break
    if (accepts(ch)) next[at++] = ch
  }
  if (at !== i) write(next)
  return at
}

function onInput(i: number, event: Event) {
  const el = event.target as HTMLInputElement
  const typed = el.value
  // One box holds one character: show what the boxes hold, whatever was typed.
  el.value = chars.value[i] ?? ''
  if (typed === '') {
    if (chars.value[i]) write(chars.value.map((c, j) => (j === i ? '' : c)))
    return
  }
  // A character typed beside the one the box holds arrives with it: keep the
  // new one. Anything longer (autofill, a paste) is the whole text.
  const old = chars.value[i] ?? ''
  const pair = old !== '' && Array.from(typed).length === 2
  const text = pair && typed.startsWith(old) ? typed.slice(old.length) : pair && typed.endsWith(old) ? typed.slice(0, -old.length) : typed
  const after = fill(i, text)
  el.value = chars.value[i] ?? ''
  if (after > i) focusBox(after)
}

function onKeydown(i: number, event: KeyboardEvent) {
  if (event.altKey || event.ctrlKey || event.metaKey) return
  const rtl = isRtl(event.target as Element)
  const back = rtl ? 'ArrowRight' : 'ArrowLeft'
  const forward = rtl ? 'ArrowLeft' : 'ArrowRight'
  switch (event.key) {
    case 'Backspace':
      event.preventDefault()
      if (chars.value[i]) write(chars.value.map((c, j) => (j === i ? '' : c)))
      else if (i > 0) {
        write(chars.value.map((c, j) => (j === i - 1 ? '' : c)))
        focusBox(i - 1)
      }
      return
    case 'Delete':
      event.preventDefault()
      if (chars.value[i]) write(chars.value.map((c, j) => (j === i ? '' : c)))
      return
    case back: event.preventDefault(); return focusBox(i - 1)
    case forward: event.preventDefault(); return focusBox(i + 1)
    case 'Home': event.preventDefault(); return focusBox(0)
    case 'End': event.preventDefault(); return focusBox(props.length - 1)
  }
}

function onPaste(i: number, event: ClipboardEvent) {
  const text = event.clipboardData?.getData('text') ?? ''
  event.preventDefault()
  const after = fill(i, text)
  focusBox(after)
}

const inputmode = computed(() => (props.type === 'number' ? 'numeric' : 'text'))
const labelFor = (i: number) => props.boxLabel.replace('{n}', String(i + 1)).replace('{count}', String(props.length))

defineExpose({
  /** Focus the first empty box, or the last. */
  focus() {
    const empty = chars.value.findIndex((c) => !c)
    focusBox(empty < 0 ? props.length - 1 : empty)
  },
})
</script>

<template>
  <div
    class="weft-pin-input"
    role="group"
    :data-size="f.size.value"
    :data-invalid="f.invalid.value ? 'true' : undefined"
    :data-disabled="f.disabled.value ? 'true' : undefined"
    :aria-labelledby="f.field?.labelId"
    :aria-describedby="f.aria.value['aria-describedby']"
    @focusout="leavesGroup($event) && f.blur()"
  >
    <input
      v-for="(ch, i) in chars"
      :key="i"
      :ref="(el) => { if (el) boxes[i] = el as HTMLInputElement }"
      class="weft-pin-input__box"
      :id="i === 0 ? f.id.value ?? `${base}-0` : undefined"
      :type="mask ? 'password' : 'text'"
      :inputmode="inputmode"
      :autocomplete="otp && i === 0 ? 'one-time-code' : 'off'"
      :value="ch"
      :placeholder="placeholder"
      :disabled="f.disabled.value || undefined"
      :required="required || undefined"
      :aria-label="labelFor(i)"
      :aria-invalid="f.aria.value['aria-invalid']"
      :data-filled="ch ? 'true' : undefined"
      @input="onInput(i, $event)"
      @keydown="onKeydown(i, $event)"
      @paste="onPaste(i, $event)"
      @focus="($event.target as HTMLInputElement).select()"
    />
    <input v-if="f.name.value" type="hidden" :name="f.name.value" :value="chars.join('')" />
  </div>
</template>
