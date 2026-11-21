<script setup vapor lang="ts">
// A value, or a range, picked along a track: the WAI-ARIA slider pattern
// (multi-thumb for a range). Each thumb is a role="slider" in the tab order.
// Right and Up raise it by `step`, Left and Down lower it (Right and Left
// swap in a right-to-left document), Page Up and Page Down move ten steps,
// Home and End go to the ends. A press on the track moves the nearest thumb
// there and drags it. With an array value it draws a thumb per value; thumbs
// never pass each other, and each one's aria-valuemin and aria-valuemax are
// its neighbours. A thumb is drawn small, and its hit area (and the track's)
// is at least --weft-hit-min, so on touch it takes a 44 px press.
import { computed, ref, useAttrs, useTemplateRef } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import { isRtl } from '../headless/dom.ts'
import { useId } from '../headless/useId.ts'
import { useField } from './field.ts'
import { clamp, snap } from './numbers.ts'
import type { Size, Tone } from '../types.ts'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The value, or an array of values for a range, bound with `v-model`. */
    modelValue?: number | number[]
    /** The value to start with when `v-model` is not bound; an array for a range. */
    defaultValue?: number | number[]
    /** The start of the track. */
    min?: number
    /** The end of the track. */
    max?: number
    /** The values it moves between, counted from `min`. */
    step?: number
    /** For a range: the fewest steps kept between neighbouring thumbs. */
    minStepsBetweenThumbs?: number
    /** `horizontal` (min at the inline start) or `vertical` (min at the bottom). */
    orientation?: 'horizontal' | 'vertical'
    /** Each thumb's name for assistive technology; for a range the default is "Minimum" and "Maximum". A single thumb takes the Slider's own aria-label, or its FormField's label. */
    thumbLabels?: string[]
    /** The form field name: each value posts under it from a hidden input. */
    name?: string
    /** `sm`, `md` or `lg`: the track's thickness and the thumb's size. */
    size?: Size
    /** The colour of the filled part and the thumbs. */
    tone?: Tone
    /** Unavailable: out of the tab order, and presses do nothing. */
    disabled?: boolean
  }>(),
  { modelValue: undefined, defaultValue: undefined, min: 0, max: 100, step: 1, minStepsBetweenThumbs: 0, orientation: 'horizontal', tone: 'primary' },
)

const emit = defineEmits<{
  /** A thumb moved, by key, press or drag. */
  'update:modelValue': [value: number | number[]]
  /** A move ended: a key press, or a drag let go; the final value. */
  change: [value: number | number[]]
}>()

const f = useField(props)
// Its thumbs are not labelable elements: they take the field's label by id.
f.field?.namedByLabel()
const base = useId('slider')
// A name given to the Slider (aria-label, aria-labelledby) names its thumb;
// the rest of the attributes go to the root.
const attrs = useAttrs()
const rootAttrs = computed(() => {
  const { 'aria-label': _label, 'aria-labelledby': _labelledBy, ...rest } = attrs
  return rest
})
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? props.min)
const range = computed(() => Array.isArray(model.value))
const values = computed(() => (Array.isArray(model.value) ? model.value : [model.value]))
const disabled = computed(() => f.disabled.value)

const percent = (v: number) => (props.max > props.min ? ((clamp(v, props.min, props.max) - props.min) / (props.max - props.min)) * 100 : 0)
const gap = computed(() => props.minStepsBetweenThumbs * props.step)
/** The lowest and highest thumb `i` may go: the ends, or its neighbours. */
const bounds = (i: number, list = values.value): [number, number] => [
  i > 0 ? list[i - 1]! + gap.value : props.min,
  i < list.length - 1 ? list[i + 1]! - gap.value : props.max,
]

function label(i: number): string | undefined {
  if (props.thumbLabels?.[i]) return props.thumbLabels[i]
  if (range.value) return i === 0 ? 'Minimum' : i === values.value.length - 1 ? 'Maximum' : undefined
  return attrs['aria-label'] as string | undefined
}
/** Thumb `i`'s id: the field's for the first, so the field's id names a control. */
const thumbId = (i: number) => (i === 0 && f.id.value ? f.id.value : `${base}-${i}`)
/**
 * A single thumb: the Slider's aria-labelledby, else its field's label. A
 * range thumb in a field: the field's label and then its own end
 * ("Price Minimum"), by pointing at itself after the label.
 */
function labelledBy(i: number): string | undefined {
  const own = attrs['aria-labelledby'] as string | undefined
  if (range.value) return f.field ? `${own ?? f.field.labelId} ${thumbId(i)}` : own && `${own} ${thumbId(i)}`
  if (label(i)) return undefined
  return own ?? f.field?.labelId
}

/** Move thumb `i` toward `to`, kept on a step and between its neighbours; true when it moved. */
function move(i: number, to: number): boolean {
  const [lo, hi] = bounds(i)
  const next = clamp(snap(to, props.step, props.min), lo, hi)
  if (next === values.value[i]) return false
  if (range.value) {
    const list = [...values.value]
    list[i] = next
    model.value = list
  } else model.value = next
  f.input()
  return true
}
const commit = () => emit('change', model.value)

