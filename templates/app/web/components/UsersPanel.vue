<script setup vapor lang="ts">
// An island on /users, for an account with users:read (owners and
// administrators): every user, with a dialog to change one's role
// (users:edit) and Delete to remove one (users:delete); and, for
// users:invite, the invitations not yet taken, with New invitation, which
// mails a link, and Delete. The server decides: a role cannot reach above
// the actor's, and no one changes their own role or removes themselves, so
// the signed-in user's own row offers neither Edit nor Delete.
import { useTemplateRef } from 'vue'
import type { TableColumn } from '@weft/ui'
import { sessionEnded, useSession, wireTimeText } from '@weft/ui'
import { ApiError, api, schema, type Invitation, type User } from '../.weft-types/api.ts'

const { session } = useSession()
/** The users table's own Edit and Delete, which the row actions below call. */
const table = useTemplateRef<{ edit(row: User): void; askRemove(row: User, index?: number): void }>('table')

const roles = ['owner', 'administrator', 'member', 'viewer']
const when = (v: unknown) => wireTimeText(v, 'datetime')

const users = { list: api.auth.users, update: api.auth.updateUser, remove: api.auth.removeUser }
const userColumns: TableColumn<User>[] = [
  { key: 'email', label: 'Email', sortable: true, rowHeader: true },
  { key: 'name', label: 'Name', sortable: true },
  { key: 'role', label: 'Role', sortable: true },
  { key: 'created_at', label: 'Joined', sortable: true, format: when },
]

const invitations = { list: api.auth.invitations, create: api.auth.invite, remove: api.auth.removeInvitation }
const invitationColumns: TableColumn<Invitation>[] = [
  { key: 'email', label: 'Email', rowHeader: true },
  { key: 'role', label: 'Role' },
  { key: 'expires_at', label: 'Link works until', format: when },
]

// A list that answers 401 means the session ended while the page was open.
function listFailed(e: unknown) {
  if (e instanceof ApiError && e.status === 401) void sessionEnded()
}
</script>

<template>
  <Guard perm="users:read">
    <Stack :gap="8">
      <Section title="Users">
        <DataTable
          ref="table"
          :resource="users"
          :schema="schema.RoleInput"
          :columns="userColumns"
          perm="users"
          name="user"
          label-key="email"
          caption="Users"
          hide-caption
          :per-page="20"
          default-sort="email"
          search-placeholder="Email or name"
          :filter-defs="[{ key: 'role', label: 'Role', items: roles }]"
          :fields="[{ name: 'role' }]"
          empty-title="No users"
          empty-description="Change the search, or invite someone below."
          empty-icon="users"
          @error="listFailed"
        >
          <template #actions-cell="{ row, index }">
            <Stack v-if="row.id !== session.user?.id" direction="row" :gap="2" justify="end">
              <Button label="Edit" :aria-label="`Edit ${row.email}`" icon="pencil" variant="ghost" size="sm" perm="users:edit" @click="table?.edit(row)" />
              <Button label="Delete" :aria-label="`Delete ${row.email}`" icon="trash" variant="ghost" tone="danger" size="sm" perm="users:delete" @click="table?.askRemove(row, index)" />
            </Stack>
          </template>
        </DataTable>
      </Section>
      <Guard perm="users:invite">
        <Section title="Invitations" description="Each mails a link that sets a password and signs in; it works for 7 days.">
          <DataTable
            :resource="invitations"
            :schema="schema.InviteInput"
            :columns="invitationColumns"
            name="invitation"
            caption="Invitations"
            hide-caption
            :per-page="20"
            search-placeholder="Email"
            :defaults="{ role: 'member' }"
            :fields="[{ name: 'email' }, { name: 'role' }]"
            empty-title="No invitations"
            empty-description="Invite someone with New invitation."
            empty-icon="mail"
            @error="listFailed"
          />
        </Section>
        <template #denied />
        <template #loading />
      </Guard>
    </Stack>
  </Guard>
</template>
