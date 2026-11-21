/**
 * useListbox(props, model, opts): the state of a WAI-ARIA APG listbox,
 * shared by Listbox, Select, Combobox and CommandPalette. The component
 * that owns the list (the one whose element has focus: a Select's button, a
 * Combobox's input, a Listbox itself) calls it, and the Listbox component
 * draws it (`provideListbox`, read by the Listbox inside, which draws that
 * state instead of its own).
 *
 * - Rows: `props.items` (see items.ts), filtered by `opts.search()` unless
 *   `props.ignoreFilter` (an app that filters itself, as async search does),
 *   become headings, separators and options, plus a "create" option for the
 *   search when `props.createItem` asks (Nuxt UI's `createItem`: when
 *   nothing matches, or `'always'` unless an item is the search; at the
 *   `top` or `bottom`). Options count from 0 in order; `id`s are stable per
 *   item (`<list id>-<group>-<item>`), so filtering keeps them.
 * - One option is active (highlighted): the element with focus carries
 *   `aria-activedescendant` = `activeId`, so focus never leaves it. The
 *   active option survives filtering while it is still shown; when the
 *   search changes, or `opts.autoHighlight()` wants one always, the first
 *   enabled option becomes active.
 * - `keydown(event, { edges, typeahead, space })` takes the APG keys, and
 *   returns true when it took one (the caller prevents its default):
 *   ArrowDown and ArrowUp move (looping when `props.loop`; from nothing
 *   they go to the first and the last), PageDown and PageUp move ten,
 *   Enter chooses the active option, Home and End go to the ends when
 *   `edges` (not in a text field, whose caret they move), Space chooses when
 *   `space`, and when `typeahead`, typing jumps to an option by its label
 *   (useTypeahead). Disabled options are skipped. With `range` and
 *   multiple, the APG's recommended keys: Shift+Arrow moves and toggles,
 *   Shift+Space chooses from the anchor (the last option chosen) to the
 *   active one, Ctrl+A all (or none), Ctrl+Arrow moves only, Ctrl+Space
 *   toggles; `selectionBehavior: 'replace'` makes the choice follow the
 *   arrows. `press(option, event)` does the same for a press (Shift, Ctrl).
 * - `choose(option, event)`: a disabled option does nothing; the create
 *   option calls `opts.onCreate(search)`; otherwise single selection sets
 *   the model to the option's value and multiple toggles it in the model's
 *   array (or, `replace`, makes it the only one); then the item's own
 *   `onSelect(event)`, then `opts.onChoose(option, event, changed)`.
 * - Long lists: past 200 options (or when `props.virtualize` says so) only
 *   the rows in view are rendered (useVirtualList), every row `rowHeight`
 *   px tall: the estimate (from the size, 48 on touch, or
 *   `virtualize.estimateSize`), grown to the tallest row the Listbox
 *   measured (`measure`), measured again from the estimate when it changes
 *   or the Listbox asks (`remeasure`: its width or the fonts changed), so
 *   rows can get smaller too. Separators are left out, each option knows its
 *   place in its group (`posInSet`, `setSize`), and the active option is
 *   always rendered and scrolled into view.
 */
import { computed, inject, nextTick, provide, ref, shallowRef, watch, type ComputedRef, type InjectionKey, type Ref, type ShallowRef } from 'vue'
import { useTypeahead } from '../headless/useTypeahead.ts'
import { useVirtualList, type VirtualList } from '../headless/useVirtualList.ts'
import type { Size } from '../types.ts'
import {
  filterPlaces,
  fold,
  get,
  groupsOf,
  isObject,
  isStructural,
  labelOf,
  sameValue,
  valueOf,
  type By,
  type SelectItem,
  type SelectItems,
  type SelectOption,
} from './items.ts'

/** Offer the search as a new item: when nothing matches (true), or `'always'` unless an item is the search. */
export type CreateItem = boolean | 'always' | { position?: 'top' | 'bottom'; when?: 'empty' | 'always' }