const root = useTemplateRef<HTMLElement>('root')
const thumbs = ref<HTMLElement[]>([])

function onKeydown(i: number, event: KeyboardEvent) {
  if (disabled.value || event.altKey || event.ctrlKey || event.metaKey) return
  const rtl = props.orientation === 'horizontal' && isRtl(root.value)
  const up = rtl ? 'ArrowLeft' : 'ArrowRight'
  const down = rtl ? 'ArrowRight' : 'ArrowLeft'
  const v = values.value[i]!
  const [lo, hi] = bounds(i)
  const target: Record<string, number> = {
    [up]: v + props.step,
    ArrowUp: v + props.step,
    [down]: v - props.step,
    ArrowDown: v - props.step,
    PageUp: v + props.step * 10,
    PageDown: v - props.step * 10,
    Home: lo,
    End: hi,
  }
  if (!(event.key in target)) return
  event.preventDefault()
  if (move(i, target[event.key]!)) commit()
}

// Pointer: a press on the track or a thumb picks the nearest thumb, moves it
// there and drags it until the pointer is let go.
let dragging = -1
function valueAt(event: PointerEvent): number {
  const track = root.value!.querySelector('.weft-slider__track')!.getBoundingClientRect()
  let share: number
  if (props.orientation === 'vertical') share = track.height ? (track.bottom - event.clientY) / track.height : 0
  else {
    share = track.width ? (event.clientX - track.left) / track.width : 0
    if (isRtl(root.value)) share = 1 - share
  }
  return props.min + clamp(share, 0, 1) * (props.max - props.min)
}
function nearest(v: number): number {
  let best = 0
  values.value.forEach((x, i) => {
    const d = Math.abs(x - v)
    const bestD = Math.abs(values.value[best]! - v)
    // On a tie (thumbs stacked) take the one that can move that way.
    if (d < bestD || (d === bestD && v > x)) best = i
  })
  return best
}
function onPointerdown(event: PointerEvent) {
  if (disabled.value || event.button !== 0) return
  event.preventDefault()
  const v = valueAt(event)
  dragging = nearest(v)
  move(dragging, v)
  thumbs.value[dragging]?.focus()
  root.value?.setPointerCapture?.(event.pointerId)
}
function onPointermove(event: PointerEvent) {
  if (dragging >= 0) move(dragging, valueAt(event))
}
function onPointerup(event: PointerEvent) {
  if (dragging < 0) return
  dragging = -1
  root.value?.releasePointerCapture?.(event.pointerId)
  commit()
}

const fill = computed(() => {
  const list = values.value
  const start = range.value ? percent(list[0]!) : 0
  const end = percent(list[list.length - 1]!)
  return props.orientation === 'vertical'
    ? { insetBlockEnd: `${start}%`, blockSize: `${end - start}%` }
    : { insetInlineStart: `${start}%`, inlineSize: `${end - start}%` }
})
const at = (v: number) => (props.orientation === 'vertical' ? { insetBlockEnd: `${percent(v)}%` } : { insetInlineStart: `${percent(v)}%` })

defineExpose({
  /** Focus the first thumb. */
  focus: (options?: FocusOptions) => thumbs.value[0]?.focus(options),
})
</script>

<template>
  <div
    ref="root"
    v-bind="rootAttrs"
    class="weft-slider"
    :data-orientation="orientation"
    :data-size="f.size.value"
    :data-tone="tone"
    :data-disabled="disabled ? 'true' : undefined"
    @pointerdown="onPointerdown"
    @pointermove="onPointermove"
    @pointerup="onPointerup"
    @pointercancel="onPointerup"
  >
    <div class="weft-slider__track">
      <div class="weft-slider__fill" :style="fill"></div>
    </div>
    <span
      v-for="(v, i) in values"
      :key="i"
      :ref="(el) => { if (el) thumbs[i] = el as HTMLElement }"
      class="weft-slider__thumb"
      role="slider"
      :id="thumbId(i)"
      :tabindex="disabled ? undefined : 0"
      :style="at(v)"
      :aria-valuenow="v"
      :aria-valuemin="bounds(i)[0]"
      :aria-valuemax="bounds(i)[1]"
      :aria-orientation="orientation"
      :aria-disabled="disabled ? 'true' : undefined"
      :aria-label="label(i)"
      :aria-labelledby="labelledBy(i)"
      :aria-describedby="f.aria.value['aria-describedby']"
      :aria-invalid="f.aria.value['aria-invalid']"
      @keydown="onKeydown(i, $event)"
      @blur="f.blur()"
    ></span>
    <input v-for="(v, i) in (f.name.value ? values : [])" :key="`h${i}`" type="hidden" :name="f.name.value" :value="v" />
  </div>
</template>
