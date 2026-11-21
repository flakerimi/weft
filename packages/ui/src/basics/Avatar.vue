<script setup vapor lang="ts">
// A person or thing as a small round picture. It falls back from the image
// (when there is none, or it fails to load) to initials from `alt`, then to an
// icon: `icon` if given, else a silhouette Avatar draws itself, so a build
// ships no icon drawing for a fallback it never names. Named by `alt`: the image's alt text, or the fallback's aria-label;
// with no alt it is decorative. Inside an AvatarGroup it takes the group's
// size, and hides itself when it is past the group's max.
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue'
import Icon from './Icon.vue'
import { useAvatarGroup } from './avatarGroup.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/avatar.css'

const props = withDefaults(
  defineProps<{
    /** The image's address. */
    src?: string
    /** Who or what it shows ("Ada Lovelace"): the image's alt, and the source of the initials. */
    alt?: string
    /** Text in place of the initials from alt, such as a short code. */
    text?: string
    /** A Lucide icon shown with neither image nor name; without it, a silhouette. */
    icon?: string
    /** `sm` 24 px, `md` 32 px, `lg` 40 px. In an AvatarGroup, the group's size wins. */
    size?: Size
  }>(),
  { size: 'md' },
)

const group = useAvatarGroup()
const size = computed(() => group?.size() ?? props.size)

const failed = ref(false)
watch(
  () => props.src,
  () => {
    failed.value = false
  },
)
// An image the server rendered may have failed before this listened: a
// finished image with no width is a broken one.
const img = useTemplateRef<HTMLImageElement>('img')
onMounted(() => {
  if (img.value?.complete && img.value.naturalWidth === 0) failed.value = true
})

const showImage = computed(() => !!props.src && !failed.value)
const initials = computed(() => {
  if (props.text) return props.text
  const words = (props.alt ?? '').trim().split(/\s+/).filter(Boolean)
  const first = (w: string | undefined) => (w ? Array.from(w)[0]!.toUpperCase() : '')
  return words.length > 1 ? first(words[0]) + first(words[words.length - 1]) : first(words[0])
})
</script>

<template>
  <span
    v-if="!group?.hidden.value"
    class="weft-avatar"
    :data-size="size"
    :role="!showImage && alt ? 'img' : undefined"
    :aria-label="!showImage && alt ? alt : undefined"
  >
    <img v-if="showImage" ref="img" class="weft-avatar__image" :src="src" :alt="alt ?? ''" @error="failed = true" />
    <span v-else-if="initials" class="weft-avatar__initials" aria-hidden="true">{{ initials }}</span>
    <Icon v-else-if="icon" :name="icon" />
    <svg
      v-else
      class="weft-icon weft-avatar__silhouette"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      aria-hidden="true"
    ><circle cx="12" cy="9" r="4" /><path d="M5 20a7 7 0 0 1 14 0" /></svg>
  </span>
</template>