/** The props of a component that owns a listbox; each is optional. */
export interface ListboxProps {
  items?: SelectItems
  multiple?: boolean
  valueKey?: string
  labelKey?: string
  descriptionKey?: string
  by?: By
  filterFields?: string[]
  ignoreFilter?: boolean
  createItem?: CreateItem
  loop?: boolean
  disabled?: boolean
  loading?: boolean
  /** Loading the items failed with this (a Combobox's `load`); undefined while not. */
  error?: unknown
  /** True or false forces it; an object turns it on with `estimateSize` (the first row height, px) and `overscan` (rows drawn past each edge). */
  virtualize?: boolean | { estimateSize?: number; overscan?: number }
  /** With multiple: `toggle`, choosing toggles; `replace`, the selection follows the arrows and a press, and Ctrl toggles. */
  selectionBehavior?: 'toggle' | 'replace'
  size?: Size
}

export interface OptionRow {
  kind: 'option'
  key: string
  id: string
  /** Its place among the options. */
  index: number
  /** Its place among the rows. */
  row: number
  item: SelectItem
  value: unknown
  label: string
  description?: string
  icon?: string
  disabled: boolean
  /** Its place in its group, from 1, and the options in the group (aria-posinset, aria-setsize). */
  posInSet: number
  setSize: number
}
export interface CreateRow {
  kind: 'create'
  key: string
  id: string
  index: number
  row: number
  /** The search, as the new item's label. */
  label: string
  disabled: false
  posInSet: number
  setSize: number
}
export type Choice = OptionRow | CreateRow
export type Row = { kind: 'heading'; key: string; id: string; label: string; row: number } | { kind: 'separator'; key: string; row: number } | Choice
export interface RowGroup {
  key: string
  /** The id of its heading, which labels the group. */
  labelledBy?: string
  rows: Row[]
}

export interface ListboxOpts {
  /** The listbox element's id; option ids start with it. */
  id: string
  search: () => string
  touch: () => boolean
  autoHighlight?: () => boolean
  /** The listbox's name: `aria-labelledby` or `aria-label`. */
  label?: () => { 'aria-labelledby'?: string; 'aria-label'?: string }
  /** A choice was made; `changed` says whether the model changed (not for the create option, or the same value again). */
  onChoose?: (choice: Choice, event: Event, changed: boolean) => void
  onCreate?: (term: string) => void
  onHighlight?: (choice: Choice | undefined) => void
  /** The list itself takes focus (a sheet with no search field): it is a tab stop with aria-activedescendant, and takes the keys. */
  ownsFocus?: () => boolean
  /** Selection follows the highlight (a command palette's list of actions): the active option is the selected one, and no check is drawn. */
  selectsActive?: boolean
}

export interface Listbox {
  id: string
  props: ListboxProps
  groups: ComputedRef<RowGroup[]>
  rows: ComputedRef<Row[]>
  options: ComputedRef<Choice[]>
  active: Readonly<Ref<number>>
  activeOption: ComputedRef<Choice | undefined>
  activeId: ComputedRef<string | undefined>
  scroller: ShallowRef<HTMLElement | null>
  virtual: VirtualList
  rowHeight: ComputedRef<number>
  touch: () => boolean
  search: () => string
  label: () => { 'aria-labelledby'?: string; 'aria-label'?: string }
  /** Whether chosen options show a check (not when selection follows the highlight). */
  checks: boolean
  /** Whether the list element itself takes focus. */
  ownsFocus: () => boolean
  isSelected: (choice: Choice) => boolean
  highlight: (at: 'first' | 'last' | 'selected' | number, opts?: { scroll?: boolean }) => void
  choose: (choice: Choice, event: Event, toggle?: boolean) => void
  /** A press on an option: with multiple, Shift chooses a range and Ctrl or Cmd toggles. */
  press: (choice: Choice, event: MouseEvent) => void
  keydown: (event: KeyboardEvent, opts: { edges: boolean; typeahead: boolean; space: boolean; range?: boolean }) => boolean
  /** A drawn row measured this tall: rows grow to the tallest. */
  measure: (height: number) => void
  /** Forget the measured height (the list's width or fonts changed): rows go back to the estimate and are measured again. */
  remeasure: () => void
  /** Counts `remeasure` calls: the Listbox measures its rows after each. */
  measureRequests: Readonly<Ref<number>>
  /** Forget the typed search of the typeahead. */
  resetTypeahead: () => void
}

