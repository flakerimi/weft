<script setup vapor lang="ts">
// An island on /account, for whoever is signed in: their name, a new
// password, the sessions they are signed in with (each can be ended, or all
// at once), and their API keys when the app takes them (API_KEYS=on). A new
// key's secret is shown once, as the server answers it once.
import { onMounted, reactive, ref, useTemplateRef } from 'vue'
import { sessionEnded, useSession, wireTimeText, type FormApi, type TableColumn } from '@weft/ui'
import { api, schema, type APIKey, type ChangePasswordInput, type KeyInput, type NewKey, type ProfileInput, type SessionInfo } from '../.weft-types/api.ts'
import { formProblem } from '../answers.ts'

const { refresh, signOut } = useSession()
const when = (v: unknown) => wireTimeText(v, 'datetime')
const roles = ['owner', 'administrator', 'member', 'viewer']

// The name.
const profile = reactive<ProfileInput>({ name: '' })
const profileForm = useTemplateRef<FormApi>('profileForm')
const profileSaid = ref('')
async function saveProfile(values: ProfileInput) {
  profileSaid.value = ''
  try {
    await api.auth.updateMe(values)
    await refresh()
    profileSaid.value = 'Saved.'
  } catch (e) {
    const { fields, message } = formProblem(e, sessionEnded)
    profileForm.value?.setErrors(fields)
    profileSaid.value = message
  }
}

// A new password, which signs out every other session.
const change = reactive<ChangePasswordInput>({ current: '', password: '' })
const changeForm = useTemplateRef<FormApi>('changeForm')
const changeSaid = ref('')
const changed = ref(false)
async function changePassword(values: ChangePasswordInput) {
  changeSaid.value = ''
  changed.value = false
  try {
    await api.auth.changePassword(values)
    Object.assign(change, { current: '', password: '' })
    changed.value = true
    await loadSessions()
  } catch (e) {
    const { fields, message } = formProblem(e, sessionEnded)
    changeForm.value?.setErrors(fields)
    changeSaid.value = message
  }
}

// The sessions.
const sessions = ref<SessionInfo[]>([])
const sessionsSaid = ref('')
const sessionColumns: TableColumn<SessionInfo>[] = [
  { key: 'agent', label: 'Browser', rowHeader: true },
  { key: 'address', label: 'Address' },
  { key: 'created_at', label: 'Signed in', format: when },
  { key: 'seen_at', label: 'Last seen', format: when },
  { key: 'id', label: '' },
]
async function loadSessions() {
  try {
    sessions.value = await api.auth.sessions()
  } catch (e) {
    sessionsSaid.value = formProblem(e, sessionEnded).message
  }
}
// Why signing out did not happen, such as a server out of reach.
const signOutSaid = ref('')
async function leave() {
  signOutSaid.value = await signOut()
}
async function endSession(s: SessionInfo) {
  sessionsSaid.value = ''
  try {
    await api.auth.endSession(s.id)
    await loadSessions()
  } catch (e) {
    sessionsSaid.value = formProblem(e, sessionEnded).message
  }
}
async function signOutEverywhere() {
  sessionsSaid.value = ''
  try {
    await api.auth.logoutEverywhere()
    location.assign('/login')
  } catch (e) {
    sessionsSaid.value = formProblem(e, sessionEnded).message
  }
}

// The API keys, when the app takes them.
const keysOn = ref(false)
const keys = ref<APIKey[]>([])
const keysSaid = ref('')
const made = ref<NewKey | null>(null)
const copied = ref(false)
const key = reactive<KeyInput>({ name: '', role: 'viewer', days: 90 })
const keyForm = useTemplateRef<FormApi>('keyForm')
const keyColumns: TableColumn<APIKey>[] = [
  { key: 'name', label: 'Name', rowHeader: true },
  { key: 'prefix', label: 'Starts' },
  { key: 'role', label: 'Role' },
  { key: 'seen_at', label: 'Last used', format: (v) => (v ? when(v) : 'Never') },
  { key: 'expires_at', label: 'Expires', format: when },
  { key: 'id', label: '' },
]
async function loadKeys() {
  try {
    keys.value = await api.auth.keys()
  } catch (e) {
    keysSaid.value = formProblem(e, sessionEnded).message
  }
}
async function makeKey(values: KeyInput) {
  keysSaid.value = ''
  copied.value = false
  try {
    made.value = await api.auth.createKey(values)
    Object.assign(key, { name: '', role: 'viewer', days: 90 })
    await loadKeys()
  } catch (e) {
    const { fields, message } = formProblem(e, sessionEnded)
    keyForm.value?.setErrors(fields)
    keysSaid.value = message
  }
}
async function removeKey(k: APIKey) {
  keysSaid.value = ''
  try {
    await api.auth.removeKey(k.id)
    await loadKeys()
  } catch (e) {
    keysSaid.value = formProblem(e, sessionEnded).message
  }
}
async function copySecret() {
  if (!made.value) return
  await navigator.clipboard.writeText(made.value.secret)
  copied.value = true
}

