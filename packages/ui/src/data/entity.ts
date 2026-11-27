// Types and helpers for EntityDialog and DataTable: schema-driven form fields,
// diffing an edited record against the original for PATCH, and mapping Warp's
// ApiError shape ({ status, code, message, fields }) onto form field errors
// and a top-level alert sentence.
import type { Width } from '../adaptive/detect.ts'
import type { ChoiceItem } from '../forms/choice.ts'
import type { FieldRule } from '../forms/schema.ts'
import type { SelectOption } from '../selection/items.ts'
import { TABLE_LABELS, type TableFetch, type TableLabels } from './table.ts'

export interface EntityFieldRule extends FieldRule {
  kind?: 'string' | 'number' | 'boolean' | 'list' | 'record' | 'object' | 'unknown'
  type?: 'string' | 'number' | 'boolean' | 'list' | 'record' | 'object' | 'unknown'
}

export function ruleKind(rule: EntityFieldRule | undefined): string | undefined {
  return rule?.kind ?? rule?.type
}

export type EntitySchema<Name extends string = string> = {
  [K in Name]?: EntityFieldRule
}

export type EntityControlKind =
  | 'input'
  | 'text'
  | 'email'
  | 'url'
  | 'textarea'
  | 'number'
  | 'checkbox'
  | 'switch'
  | 'radio'
  | 'select'

export interface EntityFieldConfig {
  /** The field's key in `schema` and `state`. */
  name: string
  /** Label shown on the FormField; default humanized from `name`. */
  label?: string
  /** Hint text beside the FormField label. */
  hint?: string
  /** Description text under the FormField label. */
  description?: string
  /** Placeholder text for text/number/select controls. */
  placeholder?: string
  /** Control override (`'textarea'`, `'select'`, `'radio'`, `'switch'`, `'checkbox'`, `'number'`, `'input'`, `'text'`, `'email'`, `'url'`). */
  control?: EntityControlKind
  /** Step for InputNumber (e.g. `0.01` for money, `1` for integers). */
  step?: number
  /** Rows for Textarea. */
  rows?: number
  /** Choice items for an enum field; default `{ label: humanLabel(v), value: v }` for each value in `rule.enum`. */
  items?: (string | { label: string; value: string | number })[]
  /** Hide the field from the auto-rendered form (its rules and field errors are also excluded from the auto-rendered form). */
  hidden?: boolean
}

export interface EntityDialogLabels {
  createTitle: (name: string) => string
  editTitle: (name: string, itemLabel?: string) => string
  createSubmit: string
  editSubmit: string
  cancel: string
  close: string
  noneOption: string
  unreachable: string
  pageFailed: string
}

export const ENTITY_DIALOG_LABELS: EntityDialogLabels = {
  createTitle: (name) => `New ${name}`,
  editTitle: (name, itemLabel) => `Edit ${itemLabel || name}`,
  createSubmit: 'Create',
  editSubmit: 'Save',
  cancel: 'Cancel',
  close: 'Close',
  noneOption: 'None',
  unreachable: 'The server could not be reached; check the connection and try again.',
  pageFailed: 'Something went wrong on this page; reload it and try again.',
}

export interface DataResource<T, TInput extends object = any> {
  list: TableFetch<T>
  create?: (body: TInput) => Promise<T>
  update?: (id: any, body: Partial<TInput>) => Promise<T>
  remove?: (id: any) => Promise<unknown>
}

export interface DataTableFilter {
  /** The filter key sent as `filter[key]=value`. */
  key: string
  /** The FormField label above the control; default humanized from `key`. */
  label?: string
  /** The choices; when no choice has `value: ''`, `{ label: allLabel || labels.anyFilter, value: '' }` is prepended unless `allLabel: false`. */
  items: (string | { label: string; value: string })[]
  /** Label for the auto-prepended `''` choice; default `labels.anyFilter` (`'Any'`), or `false` to add none. */
  allLabel?: string | false
  /** Control style: default `'radio'` when `<= 4` total options (and not narrow), else `'select'`. */
  control?: 'radio' | 'select'
}

