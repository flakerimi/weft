<script setup vapor lang="ts">
// The route guard for a page's islands: what it holds is shown only to a
// signed-in user who has `perm`. Pages are documents, built with no session,
// so the server renders `loading`, and so does the browser's first render;
// once mounted it calls requireSession, which sends a visitor with no
// session to sign in, with `next`. A user without `perm` sees `denied`, and
// a session that could not be read shows why, sending no one away. It hides
// what it holds; it guards nothing on the server, whose routes still check.
import { computed, onMounted, ref } from 'vue'
import Spinner from '../basics/Spinner.vue'
import Alert from '../feedback/Alert.vue'
import { requireSession, useSession, type SessionUser } from './session.ts'
import '../styles/base.css'

const props = defineProps<{
  /** A permission, such as `reports:read`, the user must have; without it, any signed-in user. */
  perm?: string
  /** Where a visitor with no session is sent, with `?next=` this page; `/login` by default. */
  signIn?: string
}>()

defineSlots<{
  /** What it guards, for a signed-in user with `perm`; `user` is who that is. */
  default?: (props: { user: SessionUser }) => unknown
  /** For a user without `perm`, in place of an Alert saying so. */
  denied?: () => unknown
  /** Until the session is known, in place of a Spinner; also on the server, and while a visitor is sent to sign in. */
  loading?: () => unknown
}>()

const { session, can } = useSession()
const mounted = ref(false)
onMounted(() => {
  mounted.value = true
  void requireSession({ perm: props.perm, signIn: props.signIn })
})

/** What to show: nothing is known before mounting, so the server and hydration agree. */
const shown = computed(() => {
  if (!mounted.value || session.loading) return 'loading'
  if (session.problem) return 'problem'
  if (!session.user) return 'loading'
  return props.perm && !can(props.perm) ? 'denied' : 'allowed'
})
</script>

<template>
  <slot v-if="shown === 'allowed'" :user="session.user!" />
  <slot v-else-if="shown === 'denied'" name="denied">
    <Alert tone="warning" title="Your account cannot see this; ask an administrator." />
  </slot>
  <Alert v-else-if="shown === 'problem'" role="alert" tone="danger" :title="session.problem" />
  <slot v-else name="loading"><Spinner label="Loading" /></slot>
</template>
