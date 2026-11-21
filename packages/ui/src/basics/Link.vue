<script setup vapor lang="ts">
// A link. One to another site gets rel="noopener"; one to the page the
// browser is on gets aria-current="page". The current page is known only in
// the browser, so it is marked after mounting: the server's render and the
// first client render agree, and an island hydrates without a mismatch. Pass
// `active` where the server knows (a layout's own navigation). Only a
// standalone link gets a touch hit area: in running text it would cover the
// lines above and below. In a disabled or busy Form it is disabled (a
// disabled fieldset leaves links alone), so it cannot navigate mid-submit.
import { computed, onMounted, ref } from 'vue'
import { useFormState } from '../forms/field.ts'
import { isCurrentPage, isExternal as looksExternal, relFor } from './address.ts'
import '../styles/base.css'
import '../styles/link.css'

const props = withDefaults(
  defineProps<{
    /** The address. */
    to?: string
    /** The address, for those who write links as HTML does; `to` wins. */
    href?: string
    /** Another site: adds rel="noopener". Unset, an address with a scheme and host (`https://…`, `//…`) is external. */
    external?: boolean
    /** Where to open it, as the native attribute: `_blank` for a new tab. */
    target?: string
    /** The native rel; `noopener` is added for an external link. */
    rel?: string
    /** This link is the current page: sets aria-current="page". Unset, the link compares its address with the browser's. */
    active?: boolean
    /** Unavailable: no address, so not focusable, and announced as a disabled link. */
    disabled?: boolean
    /** Not in running text (a list of links, a footer, a toolbar): drawn as an inline box whose hit area is at least --weft-hit-min on touch. In a sentence, leave it unset, so the hit area does not cover the lines around it. */
    standalone?: boolean
  }>(),
  { external: undefined, active: undefined },
)

const emit = defineEmits<{
  /** Followed by mouse, touch or keyboard; never while disabled. */
  click: [event: MouseEvent]
}>()

const address = computed(() => props.to ?? props.href)
const isExternal = computed(() => props.external ?? looksExternal(address.value))
const rel = computed(() => relFor(props.rel, isExternal.value))

const mounted = ref(false)
onMounted(() => {
  mounted.value = true
})
const current = computed(() => {
  if (props.active !== undefined) return props.active
  return mounted.value && !isExternal.value && isCurrentPage(address.value)
})

const form = useFormState()
const off = computed(() => props.disabled || !!form?.disabled())

function onClick(event: MouseEvent) {
  if (off.value) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  emit('click', event)
}
</script>

<template>
  <a
    class="weft-link"
    :href="off ? undefined : address"
    :target="off ? undefined : target"
    :rel="off ? undefined : rel"
    :role="off ? 'link' : undefined"
    :aria-disabled="off ? 'true' : undefined"
    :aria-current="current ? 'page' : undefined"
    :data-standalone="standalone ? 'true' : undefined"
    @click="onClick"
  ><slot /></a>
</template>
