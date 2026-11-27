// The permission check Button and DataTable draw by, without the session's
// code: session.ts is loaded (through load.ts), and GET /api/auth/me asked,
// only once a component with `perm` has mounted and asks, so a page whose
// buttons have no `perm` fetches neither. Its chunk is the one every island
// shares.
import { onMounted, ref, type Ref } from 'vue'

type Can = (permission: string) => boolean
const no = () => false

/**
 * For a component that shows something by permission: a ref of `can`, which
 * says no until the component has mounted and its session is loaded. Two
 * things keep it so. The stand-in installed on mount says no and starts an
 * `import()` that resolves only after the render that called it, so the
 * browser's first render, hydration's included (even in an island that wakes
 * after the session is known), draws what the server drew. And the stand-in
 * is installed only on mount, so the server, which never mounts, never calls
 * `import()` at all (session-island-ssr.test.ts). Reactive; call it in setup,
 * and use it as `can(perm)` in a template or `can.value(perm)` in a script.
 */
export function useCan(): Ref<Can> {
  // No before mounting; then a stand-in that loads the session when asked, and says no until it has.
  // Not fetched (offline, say), the stand-in stays, and asks again the next time.
  const can = ref<Can>(no)
  onMounted(() => (can.value = () => (import('./load.ts').then((m) => (can.value = m.sessionCan()), no), false)))
  return can
}