export interface DataTableLabels extends TableLabels, EntityDialogLabels {
  search: string
  searchPlaceholder: string
  anyFilter: string
  booleanYes: string
  booleanNo: string
  newAction: (name: string) => string
  actionsColumn: string
  editAction: string
  editRowAction: (itemLabel: string) => string
  deleteAction: string
  deleteRowAction: (itemLabel: string) => string
  deleteTitle: (itemLabel: string) => string
  deleteDescription: string
  deleteConfirm: string
}

export const DATA_TABLE_LABELS: DataTableLabels = {
  ...TABLE_LABELS,
  ...ENTITY_DIALOG_LABELS,
  search: 'Search',
  searchPlaceholder: 'Search',
  anyFilter: 'Any',
  booleanYes: 'Yes',
  booleanNo: 'No',
  newAction: (name) => `New ${name}`,
  actionsColumn: 'Actions',
  editAction: 'Edit',
  editRowAction: (itemLabel) => `Edit ${itemLabel}`,
  deleteAction: 'Delete',
  deleteRowAction: (itemLabel) => `Delete ${itemLabel}`,
  deleteTitle: (itemLabel) => `Delete ${itemLabel}?`,
  deleteDescription: 'It is gone for good; this cannot be undone.',
  deleteConfirm: 'Delete',
}

/** Formats a field key (`firstName`, `unit_price`) as a sentence-cased label (`First name`, `Unit price`). */
export function humanLabel(key: string): string {
  const spaced = key
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .trim()
  if (!spaced) return key
  return spaced[0]!.toUpperCase() + spaced.slice(1).toLowerCase()
}

/** Returns true when a schema rule is a non-scalar kind (`list`, `record`, `object`) that requires a slot rather than an auto-rendered scalar input. */
export function isNonScalarRule(rule: EntityFieldRule | undefined): boolean {
  const k = ruleKind(rule)
  return k === 'list' || k === 'record' || k === 'object'
}

/** Chooses the control kind for a schema rule and optional field configuration at `width`. */
export function inferControl(rule: EntityFieldRule | undefined, config: Partial<EntityFieldConfig> | undefined, width: Width): Exclude<EntityControlKind, 'input'> {
  if (config?.control) return config.control === 'input' ? (rule?.email ? 'email' : rule?.url ? 'url' : 'text') : config.control
  const choices = config?.items ?? rule?.enum
  if (choices && choices.length > 0) {
    return choices.length <= 4 && width !== 'narrow' ? 'radio' : 'select'
  }
  const k = ruleKind(rule)
  if (k === 'boolean') return 'checkbox'
  if (k === 'number') return 'number'
  if (rule?.email) return 'email'
  if (rule?.url) return 'url'
  return 'text'
}

/** Normalizes enum or configured items into `{ label, value }` options, prepending a `noneOption` choice when optional. */
export function fieldChoices(
  rule: EntityFieldRule | undefined,
  config: Partial<EntityFieldConfig> | undefined,
  noneLabel: string,
): { label: string; value: string | number }[] {
  const raw = config?.items ?? rule?.enum ?? []
  const isNum = ruleKind(rule) === 'number'
  const items: { label: string; value: string | number }[] = raw.map((item) => {
    if (typeof item === 'object' && item !== null && 'value' in item) {
      return { label: item.label, value: item.value }
    }
    const str = String(item)
    return {
      label: humanLabel(str),
      value: isNum ? Number(str) : str,
    }
  })
  if (!rule?.required && !rule?.nullable) {
    const emptyVal = isNum ? 0 : ''
    if (!items.some((i) => i.value === emptyVal)) {
      items.unshift({ label: noneLabel, value: emptyVal })
    }
  }
  return items
}

