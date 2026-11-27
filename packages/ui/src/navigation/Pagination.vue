<script setup vapor lang="ts">
// Moves through the pages of a long list: a navigation landmark of buttons
// (links with `to`) for the first, previous, next and last pages and for the
// pages around the current one, which says aria-current="page". The
// controls at an end are disabled. With `show-edges` the first and last
// pages always show, ellipses (hidden from assistive technology) for the
// gaps. When the viewport is narrow it shows at most one page each side and
// drops the first and last controls. Before mounting (on the server, and
// while hydrating) it draws both forms, each entry of one form only marked
// `data-only`, and CSS shows the narrow one on a narrow viewport, so a page
// that is never hydrated pages right too. On touch every button is at least
// --weft-hit-min tall (Button's rule).
import { computed, onMounted, shallowRef } from 'vue'
import Button from '../basics/Button.vue'
import { useWeft } from '../adaptive/context.ts'
import { useViewportWidth } from '../adaptive/useWidth.ts'
import { useControllable } from '../headless/useControllable.ts'
import type { Size, Tone, Variant } from '../types.ts'
import { PAGINATION_LABELS, pageRange, pageRangeBoth, type PageSlot, type PaginationLabels } from './pagination.ts'
import '../styles/base.css'
import '../styles/pagination.css'

const props = withDefaults(
  defineProps<{
    /** The current page, from 1, bound with `v-model:page`. */
    page?: number
    /** The page shown at first when `v-model:page` is not bound. */
    defaultPage?: number
    /** How many items there are in all. */
    total?: number
    /** How many items a page holds. */
    itemsPerPage?: number
    /** How many pages to show on each side of the current one. */
    siblingCount?: number
    /** Always show the first and last pages, with ellipses for the gaps. */
    showEdges?: boolean
    /** Show the first, previous, next and last buttons. */
    showControls?: boolean
    /** Every button is disabled. */
    disabled?: boolean
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** The colour of the other pages and the controls. */
    tone?: Tone
    /** The variant of the other pages and the controls. */
    variant?: Variant
    /** The colour of the current page. */
    activeTone?: Tone
    /** The variant of the current page. */
    activeVariant?: Variant
    /** The address of a page: the buttons become links to it. */
    to?: (page: number) => string
    /** The words it says, for assistive technology; give some or all to translate them. */
    labels?: Partial<PaginationLabels>
  }>(),
  {
    page: undefined,
    defaultPage: 1,
    total: 0,
    itemsPerPage: 10,
    siblingCount: 2,
    showControls: true,
    size: 'md',
    tone: 'neutral',
    variant: 'ghost',
    activeTone: 'primary',
    activeVariant: 'solid',
  },
)

const emit = defineEmits<{
  /** Another page was chosen. */
  'update:page': [page: number]
}>()

const page = useControllable(props, 'page', emit, () => props.defaultPage)
const t = computed<PaginationLabels>(() => ({ ...PAGINATION_LABELS, ...props.labels }))
const count = computed(() => Math.max(0, Math.ceil(props.total / Math.max(1, props.itemsPerPage))))

// The viewport's width class, as the `@media` fallback before mounting sees it.
const width = useViewportWidth()
const mounted = shallowRef(false)
onMounted(() => {
  mounted.value = true
})
// A WeftProvider's `width` says which form to draw before mounting; without
// one, both are drawn and CSS picks from the viewport until mounted.
const given = useWeft().width.value
const narrow = computed(() => width.value === 'narrow')
const both = computed(() => !mounted.value && given === undefined)
const entries = computed<PageSlot[]>(() =>
  both.value
    ? pageRangeBoth(page.value, count.value, props.siblingCount, props.showEdges)
    : pageRange(page.value, count.value, narrow.value ? Math.min(props.siblingCount, 1) : props.siblingCount, props.showEdges).map((entry) => ({ entry })),
)
const edgeOnly = computed(() => (both.value ? 'wide' : undefined))

function go(n: number) {
  if (props.disabled || n < 1 || n > count.value) return
  page.value = n
}
const href = (n: number) => (props.to && n >= 1 && n <= count.value ? props.to(n) : undefined)
</script>

<template>
  <nav class="weft-pagination" :aria-label="t.nav" :data-size="size">
    <ul class="weft-pagination__list">
      <li v-if="showControls && (both || !narrow)" :data-only="edgeOnly">
        <Button :variant="variant" :tone="tone" :size="size" square :aria-label="t.first" :to="href(1)" :disabled="disabled || page <= 1" @click="go(1)">
          <template #leading><svg class="weft-pagination__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m11 17-5-5 5-5" /><path d="m18 17-5-5 5-5" /></svg></template>
        </Button>
      </li>
      <li v-if="showControls">
        <Button :variant="variant" :tone="tone" :size="size" square :aria-label="t.previous" :to="href(page - 1)" :disabled="disabled || page <= 1" @click="go(page - 1)">
          <template #leading><svg class="weft-pagination__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg></template>
        </Button>
      </li>
      <li v-for="({ entry, only }, i) in entries" :key="entry === 'ellipsis' ? `e${i}` : entry" :data-only="only">
        <span v-if="entry === 'ellipsis'" class="weft-pagination__ellipsis" aria-hidden="true">…</span>
        <Button
          v-else
          class="weft-pagination__page"
          :label="String(entry)"
          :variant="entry === page ? activeVariant : variant"
          :tone="entry === page ? activeTone : tone"
          :size="size"
          :active="entry === page"
          :aria-label="t.page(entry)"
          :aria-current="entry === page ? 'page' : undefined"
          :to="href(entry)"
          :disabled="disabled"
          @click="go(entry)"
        />
      </li>
      <li v-if="showControls">
        <Button :variant="variant" :tone="tone" :size="size" square :aria-label="t.next" :to="href(page + 1)" :disabled="disabled || page >= count" @click="go(page + 1)">
          <template #leading><svg class="weft-pagination__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg></template>
        </Button>
      </li>
      <li v-if="showControls && (both || !narrow)" :data-only="edgeOnly">
        <Button :variant="variant" :tone="tone" :size="size" square :aria-label="t.last" :to="href(count)" :disabled="disabled || page >= count" @click="go(count)">
          <template #leading><svg class="weft-pagination__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m13 17 5-5-5-5" /><path d="m6 17 5-5-5-5" /></svg></template>
        </Button>
      </li>
    </ul>
  </nav>
</template>
