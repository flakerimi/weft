<script setup vapor lang="ts">
// Panels of content, one shown at a time, picked from a row of tabs
// (WAI-ARIA APG, Tabs). The tab list is one tab stop, on the selected tab;
// the arrow keys move along it (Left and Right, mirrored right to left, or
// Up and Down when vertical), wrapping and skipping disabled tabs, Home and
// End to the ends. `automatic` activation (the default) selects the tab
// focus lands on; `manual` waits for Enter, Space or a press. Every tab
// controls a panel, named by its tab; only the selected one is shown. A
// panel's content renders while it is selected (lazy), or from its first
// visit on with `keep-alive`. With `hash`, the address's #fragment names the
// selected tab, both ways (a disabled tab or an unknown name is ignored);
// one Tabs per page owns the hash, the first mounted. A vertical list goes above the panels when narrow;
// on touch every tab is at least --weft-hit-min tall.
import { computed, onMounted, onScopeDispose, shallowRef, useAttrs, useTemplateRef, watch } from 'vue'
import Badge from '../basics/Badge.vue'
import Icon from '../basics/Icon.vue'
import { useWidth } from '../adaptive/useWidth.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import type { Size, Tone } from '../types.ts'
import type { TabsItem } from './tabs.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/tabs.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The tabs: `{ label, value, icon, badge, content, disabled, slot }`. */
    items?: TabsItem[]
    /** The selected tab's value, bound with `v-model`: its `value`, else its index. */
    modelValue?: string | number
    /** The tab selected at first when `v-model` is not bound; default the first enabled one. */
    defaultValue?: string | number
    /** `automatic`: focusing a tab selects it. `manual`: Enter, Space or a press selects the focused tab. */
    activation?: 'automatic' | 'manual'
    /** `horizontal` (a row above the panels) or `vertical` (a column beside them, above them when narrow). */
    orientation?: 'horizontal' | 'vertical'
    /** `pill`: tabs in a tinted track, the selected one raised. `link`: an underline marks the selected tab. */
    variant?: 'pill' | 'link'
    /** The colour of the selected tab. */
    tone?: Tone
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** Draw the panels; false draws the tab list alone (to switch something else). */
    content?: boolean
    /** Keep a panel's content mounted, hidden, once it has been shown, so its state survives; by default only the selected panel's content is rendered. */
    keepAlive?: boolean
    /** Read the selected tab from the address's #fragment, and write it there on a change (replacing the history entry, not adding one). One Tabs per page may: a second warns and leaves the hash alone. */
    hash?: boolean
  }>(),
  {
    items: () => [],
    modelValue: undefined,
    defaultValue: undefined,
    activation: 'automatic',
    orientation: 'horizontal',
    variant: 'pill',
    tone: 'primary',
    size: 'md',
    content: true,
  },
)

const emit = defineEmits<{
  /** A tab was selected. */
  'update:modelValue': [value: string | number]
}>()

defineSlots<{
  /** Before a tab's label, in place of its icon. */
  leading?: (props: { item: TabsItem; index: number; selected: boolean }) => unknown
  /** A tab's label, in place of `label`. */
  default?: (props: { item: TabsItem; index: number; selected: boolean }) => unknown
  /** After a tab's label, in place of its badge. */
  trailing?: (props: { item: TabsItem; index: number; selected: boolean }) => unknown
  /** A panel's content, in place of `content`. */
  content?: (props: { item: TabsItem; index: number }) => unknown
  /** The panel of the item whose `slot` names it. */
  [name: string]: ((props: { item: TabsItem; index: number; selected: boolean }) => unknown) | undefined
}>()

const attrs = useAttrs()
const listAttrs = computed(() => ({ 'aria-label': attrs['aria-label'] as string | undefined, 'aria-labelledby': attrs['aria-labelledby'] as string | undefined }))
const rootAttrs = computed(() => {
  const { 'aria-label': _l, 'aria-labelledby': _b, ...rest } = attrs
  return rest
})

const valueOf = (item: TabsItem, index: number) => item.value ?? index
const firstEnabled = () => {
  const i = props.items.findIndex((t) => !t.disabled)
  return i < 0 ? undefined : valueOf(props.items[i]!, i)
}
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? firstEnabled() ?? 0)
const selectedIndex = computed(() => props.items.findIndex((t, i) => valueOf(t, i) === model.value))

const id = useId('tabs')
const root = useTemplateRef<HTMLElement>('root')
const list = useTemplateRef<HTMLElement>('list')
const width = useWidth(root, { prepaint: false })
const orientation = computed(() => (props.orientation === 'vertical' && width.value !== 'narrow' ? 'vertical' : 'horizontal'))
const roving = useRovingFocus(list, {
  get orientation() {
    return orientation.value
  },
  items: '[role="tab"]',
})