/** Builds initial draft state for create or edit mode from `schema`, `record`, and `defaults`. */
export function initialDraft<T extends object, TInput extends object>(
  schema: EntitySchema,
  record: T | null | undefined,
  defaults?: Partial<TInput> | (() => Partial<TInput>),
  fields?: EntityFieldConfig[],
): TInput {
  const out: Record<string, unknown> = {}
  const resolvedDefaults = typeof defaults === 'function' ? defaults() : defaults
  const def = resolvedDefaults as Record<string, unknown> | undefined
  const rec = record as Record<string, unknown> | null | undefined
  const keys = new Set<string>(Object.keys(schema))
  if (fields) {
    for (const f of fields) keys.add(f.name)
  }
  for (const name of keys) {
    const rule = schema[name]
    const k = ruleKind(rule)
    if (rec && name in rec && rec[name] !== undefined) {
      out[name] = rec[name]
      continue
    }
    if (rec && rule?.nullable) {
      out[name] = rec[name] ?? null
      continue
    }
    if (def && name in def && def[name] !== undefined) {
      out[name] = def[name]
      continue
    }
    if (rule?.nullable) {
      out[name] = null
    } else if (k === 'boolean') {
      out[name] = false
    } else if (k === 'number') {
      out[name] = rule?.enum?.length ? Number(rule.enum[0]) : 0
    } else if (k === 'list') {
      out[name] = []
    } else if (k === 'record' || k === 'object') {
      out[name] = {}
    } else if (rule?.enum?.length) {
      out[name] = rule.required ? rule.enum[0]! : ''
    } else {
      out[name] = ''
    }
  }
  if (def && !rec) {
    for (const [k, v] of Object.entries(def)) {
      if (!(k in out) && v !== undefined) out[k] = v
    }
  }
  return out as TInput
}

/** Returns only the fields in `values` that differ from `was`, normalizing nullable empty values to `null`. */
export function changes<TInput extends object, T extends object>(
  values: TInput,
  was: T,
  schema: EntitySchema,
): Partial<TInput> {
  const out: Record<string, unknown> = {}
  const vals = values as Record<string, unknown>
  const orig = was as Record<string, unknown>
  const keys = new Set([...Object.keys(schema), ...Object.keys(vals)])
  for (const key of keys) {
    if (!(key in vals)) continue
    const rule = schema[key]
    const nextVal = rule?.nullable ? (vals[key] ?? null) : vals[key]
    const prevVal = rule?.nullable ? (orig[key] ?? null) : orig[key]
    if (!Object.is(nextVal, prevVal)) {
      out[key] = nextVal
    }
  }
  return out as Partial<TInput>
}

interface ApiErrorLike {
  status: number
  code: string
  message: string
  fields: Record<string, string>
}

function isApiErrorLike(e: unknown): e is ApiErrorLike {
  if (typeof e !== 'object' || e === null) return false
  const o = e as Record<string, unknown>
  return (
    typeof o.status === 'number' &&
    typeof o.code === 'string' &&
    typeof o.message === 'string' &&
    typeof o.fields === 'object' &&
    o.fields !== null
  )
}

/**
 * Maps an error thrown by `create`, `update` or `remove` to known form field
 * errors and a top-level alert sentence, without importing an app's generated client.
 */
export function entityProblem(
  e: unknown,
  knownFields: ReadonlySet<string>,
  labels: Pick<EntityDialogLabels, 'unreachable' | 'pageFailed'> = ENTITY_DIALOG_LABELS,
): { fields: Record<string, string>; message: string } {
  if (!isApiErrorLike(e)) {
    console.error('The page failed:', e)
    return { fields: {}, message: labels.pageFailed }
  }
  if (e.code === 'network' || e.code === 'http') {
    return { fields: {}, message: labels.unreachable }
  }
  const fields: Record<string, string> = {}
  const extra: string[] = []
  for (const [k, v] of Object.entries(e.fields)) {
    if (knownFields.has(k)) fields[k] = v
    else if (v) extra.push(v)
  }
  if (Object.keys(fields).length > 0 || extra.length > 0) {
    return { fields, message: extra.join(' ') }
  }
  return { fields: {}, message: e.message }
}

export type { ChoiceItem, SelectOption }
