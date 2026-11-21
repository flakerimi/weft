<script setup vapor lang="ts">
// A scene: what happened on the network this morning, a Timeline whose dots
// are Avatars: who did it, or a mark for what the network did itself.
import { Avatar, Timeline, type TimelineItem } from '@weft/ui'

const events: (TimelineItem & { mark: string })[] = [
  { value: 'moved', title: 'Port 1/3/1 moved to the new OLT', description: 'PPPoE kept for 31 customers', date: '09:12', mark: 'FI' },
  { value: 'closed', title: 'Ticket #418 closed', description: 'Signal back to −20.1 dBm', date: '08:40', mark: 'AB' },
  { value: 'offline', title: 'Gashi M. went offline', description: 'Last seen at 07:55', date: '07:55', mark: '!' },
]
</script>

<template>
  <Timeline :items="events" class="scene-activity">
    <template #indicator="{ item }"><Avatar :text="(item as (typeof events)[number]).mark" size="sm" /></template>
  </Timeline>
</template>