/** Values whose panel has been shown, for keep-alive. */
const visited = shallowRef<(string | number)[]>([])
watch(
  model,
  (v) => {
    if (v !== undefined && !visited.value.includes(v)) visited.value = [...visited.value, v]
  },
  { immediate: true },
)
// The selected tab holds the tab stop, whoever selected it.
watch([selectedIndex, list], ([i]) => i >= 0 && roving.setActive(i), { flush: 'post' })

function select(item: TabsItem, index: number) {
  if (item.disabled) return
  model.value = valueOf(item, index)
}
function onFocusin(event: FocusEvent) {
  if (props.activation !== 'automatic') return
  const el = (event.target as Element).closest<HTMLElement>('[role="tab"]')
  const index = Number(el?.dataset.index)
  const item = props.items[index]
  if (el && item) select(item, index)
}

// The address's #fragment, with `hash`.
const fromHash = () => {
  const name = decodeURIComponent(location.hash.slice(1))
  const index = props.items.findIndex((t, i) => String(valueOf(t, i)) === name && !t.disabled)
  if (index >= 0) model.value = valueOf(props.items[index]!, index)
}
// One Tabs on a page owns the hash: the first mounted with `hash`.
const me = {}
let owns = false
onMounted(() => {
  if (!props.hash) return
  if (hashOwner && hashOwner !== me) {
    console.warn('Tabs: another Tabs on this page already reads and writes the address\'s #hash; this one leaves it alone. Give `hash` to one Tabs per page.')
    return
  }
  hashOwner = me
  owns = true
  fromHash()
  window.addEventListener('hashchange', fromHash)
})
onScopeDispose(() => {
  if (!owns) return
  window.removeEventListener('hashchange', fromHash)
  if (hashOwner === me) hashOwner = undefined
})
watch(model, (v) => {
  if (!owns || v === undefined) return
  const next = `#${encodeURIComponent(String(v))}`
  if (location.hash !== next) history.replaceState(history.state, '', next)
})

const tabId = (i: number) => `${id}-tab-${i}`
const panelId = (i: number) => `${id}-panel-${i}`
const rendered = (item: TabsItem, i: number) => {
  const v = valueOf(item, i)
  return v === model.value || (props.keepAlive && visited.value.includes(v))
}
</script>

<script lang="ts">
/** The Tabs that owns the address's #hash on this page, if any. */
let hashOwner: object | undefined
</script>

<template>
  <div
    ref="root"
    v-bind="rootAttrs"
    class="weft-tabs"
    :data-orientation="orientation"
    :data-variant="variant"
    :data-size="size"
    :data-tone="tone"
  >
    <div
      ref="list"
      class="weft-tabs__list"
      role="tablist"
      :aria-orientation="orientation"
      :aria-label="listAttrs['aria-label']"
      :aria-labelledby="listAttrs['aria-labelledby']"
      @focusin="onFocusin"
    >
      <button
        v-for="(item, i) in items"
        :id="tabId(i)"
        :key="String(valueOf(item, i))"
        type="button"
        class="weft-tabs__tab"
        role="tab"
        :data-index="i"
        :aria-selected="i === selectedIndex ? 'true' : 'false'"
        :aria-controls="content ? panelId(i) : undefined"
        :disabled="item.disabled || undefined"
        :data-state="i === selectedIndex ? 'active' : 'inactive'"
        @click="select(item, i)"
      >
        <slot name="leading" :item="item" :index="i" :selected="i === selectedIndex"><Icon v-if="item.icon" :name="item.icon" class="weft-tabs__icon" /></slot>
        <span class="weft-tabs__label"><slot :item="item" :index="i" :selected="i === selectedIndex">{{ item.label }}</slot></span>
        <slot name="trailing" :item="item" :index="i" :selected="i === selectedIndex"><Badge v-if="item.badge !== undefined" :label="item.badge" variant="soft" tone="neutral" size="sm" /></slot>
      </button>
    </div>
    <template v-if="content">
      <div
        v-for="(item, i) in items"
        :id="panelId(i)"
        :key="String(valueOf(item, i))"
        class="weft-tabs__panel"
        role="tabpanel"
        tabindex="0"
        :aria-labelledby="tabId(i)"
        :hidden="i !== selectedIndex || undefined"
      >
        <template v-if="rendered(item, i)">
          <slot v-if="item.slot && $slots[item.slot]" :name="item.slot" :item="item" :index="i" :selected="i === selectedIndex" />
          <slot v-else name="content" :item="item" :index="i">{{ item.content }}</slot>
        </template>
      </div>
    </template>
  </div>
</template>
