<script setup vapor lang="ts">
// An island in every page's header: the signed-in email, links to the
// account and, for users:read, the users, and a sign-out button; or a link
// to sign in. Pages are built with no session, so the server draws nothing
// here and the browser asks.
import { computed, ref } from 'vue'
import { useSession } from '@weft/ui'

const { session, signOut } = useSession()
const leaving = ref(false)
// Why signing out did not happen, such as a server out of reach: the visitor
// is still signed in, and is told so.
const problem = ref('')

async function leave() {
  leaving.value = true
  problem.value = await signOut()
  leaving.value = false
}

// On /login, /setup and the pages that ask for or use a mailed link there is
// nothing to come back to, and a link from /login would wrap its own next in
// another one.
const signIn = computed(() => {
  if (typeof location === 'undefined' || ['/login', '/setup', '/register', '/forgot', '/password'].includes(location.pathname)) return ''
  return `/login?next=${encodeURIComponent(location.pathname + location.search)}`
})
</script>

<template>
  <Stack direction="row" :gap="3" align="center">
    <template v-if="session.user">
      <Link to="/account">{{ session.user.email }}</Link>
      <Button to="/users" label="Users" variant="ghost" size="sm" perm="users:read" />
      <span v-if="problem" role="alert">{{ problem }}</span>
      <Button label="Sign out" variant="ghost" size="sm" :loading="leaving" @click="leave" />
    </template>
    <Link v-else-if="!session.loading && !session.problem && signIn" :to="signIn">Sign in</Link>
  </Stack>
</template>
