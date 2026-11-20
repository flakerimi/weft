<script setup vapor lang="ts">
import { Card, Divider, Grid, Page, Section, Stack } from '../../src/index.ts'
defineProps<{ case: string }>()
</script>

<template>
  <Stack v-if="$props.case === 'stack-default'"><i>a</i></Stack>
  <Stack v-else-if="$props.case === 'stack-row'" direction="row" :gap="2" wrap align="center" justify="between"><i>a</i></Stack>
  <Grid v-else-if="$props.case === 'grid-default'"><i>a</i></Grid>
  <Grid v-else-if="$props.case === 'grid-fixed'" :cols="4"><i>a</i></Grid>
  <Page v-else-if="$props.case === 'page'" title="Customers" description="Everyone who pays">
    <template #actions><button class="act">New</button></template>
    <p class="body">rows</p>
  </Page>
  <Section v-else-if="$props.case === 'section'" title="Billing"><p>x</p></Section>
  <Card v-else-if="$props.case === 'card-plain'"><p class="card-body">x</p></Card>
  <Card v-else-if="$props.case === 'card-header'" :padding="6">
    <template #header>Plan</template>
    <template #actions><button>Edit</button></template>
    <template #footer>Renews monthly</template>
    <p>x</p>
  </Card>
  <Divider v-else-if="$props.case === 'divider'" />
  <Divider v-else-if="$props.case === 'divider-vertical'" orientation="vertical" />
</template>
