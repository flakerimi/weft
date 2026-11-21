<script setup vapor lang="ts">
// A search field over a list of commands in groups: type to find one
// fuzzily (the matched letters marked), the arrows move across groups
// (skipping disabled and loading commands), Enter runs the active one, a
// press runs one. A command with children opens them as a page of their
// own, with a back button; Backspace in the empty field goes back. With
// `recent`, the commands run last are listed first while the search is
// empty, kept in localStorage under `recent-key`. The field keeps focus and
// names the active command with aria-activedescendant (WAI-ARIA APG
// Combobox, its list always shown). Usually opened from a Dialog, a Cmd+K
// launcher: the Dialog's focus trap moves focus into the field, Escape
// closes the Dialog, and the app closes it on `select`.
import { computed, nextTick, onMounted, ref, useTemplateRef } from 'vue'
import Kbd from '../basics/Kbd.vue'
import Spinner from '../basics/Spinner.vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import CloseButton from '../overlays/CloseButton.vue'
import { commandsByKey, keyOf, searchGroups, type CommandGroup, type CommandItem } from './command.ts'
import { segments } from './fuzzy.ts'
import type { SelectItem, SelectOption } from './items.ts'
import { listProps, provideListbox, useListbox } from './listbox.ts'
import Listbox from './Listbox.vue'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/command-palette.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The commands, in groups: `{ id, label, items, ignoreFilter?, postFilter? }`. */
    groups?: CommandGroup[]
    /** The search, bound with `v-model:search-term`. */
    searchTerm?: string
    /** The search field's placeholder; a page of children shows its command's `placeholder`. */
    placeholder?: string
    /** At most this many matches per group while searching. */
    limit?: number
    /** Keep the groups in their order while searching, instead of best match first. */
    preserveGroupOrder?: boolean
    /** List the commands run last first while the search is empty: true keeps 5, a number that many. */
    recent?: boolean | number
    /** Keep the recent commands in localStorage under this key (`weft:recent:<key>`), so they outlast the page. */
    recentKey?: string
    /** The recent group's heading. */
    recentLabel?: string
    /** Results are on their way: a spinner in the field, aria-busy on the list. */
    loading?: boolean
    /** A close button in the field (emits `update:open` with false), for a palette in a Dialog. */
    close?: boolean
    /** A back button in a page of children. */
    back?: boolean
    /** Focus the search field once mounted. */
    autofocus?: boolean
    /** Nothing can be run. */
    disabled?: boolean
    /** The arrows wrap round at the ends. */
    loop?: boolean
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** Draw only the rows in view; by default past 200 commands. */
    virtualize?: boolean
  }>(),
  {
    groups: () => [],
    searchTerm: undefined,
    limit: 12,
    recentLabel: 'Recent',
    back: true,
    size: 'md',
    virtualize: undefined,
  },
)

const emit = defineEmits<{
  /** A command was run (after its own `onSelect`). */
  select: [item: CommandItem, event: Event]
  /** The search changed. */
  'update:searchTerm': [term: string]
  /** The close button was pressed: false. */
  'update:open': [open: boolean]
  /** The active command changed, or undefined. */
  highlight: [item: CommandItem | undefined]
}>()

defineSlots<{
  /** A command's label, in place of its marked label, prefix and suffix. */
  'item-label'?: (props: { item: CommandItem }) => unknown
  /** After a command's label, before its shortcut. */
  'item-trailing'?: (props: { item: CommandItem }) => unknown
  /** Shown when no command matches: by default "No results". */
  empty?: (props: { searchTerm: string }) => unknown
  /** Under the list (hints on the keys). */
  footer?: () => unknown
}>()

const term = useControllable(props, 'searchTerm', emit, '')
const listId = useId('command-list')
const input = useTemplateRef<HTMLInputElement>('input')
/** The pages of children opened, the latest last. */
const pages = ref<CommandItem[]>([])
const page = computed(() => pages.value[pages.value.length - 1])
const recentKeys = ref<string[]>([])
const keep = computed(() => (props.recent === true ? 5 : typeof props.recent === 'number' ? props.recent : 0))
const storageKey = computed(() => (props.recentKey ? `weft:recent:${props.recentKey}` : undefined))

const shownGroups = computed<CommandGroup[]>(() => {
  if (page.value) return [{ id: 'page', label: page.value.label, items: page.value.children }]
  const all = props.groups
  if (!keep.value || term.value.trim() || !recentKeys.value.length) return all
  const byKey = commandsByKey(all)
  const items = recentKeys.value.map((k) => byKey.get(k)).filter((i): i is CommandItem => !!i)
  return items.length ? [{ id: 'recent', label: props.recentLabel, items }, ...all] : all
})
const found = computed(() => searchGroups(shownGroups.value, term.value, { limit: props.limit, preserveGroupOrder: !!props.preserveGroupOrder }))

// The list draws each command as an option: a copy that carries the command,
// disabled while it is loading.
const SOURCE = Symbol('command')
type Option = SelectOption & { [SOURCE]: CommandItem }
const items = computed(() =>
  found.value.groups.map(({ group, items: list }) => [
    ...(group.label ? [{ type: 'label' as const, label: group.label }] : []),
    ...list.map((item): Option => ({ label: item.label, description: item.description, icon: item.icon, disabled: !!item.disabled || !!item.loading, [SOURCE]: item })),
  ]),
)
const source = (item: SelectItem) => (item as Option)[SOURCE]