/** Past this many options a list renders only what is in view. */
export const VIRTUAL_FROM = 200
const ROW: Record<Size, number> = { sm: 28, md: 32, lg: 40 }
const TOUCH_ROW = 48

const createOf = (c: CreateItem | undefined) =>
  !c ? null : { position: (typeof c === 'object' && c.position) || 'bottom', when: c === 'always' || (typeof c === 'object' && c.when === 'always') ? 'always' : 'empty' }

export function useListbox(props: ListboxProps, model: Ref<unknown>, opts: ListboxOpts): Listbox {
  const id = opts.id
  const labelKey = () => props.labelKey ?? 'label'
  const descriptionKey = () => props.descriptionKey ?? 'description'

  const source = computed(() => groupsOf(props.items))
  /** The places of the items shown: each group's index and its items' indices. */
  const filtered = computed(() =>
    filterPlaces(source.value, props.ignoreFilter ? '' : opts.search(), props.filterFields ?? [labelKey()]),
  )
  const optionCount = computed(() =>
    filtered.value.reduce((n, p) => n + p.items.filter((i) => !isStructural(source.value[p.group]![i]!)).length, 0),
  )
  const virtualOn = computed(() =>
    props.virtualize === undefined ? optionCount.value > VIRTUAL_FROM : props.virtualize !== false,
  )
  const virtualOpts = () => (typeof props.virtualize === 'object' ? props.virtualize : {})

  const create = computed<string | null>(() => {
    const c = createOf(props.createItem)
    const term = opts.search().trim()
    if (!c || !term) return null
    if (c.when === 'empty') return optionCount.value === 0 ? term : null
    const exists = filtered.value.some((p) =>
      p.items.some((i) => {
        const item = source.value[p.group]![i]!
        return !isStructural(item) && fold(labelOf(item, labelKey())) === fold(term)
      }),
    )
    return exists ? null : term
  })

  /** Each group's rows, before row and option numbers are given. */
  const built = computed(() => {
    // Ids and keys follow the items' places in the props, so filtering keeps them.
    const out: RowGroup[] = []
    let index = 0
    let row = 0
    const addCreate = () => {
      const term = create.value
      if (term === null) return
      out.push({ key: 'create', rows: [{ kind: 'create', key: 'create', id: `${id}-create`, index: index++, row: row++, label: term, disabled: false, posInSet: 1, setSize: 1 }] })
    }
    const position = createOf(props.createItem)?.position
    if (position === 'top') addCreate()
    filtered.value.forEach((place) => {
      const gi = place.group
      const rows: Row[] = []
      let labelledBy: string | undefined
      if (out.length && !virtualOn.value) rows.push({ kind: 'separator', key: `s${gi}`, row: row++ })
      for (const i of place.items) {
        const item = source.value[gi]![i]!
        if (isStructural(item)) {
          const o = item as SelectOption
          if (o.type === 'separator') {
            if (!virtualOn.value) rows.push({ kind: 'separator', key: `s${gi}-${i}`, row: row++ })
            continue
          }
          const hid = `${id}-h${gi}-${i}`
          labelledBy ??= hid
          rows.push({ kind: 'heading', key: `h${gi}-${i}`, id: hid, label: String(o.label ?? ''), row: row++ })
          continue
        }
        const obj = isObject(item) ? (item as SelectOption) : undefined
        const description = obj ? get(obj, descriptionKey()) : undefined
        rows.push({
          kind: 'option',
          key: `o${gi}-${i}`,
          id: `${id}-${gi}-${i}`,
          index: index++,
          row: row++,
          item,
          value: valueOf(item, props.valueKey),
          label: labelOf(item, labelKey()),
          description: description == null ? undefined : String(description),
          icon: obj?.icon,
          disabled: !!obj?.disabled,
          posInSet: 0,
          setSize: 0,
        })
      }
      const inGroup = rows.filter((r): r is OptionRow => r.kind === 'option')
      inGroup.forEach((r, n) => {
        r.posInSet = n + 1
        r.setSize = inGroup.length
      })
      out.push({ key: `g${gi}`, labelledBy, rows })
    })
    if (position !== 'top') {
      // A separator before the create option at the bottom when there is a list above it.
      if (create.value !== null && out.length && !virtualOn.value) out[out.length - 1]!.rows.push({ kind: 'separator', key: 's-create', row: row++ })
      addCreate()
    }
    return out
  })
  const groups = computed(() => built.value)
  const rows = computed(() => built.value.flatMap((g) => g.rows))
  const options = computed(() => rows.value.filter((r): r is Choice => r.kind === 'option' || r.kind === 'create'))

  const active = ref(opts.autoHighlight?.() ? options.value.findIndex((o) => !o.disabled) : -1)
  const activeOption = computed(() => options.value[active.value])
  const activeId = computed(() => activeOption.value?.id)
  const scroller = shallowRef<HTMLElement | null>(null)
  /** The tallest a drawn row measured, px: rows never get shorter than it. */
  const measured = ref(0)
  /** Counts the requests to measure the drawn rows again (`remeasure`), which the Listbox watches. */
  const measureRequests = ref(0)
  const estimate = computed(() => {
    const given = virtualOpts().estimateSize
    if (given) return given
    const base = opts.touch() ? TOUCH_ROW : ROW[props.size ?? 'md']
    const described = options.value.some((o) => o.kind === 'option' && o.description)
    return described ? base + (opts.touch() ? 20 : 16) : base
  })
  const rowHeight = computed(() => Math.max(estimate.value, measured.value))
  // A new estimate (size, input, estimateSize, descriptions) draws rows
  // anew: measure them again, from the estimate, so they can get smaller.
  watch(estimate, () => (measured.value = 0))
  const virtual = useVirtualList(scroller, {
    count: () => rows.value.length,
    itemHeight: () => rowHeight.value,
    enabled: () => virtualOn.value,
    pinned: () => activeOption.value?.row ?? -1,
    get overscan() {
      return virtualOpts().overscan
    },
  })

  const values = () => (props.multiple ? (Array.isArray(model.value) ? model.value : []) : [model.value])
  const isSelected = (c: Choice) =>
    opts.selectsActive
      ? c === activeOption.value
      : c.kind === 'option' && values().some((v) => v !== undefined && v !== null && sameValue(v, c.value, props.by))

  const enabled = (i: number) => {
    const o = options.value[i]
    return !!o && !o.disabled
  }
  const firstEnabled = () => options.value.findIndex((o) => !o.disabled)
  const lastEnabled = () => {
    for (let i = options.value.length - 1; i >= 0; i--) if (enabled(i)) return i
    return -1
  }

  function setActive(i: number, scroll = true) {
    active.value = i
    const o = options.value[i]
    if (o && scroll) {
      virtual.scrollTo(o.row)
      // A row not rendered yet (the list just appeared) is scrolled to once it is.
      void nextTick(() => virtual.scrollTo(o.row))
    }
  }
  watch(activeOption, (o) => opts.onHighlight?.(o))

  function highlight(at: 'first' | 'last' | 'selected' | number, o: { scroll?: boolean } = {}) {
    let i: number
    if (at === 'first') i = firstEnabled()
    else if (at === 'last') i = lastEnabled()
    else if (at === 'selected') {
      i = options.value.findIndex((c) => !c.disabled && isSelected(c))
      if (i < 0) i = firstEnabled()
    } else i = enabled(at) ? at : active.value
    setActive(i, o.scroll !== false)
  }

  // Keep the active option through a change of rows when it is still there;
  // after a new search, or when one is always wanted, start on the first.
  let lastSearch = opts.search()
  watch(options, (now, before) => {
    const search = opts.search()
    const searched = search !== lastSearch
    lastSearch = search
    const key = before?.[active.value]?.key
    const kept = key === undefined ? -1 : now.findIndex((o) => o.key === key)
    if (searched && search) setActive(firstEnabled())
    else if (kept >= 0) active.value = kept
    else setActive(opts.autoHighlight?.() ? firstEnabled() : -1)
  })

  function move(delta: number) {
    const count = options.value.length
    if (!count) return
    if (active.value < 0 || !enabled(active.value)) return setActive(delta > 0 ? firstEnabled() : lastEnabled())
    const step = Math.sign(delta)
    let at = active.value
    let left = Math.abs(delta)
    let found = at
    while (left > 0) {
      let next = at + step
      if (next < 0 || next >= count) {
        if (!props.loop || Math.abs(delta) > 1) break
        next = (next + count) % count
      }
      at = next
      if (at === active.value) break
      if (enabled(at)) {
        found = at
        left--
      }
    }
    setActive(found)
  }

  const typeahead = useTypeahead(
    () => options.value.map((o) => (o.kind === 'option' && !o.disabled ? o.label : '')),
    (i) => setActive(i),
  )

  /** The option the last choice (or range) started from, by key. */
  let anchor: string | null = null
  const replacing = () => props.multiple && props.selectionBehavior === 'replace'
  const valuesOf = (list: Choice[]) => list.filter((c): c is OptionRow => c.kind === 'option' && !c.disabled).map((c) => c.value)

  /** Set the model, telling the owner whether it changed. */
  function commit(next: unknown, c: Choice, event: Event) {
    const changed = !sameValue(model.value, next, props.by)
    model.value = next
    if (c.kind === 'option' && isObject(c.item)) (c.item as SelectOption).onSelect?.(event)
    opts.onChoose?.(c, event, changed)
  }

  /**
   * Choose `c`: a disabled option does nothing, the create option creates;
   * single selection sets the value; multiple toggles it, or with
   * `toggle: false` (replace behaviour) makes it the only one.
   */
  function choose(c: Choice, event: Event, toggle = !replacing()) {
    if (c.disabled || props.disabled) return
    if (c.kind === 'create') {
      opts.onCreate?.(c.label)
      opts.onChoose?.(c, event, false)
      return
    }
    anchor = c.key
    if (!props.multiple) return commit(c.value, c, event)
    const list = values()
    if (!toggle) return commit([c.value], c, event)
    commit(isSelected(c) ? list.filter((v) => !sameValue(v, c.value, props.by)) : [...list, c.value], c, event)
  }

  /** Choose every enabled option from the anchor to `c`: added to the choice, or (replace) as the whole choice. */
  function chooseRange(c: Choice, event: Event, replace = replacing()) {
    if (!props.multiple || c.disabled || props.disabled || c.kind !== 'option') return
    // No anchor, or one filtered out of view: the range starts here.
    const at = options.value.findIndex((o) => o.key === anchor)
    const from = at < 0 ? c.index : at
    if (at < 0) anchor = c.key
    const [a, b] = [Math.min(from, c.index), Math.max(from, c.index)]
    const range = valuesOf(options.value.slice(a, b + 1))
    const kept = replace ? [] : values()
    commit([...kept, ...range.filter((v) => !kept.some((k) => sameValue(k, v, props.by)))], c, event)
  }

  /** Every enabled option chosen, or none when they all are. */
  function chooseAll(event: Event) {
    const c = activeOption.value ?? options.value[firstEnabled()]
    if (!props.multiple || props.disabled || !c) return
    const shown = valuesOf(options.value)
    const list = values()
    const all = shown.every((v) => list.some((x) => sameValue(x, v, props.by)))
    const others = list.filter((x) => !shown.some((v) => sameValue(x, v, props.by)))
    commit(all ? others : [...others, ...shown], c, event)
  }

  /** A press on `c`: Shift chooses a range, Ctrl or Cmd toggles, else as `choose`. */
  function press(c: Choice, event: MouseEvent) {
    if (props.multiple && event.shiftKey) return chooseRange(c, event)
    if (props.multiple && (event.ctrlKey || event.metaKey)) return choose(c, event, true)
    choose(c, event)
  }

  /** After a move with the arrows (multiple): Shift extends or toggles, Ctrl only moves, replace makes the choice follow. */
  function afterMove(event: KeyboardEvent) {
    const c = activeOption.value
    if (!props.multiple || !c || c.kind !== 'option' || event.ctrlKey || event.metaKey) return
    if (event.shiftKey) {
      if (replacing()) chooseRange(c, event, true)
      else choose(c, event, true)
    } else if (replacing()) choose(c, event, false)
  }

  function keydown(event: KeyboardEvent, k: { edges: boolean; typeahead: boolean; space: boolean; range?: boolean }): boolean {
    if (event.defaultPrevented || event.isComposing) return false
    // Ctrl (or Cmd) with an arrow moves without choosing, in a list that takes ranges.
    const ctrl = event.ctrlKey || event.metaKey
    const plain = !event.altKey && (!ctrl || (!!k.range && !!props.multiple))
    const moved = (fn: () => void) => {
      fn()
      if (k.range) afterMove(event)
      return true
    }
    switch (event.key) {
      case 'ArrowDown':
        return plain && moved(() => move(1))
      case 'ArrowUp':
        return plain && moved(() => move(-1))
      case 'PageDown':
        return moved(() => move(10))
      case 'PageUp':
        return moved(() => move(-10))
      case 'Home':
      case 'End':
        if (!k.edges) return false
        return moved(() => setActive(event.key === 'Home' ? firstEnabled() : lastEnabled()))
      case 'Enter': {
        const o = activeOption.value
        if (!o) return false
        choose(o, event)
        return true
      }
    }
    if (k.range && props.multiple && ctrl && !event.altKey && event.key.toLowerCase() === 'a') {
      chooseAll(event)
      return true
    }
    // Space inside a typed search is part of it; otherwise it chooses.
    if (k.typeahead && !ctrl && typeahead.onKeydown(event, active.value)) return true
    if (event.key === ' ' && k.space) {
      const o = activeOption.value
      if (!o) return false
      if (k.range && event.shiftKey) chooseRange(o, event)
      else if (k.range && ctrl) choose(o, event, true)
      else choose(o, event)
      return true
    }
    return false
  }

  return {
    id,
    props,
    groups,
    rows,
    options,
    active,
    activeOption,
    activeId,
    scroller,
    virtual,
    rowHeight,
    touch: opts.touch,
    search: opts.search,
    label: () => opts.label?.() ?? {},
    checks: !opts.selectsActive,
    ownsFocus: () => !!opts.ownsFocus?.(),
    isSelected,
    highlight,
    choose,
    press,
    keydown,
    resetTypeahead: typeahead.reset,
    measure: (height: number) => {
      if (height > measured.value) measured.value = Math.ceil(height)
    },
    remeasure: () => {
      measured.value = 0
      // Measure again even when nothing was measured before (no change above to react to).
      measureRequests.value++
    },
    measureRequests,
  }
}

/**
 * `props` as a listbox reads them, with some read elsewhere instead (a
 * control's size and disabled come from its field too). Reads stay
 * reactive: each goes to the props, or to its getter, when made.
 */
export function listProps<P extends object>(props: P, own: { [K in keyof ListboxProps]?: () => ListboxProps[K] }): ListboxProps {
  return new Proxy(props, {
    get: (target, key) => (key in own ? own[key as keyof ListboxProps]!() : Reflect.get(target, key)),
  }) as ListboxProps
}

const listboxKey: InjectionKey<Listbox | null> = Symbol('weft-listbox')

/** For the component that owns a list: the Listbox inside it draws `list`. */
export const provideListbox = (list: Listbox) => provide(listboxKey, list)

/** For Listbox: the list its owner gives it, if any; none for the Listboxes inside it. */
export function injectListbox(): Listbox | null {
  const list = inject(listboxKey, null)
  provide(listboxKey, null)
  return list
}
