// How an AvatarGroup and its Avatars talk. Each Avatar joins the group when it
// is set up, in document order, and leaves when it unmounts; the group hides
// those past `max` and counts them. Set-up order is render order on the server
// and on the client alike, so the count hydrates as rendered. An avatar that
// mounts later joins at the end, wherever it renders.
import { computed, inject, onScopeDispose, provide, shallowRef, type ComputedRef, type InjectionKey } from 'vue'
import type { Size } from '../types.ts'

interface AvatarGroupContext {
  size: () => Size
  /** Join as the next avatar; true while this avatar is past the group's max. */
  join(): ComputedRef<boolean>
}

const key: InjectionKey<AvatarGroupContext> = Symbol('weft-avatar-group')

/** For AvatarGroup: provide the group, and how many avatars it hides. */
export function provideAvatarGroup(size: () => Size, max: () => number | undefined): ComputedRef<number> {
  const members = shallowRef<symbol[]>([])
  const limit = () => {
    const m = max()
    return m !== undefined && m > 0 ? m : Infinity
  }
  provide(key, {
    size,
    join() {
      const id = Symbol('avatar')
      members.value = [...members.value, id]
      onScopeDispose(() => {
        members.value = members.value.filter((m) => m !== id)
      })
      return computed(() => members.value.indexOf(id) >= limit())
    },
  })
  return computed(() => Math.max(0, members.value.length - limit()))
}

/** For Avatar: the group it is in, if any, with the size it imposes and whether this avatar is hidden. */
export function useAvatarGroup(): { size: () => Size; hidden: ComputedRef<boolean> } | null {
  const group = inject(key, null)
  return group ? { size: group.size, hidden: group.join() } : null
}