const unused = ref<unknown>(undefined)
const list = useListbox(
  listProps(props, { items: () => items.value as SelectItem[][], ignoreFilter: () => true }),
  unused,
  {
    id: listId,
    search: () => term.value,
    touch: () => false,
    autoHighlight: () => true,
    selectsActive: true,
    label: () => ({ 'aria-label': page.value?.label ?? 'Commands' }),
    onChoose: (c, event) => c.kind === 'option' && run(source(c.item), event),
    onHighlight: (c) => emit('highlight', c?.kind === 'option' ? source(c.item) : undefined),
  },
)
provideListbox(list)

function remember(item: CommandItem) {
  if (!keep.value) return
  const k = keyOf(item)
  recentKeys.value = [k, ...recentKeys.value.filter((x) => x !== k)].slice(0, keep.value)
  if (storageKey.value) {
    try {
      localStorage.setItem(storageKey.value, JSON.stringify(recentKeys.value))
    } catch {
      // Storage full or off: the list lasts as long as the page.
    }
  }
}
onMounted(() => {
  if (storageKey.value) {
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey.value) ?? '[]')
      if (Array.isArray(saved)) recentKeys.value = saved.filter((k): k is string => typeof k === 'string').slice(0, keep.value || undefined)
    } catch {
      // Unreadable: start afresh.
    }
  }
  if (props.autofocus) input.value?.focus()
})

function follow(to: string, target?: string) {
  const a = document.createElement('a')
  a.href = to
  if (target) a.target = target
  if (target === '_blank') a.rel = 'noopener'
  a.click()
}
function run(item: CommandItem, event: Event) {
  if (props.disabled || item.disabled || item.loading) return
  if (item.children?.length) {
    pages.value = [...pages.value, item]
    term.value = ''
    void nextTick(() => list.highlight('first'))
    return
  }
  item.onSelect?.(event)
  remember(item)
  emit('select', item, event)
  if (item.to && !event.defaultPrevented) follow(item.to, item.target)
}
function goBack() {
  if (!pages.value.length) return
  pages.value = pages.value.slice(0, -1)
  term.value = ''
  input.value?.focus()
  void nextTick(() => list.highlight('first'))
}

function onInput(event: Event) {
  term.value = (event.target as HTMLInputElement).value
}
function onKeydown(event: KeyboardEvent) {
  if (props.disabled) return
  if (event.key === 'Backspace' && term.value === '' && pages.value.length) {
    event.preventDefault()
    goBack()
    return
  }
  if (list.keydown(event, { edges: false, typeahead: false, space: false })) event.preventDefault()
}

const marked = (item: SelectItem) => {
  const c = source(item)
  return segments(c.label, found.value.marks.get(c) ?? [])
}

defineExpose({
  /** Focus the search field. */
  focus: () => input.value?.focus(),
  /** Leave the open page of children. */
  back: goBack,
})
</script>

<template>
  <div class="weft-command-palette" :class="$attrs.class" :style="$attrs.style as string | undefined" :data-size="size">
    <div class="weft-command-palette__search">
      <button v-if="back && pages.length" type="button" class="weft-command-palette__back" aria-label="Back" @click="goBack">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg>
      </button>
      <svg v-else class="weft-command-palette__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" /></svg>
      <input
        ref="input"
        class="weft-command-palette__input"
        type="text"
        role="combobox"
        autocomplete="off"
        spellcheck="false"
        aria-expanded="true"
        aria-autocomplete="list"
        :aria-label="($attrs['aria-label'] as string | undefined) ?? page?.placeholder ?? placeholder ?? 'Search commands'"
        :placeholder="page?.placeholder ?? placeholder ?? 'Type a command or search…'"
        :value="term"
        :disabled="disabled || undefined"
        :aria-controls="listId"
        :aria-activedescendant="list.activeId.value"
        @input="onInput"
        @keydown="onKeydown"
      />
      <Spinner v-if="loading" />
      <CloseButton v-if="close" @click="emit('update:open', false)" />
    </div>
    <Listbox>
      <template #item-label="{ item }">
        <slot name="item-label" :item="source(item)">
          <span v-if="source(item).prefix" class="weft-command-palette__prefix">{{ source(item).prefix }}</span>
          <span class="weft-command-palette__label"><template v-for="(s, i) in marked(item)" :key="i"><mark v-if="s.match">{{ s.text }}</mark><template v-else>{{ s.text }}</template></template></span>
          <span v-if="source(item).suffix" class="weft-command-palette__suffix">{{ source(item).suffix }}</span>
        </slot>
      </template>
      <template #item-trailing="{ item }">
        <slot name="item-trailing" :item="source(item)" />
        <span v-if="source(item).kbds?.length" class="weft-command-palette__kbds"><Kbd v-for="k in source(item).kbds" :key="k" :value="k" size="sm" /></span>
        <Spinner v-if="source(item).loading" />
        <svg v-else-if="source(item).children?.length" class="weft-command-palette__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg>
      </template>
      <template v-if="$slots.empty" #empty="p"><slot name="empty" v-bind="p" /></template>
    </Listbox>
    <div v-if="$slots.footer" class="weft-command-palette__footer"><slot name="footer" /></div>
  </div>
</template>
