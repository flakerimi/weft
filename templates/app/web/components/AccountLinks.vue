<script setup vapor lang="ts">
// An island on the front page: for whoever is signed in, their address, a
// link to /account, one to /users for an account with users:read, and Sign
// out. Pages are built with no session, so the server draws none of it.
import { ref } from 'vue'
import { useSession } from '@weft/ui'

const { session, signOut } = useSession()
const problem = ref('')
</script>

<template>
  <Stack v-if="session.user" direction="row" :gap="4" align="center" wrap>
    <span>{{ session.user.email }}</span>
    <Link to="/account">Account</Link>
    <Button to="/users" label="Users" variant="ghost" size="sm" perm="users:read" />
    <Button label="Sign out" variant="ghost" size="sm" @click="async () => (problem = await signOut())" />
    <span v-if="problem" role="alert">{{ problem }}</span>
  </Stack>
</template>
