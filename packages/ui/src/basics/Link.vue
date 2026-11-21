<script setup vapor lang="ts">
// A link. One to another site gets rel="noopener"; one to the page the
// browser is on gets aria-current="page". The current page is known only in
// the browser, so it is marked after mounting: the server's render and the
// first client render agree, and an island hydrates without a mismatch. Pass
// `active` where the server knows (a layout's own navigation).
import { computed, onMounted, ref } from 'vue'
import { isExternal as looksExternal, relFor } from './address.ts'

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
const trim = (path: string) => (path.length > 1 ? path.replace(/\/+$/, '') : path)
const current = computed(() => {
  if (props.active !== undefined) return props.active
  if (!mounted.value || isExternal.value || !address.value) return false
  const url = new URL(address.value, location.href)
  return url.origin === location.origin && trim(url.pathname) === trim(location.pathname)
})

function onClick(event: MouseEvent) {
  if (props.disabled) {
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
    :href="disabled ? undefined : address"
    :target="target"
    :rel="disabled ? undefined : rel"
    :role="disabled ? 'link' : undefined"
    :aria-disabled="disabled ? 'true' : undefined"
    :aria-current="current ? 'page' : undefined"
    @click="onClick"
  ><slot /></a>
</template>
