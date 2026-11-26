<script setup vapor lang="ts">
// An island in every page's header: the signed-in email and a sign-out
// button, or a link to sign in. Pages are built with no session, so the
// server draws nothing here and the browser asks.
import { ref } from 'vue'
import { useSession } from '../session.ts'

const { session, signOut } = useSession()
const leaving = ref(false)

async function leave() {
  leaving.value = true
  try {
    await signOut()
  } finally {
    leaving.value = false
  }
}

const here = () => (typeof location === 'undefined' ? '/' : location.pathname + location.search)
</script>

<template>
  <Stack direction="row" :gap="3" align="center">
    <template v-if="session.user">
      <span>{{ session.user.email }}</span>
      <Button label="Sign out" variant="ghost" size="sm" :loading="leaving" @click="leave" />
    </template>
    <Link v-else-if="!session.loading" :to="`/login?next=${encodeURIComponent(here())}`">Sign in</Link>
  </Stack>
</template>