onMounted(async () => {
  const user = await useSession().ready
  if (!user) return
  profile.name = user.name
  await loadSessions()
  try {
    keysOn.value = (await api.auth.settings()).api_keys
  } catch {
    // Without the setting, no keys section: the rest of the page works.
  }
  if (keysOn.value) await loadKeys()
})
</script>

<template>
  <Guard>
    <template #default="{ user }">
      <Stack :gap="8">
        <Section :title="user.email" :description="`Signed in as ${user.role}.`">
          <template #actions>
            <Button label="Sign out" variant="soft" @click="leave" />
          </template>
          <Alert v-if="signOutSaid" tone="danger" :title="signOutSaid" />
          <Form ref="profileForm" :state="profile" :schema="schema.ProfileInput" @submit="saveProfile">
            <Stack :gap="4">
              <FormField label="Name" name="name">
                <Input v-model="profile.name" autocomplete="name" />
              </FormField>
              <Stack direction="row" :gap="3" align="center">
                <Button type="submit" label="Save the name" />
                <span v-if="profileSaid" role="status">{{ profileSaid }}</span>
              </Stack>
            </Stack>
          </Form>
        </Section>

        <Section title="Password" description="Changing it signs out every other session; your API keys keep working.">
          <Form ref="changeForm" :state="change" :schema="schema.ChangePasswordInput" @submit="changePassword">
            <Stack :gap="4">
              <Alert v-if="changed" tone="success" title="Your password is changed, and your other sessions are signed out." />
              <Alert v-if="changeSaid" tone="danger" :title="changeSaid" />
              <FormField label="Current password" name="current">
                <Input v-model="change.current" type="password" autocomplete="current-password" />
              </FormField>
              <FormField label="New password" name="password" description="At least 8 characters.">
                <Input v-model="change.password" type="password" autocomplete="new-password" />
              </FormField>
              <Button type="submit" label="Change the password" />
            </Stack>
          </Form>
        </Section>

        <Section title="Sessions" description="Where you are signed in, newest first.">
          <Stack :gap="4">
            <Alert v-if="sessionsSaid" tone="danger" :title="sessionsSaid" />
            <Table :columns="sessionColumns" :rows="sessions" caption="Sessions" hide-caption empty="No sessions.">
              <template #id-cell="{ row }">
                <Badge v-if="row.current" label="This one" variant="soft" />
                <Button v-else label="End" variant="ghost" size="sm" @click="endSession(row)" />
              </template>
            </Table>
            <div>
              <Button label="Sign out everywhere" tone="danger" variant="soft" @click="signOutEverywhere" />
            </div>
          </Stack>
        </Section>

        <Section v-if="keysOn" title="API keys" description="For other programs: each acts as you, with its role or yours, whichever is lower.">
          <Stack :gap="4">
            <Alert v-if="made" tone="success" title="Copy the key now; it is not shown again." :description="made.secret">
              <template #actions>
                <Button :label="copied ? 'Copied' : 'Copy'" icon="copy" size="sm" @click="copySecret" />
              </template>
            </Alert>
            <Alert v-if="keysSaid" tone="danger" :title="keysSaid" />
            <Table :columns="keyColumns" :rows="keys" caption="API keys" hide-caption empty="No API keys.">
              <template #id-cell="{ row }">
                <Button label="Remove" variant="ghost" size="sm" tone="danger" @click="removeKey(row)" />
              </template>
            </Table>
            <Form ref="keyForm" :state="key" :schema="schema.KeyInput" @submit="makeKey">
              <Stack :gap="4">
                <FormField label="Name" name="name" description="What the key is for, such as the nightly export.">
                  <Input v-model="key.name" />
                </FormField>
                <FormField label="Role" name="role">
                  <Select v-model="key.role" :items="roles" />
                </FormField>
                <FormField label="Days" name="days" description="How long it works, from 1 to 365.">
                  <InputNumber v-model="key.days" :min="1" :max="365" />
                </FormField>
                <Button type="submit" label="Make a key" />
              </Stack>
            </Form>
          </Stack>
        </Section>
      </Stack>
    </template>
  </Guard>
</template>
