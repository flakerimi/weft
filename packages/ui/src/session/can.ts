// The permission check Button and DataTable draw by, without the session's
// code: session.ts is loaded, and GET /api/auth/me asked, only once a
// component with `perm` has mounted and asks, so a page whose buttons have
// no `perm` fetches neither. Its chunk is the one every island shares.
import { onMounted, shallowRef } from 'vue'

type Can = (permission: string) => boolean

/**
 * For a component that shows something by permission: `can`, but false until
 * the component has mounted and the session is known. The server renders it
 * signed out, and so does hydration's first render, even in an island that
 * wakes after the session is known, so the two always agree. Reactive; call
 * it in setup.
 */
export function useCan(): Can {
  // false before mounting; then a stand-in that loads the session when asked, and says no until it has. Not
  // fetched (offline, say), it stays, and asks again the next time.
  const can = shallowRef<Can | false>(false)
  onMounted(() => (can.value = () => (import('./session.ts').then((m) => (can.value = m.useSession().can), () => {}), false)))
  return (permission) => !!can.value && can.value(permission)
}
