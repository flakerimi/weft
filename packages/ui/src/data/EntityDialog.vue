<script setup vapor lang="ts" generic="T extends object, TInput extends object = any">
// A modal dialog that creates or edits a record from a Weft field schema
// (such as `schema.<Name>Input` from `weft gen`). Each field in `schema`
// renders its matching form control unless hidden or replaced by a
// `#<name>-field` slot or the default slot. In edit mode (`record` non-null)
// only the fields that changed are sent to `update(id, changed)`, and
// submitting with no changes closes the dialog without a request. Server
// errors matching Warp's `ApiError` shape show beside their fields or in an
// Alert at the top of the form.
import { computed, onMounted, reactive, ref, shallowRef, useSlots, useTemplateRef, watch } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import Button from '../basics/Button.vue'
import Alert from '../feedback/Alert.vue'
import Checkbox from '../forms/Checkbox.vue'
import Form from '../forms/Form.vue'
import FormField from '../forms/FormField.vue'
import type { FormApi } from '../forms/form-state.ts'
import Input from '../forms/Input.vue'
import InputNumber from '../forms/InputNumber.vue'
import RadioGroup from '../forms/RadioGroup.vue'
import type { FieldRule, FieldSchema } from '../forms/schema.ts'
import Switch from '../forms/Switch.vue'
import Textarea from '../forms/Textarea.vue'
import { useControllable } from '../headless/useControllable.ts'
import Stack from '../layout/Stack.vue'
import Dialog from '../overlays/Dialog.vue'
import Select from '../selection/Select.vue'
import {
  changes,
  ENTITY_DIALOG_LABELS,
  entityProblem,
  fieldChoices,
  humanLabel,
  inferControl,
  initialDraft,
  isNonScalarRule,
  ruleKind,
  type EntityControlKind,
  type EntityDialogLabels,
  type EntityFieldConfig,
  type EntityFieldRule,
  type EntitySchema,
} from './entity.ts'
import '../styles/base.css'

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** `null` or `undefined` for create mode; a record for edit mode. */
    record?: T | null
    /** The generated `schema.<Name>Input` object. */
    schema: EntitySchema
    /** Called on create submit (`record == null`). */
    create?: (body: TInput) => Promise<T>
    /** Called on edit submit (`record != null`) with only changed fields. */
    update?: (id: any, body: Partial<TInput>) => Promise<T>
    /** Key used to read the record ID for `update`; default `'id'`. */
    idKey?: string
    /** Key used to read the record's display name for the edit title; default `'name'`, falling back to `'title'`, then `name`. */
    labelKey?: string
    /** Singular entity name (e.g. `'product'`), used for default titles (`"New product"`, `"Edit Kettle"`). */
    name?: string
    /** Explicit dialog title override. */
    title?: string
    /** Optional dialog description. */
    description?: string
    /** Initial field values for create mode. */
    defaults?: Partial<TInput> | (() => Partial<TInput>)
    /** Ordered field configurations (`name`, `label`, `hint`, `description`, `placeholder`, `control`, `step`, `rows`, `items`, `hidden`). */
    fields?: EntityFieldConfig[]
    /** The words it says; give some or all to translate them. */
    labels?: Partial<EntityDialogLabels>
  }>(),
  {
    open: undefined,
    defaultOpen: false,
    create: undefined,
    update: undefined,
    idKey: 'id',
    labelKey: 'name',
    name: 'item',
    title: undefined,
    description: undefined,
    defaults: undefined,
    fields: undefined,
  },
)

const emit = defineEmits<{
  /** Opened or closed. */
  'update:open': [open: boolean]
  /** Emitted after `create` or `update` resolves. */
  saved: [record: T]
  /** Emitted when `create` or `update` rejects. */
  error: [error: unknown]
}>()

defineSlots<{
  /** Custom form body replacing the auto-rendered fields. */
  default?: (props: { state: TInput; record: T | null; schema: EntitySchema }) => unknown
  /** Custom control for a single field, inside its `FormField`. */
  [field: `${string}-field`]: ((props: { state: TInput; record: T | null; rule: EntityFieldRule; field: EntityFieldConfig }) => unknown) | undefined
}>()

const slots = useSlots()
const t = computed<EntityDialogLabels>(() => ({ ...ENTITY_DIALOG_LABELS, ...props.labels }))
const open = useControllable<{ open?: boolean }, 'open'>(props, 'open', emit, () => props.defaultOpen)

const viewport = shallowRef<HTMLElement | null>(null)
onMounted(() => (viewport.value = document.documentElement))
const width = useWidth(viewport, { prepaint: false })

const draft = reactive<Record<string, unknown>>({})
const getDraft = (): TInput => draft as TInput
const problem = ref('')
const formKey = ref(0)
const form = useTemplateRef<FormApi>('form')

function resetDraft() {
  const next = initialDraft<T, TInput>(props.schema as EntitySchema, props.record, props.defaults, props.fields)
  for (const k of Object.keys(draft)) {
    if (!(k in next)) delete draft[k]
  }
  Object.assign(draft, next)
  problem.value = ''
  formKey.value++
}

resetDraft()
watch(
  [() => open.value, () => props.record],
  ([isOpen], [wasOpen]) => {
    if (isOpen && !wasOpen) resetDraft()
    else if (isOpen) resetDraft()
  },
)

interface FieldEntry {
  key: Extract<keyof TInput, string>
  rule: EntityFieldRule
  field: EntityFieldConfig
  label: string
  control: Exclude<EntityControlKind, 'input'>
  choices: { label: string; value: string | number }[]
}

const entries = computed<FieldEntry[]>(() => {
  const out: FieldEntry[] = []
  const schemaMap = props.schema as EntitySchema
  const byName = new Map<string, EntityFieldConfig>()
  if (props.fields) {
    for (const f of props.fields) byName.set(f.name, f)
  }
  const orderedKeys: string[] = props.fields
    ? [...props.fields.map((f) => f.name), ...Object.keys(schemaMap).filter((k) => !byName.has(k))]
    : Object.keys(schemaMap)
  const seen = new Set<string>()
  for (const k of orderedKeys) {
    if (seen.has(k)) continue
    seen.add(k)
    const config = byName.get(k)
    if (config?.hidden) continue
    const rule = schemaMap[k] ?? {}
    // Non-scalar fields (list, record, object) require a `#<name>-field` slot.
    if (isNonScalarRule(rule) && !slots[`${k}-field`]) continue
    const field: EntityFieldConfig = config ?? { name: k }
    out.push({
      key: k as Extract<keyof TInput, string>,
      rule,
      field,
      label: field.label ?? humanLabel(k),
      control: inferControl(rule, field, width.value),
      choices: fieldChoices(rule, field, t.value.noneOption),
    })
  }
  return out
})

const renderedKeys = computed(() => new Set<string>(entries.value.map((e) => e.key)))

// Only validate fields that are actually rendered (unless a custom default slot
// replaces the field list), and in edit mode require a non-nullable number so an
// emptied spinbutton cannot silently leave the old value unsaid.
const formSchema = computed<FieldSchema<Extract<keyof TInput, string>>>(() => {
  const out: Record<string, FieldRule> = {}
  const customBody = Boolean(slots.default)
  const shown = renderedKeys.value
  for (const [k, rule] of Object.entries(props.schema as EntitySchema)) {
    if (!rule) continue
    if (!customBody && !shown.has(k)) continue
    out[k] = props.record && ruleKind(rule) === 'number' && !rule.nullable ? { ...rule, required: true } : rule
  }
  return out as FieldSchema<Extract<keyof TInput, string>>
})

const itemTitle = computed(() => {
  if (props.title) return props.title
  if (!props.record) return t.value.createTitle(props.name)
  const rec = props.record as Record<string, unknown>
  const first = rec[props.labelKey]
  const second = rec.title
  const raw = first !== undefined && first !== null && String(first) !== '' ? first : second
  const label = raw !== undefined && raw !== null && String(raw) !== '' ? String(raw) : undefined
  return t.value.editTitle(props.name, label)
})

function getStr(key: string): string | null {
  const v = draft[key]
  return v == null ? '' : String(v)
}
function setStr(key: string, v: string | number | null) {
  draft[key] = v == null ? '' : String(v)
}
function getNum(key: string): number | null {
  const v = draft[key]
  return typeof v === 'number' ? v : null
}
function setNum(key: string, v: number | null) {
  draft[key] = v
}
function getBool(key: string): boolean {
  return Boolean(draft[key])
}
function setBool(key: string, v: boolean | 'indeterminate') {
  draft[key] = v === true
}
function getChoice(key: string): string | number | null {
  const v = draft[key]
  return typeof v === 'string' || typeof v === 'number' ? v : null
}
function setChoice(key: string, v: unknown) {
  draft[key] = v
}

async function onSubmit(values: Record<string, unknown>) {
  problem.value = ''
  try {
    let saved: T
    if (props.record) {
      const diff = changes(values as TInput, props.record, props.schema as EntitySchema)
      if (Object.keys(diff).length === 0) {
        open.value = false
        return
      }
      if (!props.update) return
      const id = (props.record as Record<string, unknown>)[props.idKey]
      saved = await props.update(id, diff)
    } else {
      if (!props.create) return
      saved = await props.create(values as TInput)
    }
    open.value = false
    emit('saved', saved)
  } catch (e) {
    emit('error', e)
    const known = slots.default ? new Set(Object.keys(props.schema as EntitySchema)) : renderedKeys.value
    const { fields, message } = entityProblem(e, known, t.value)
    if (Object.keys(fields).length > 0) form.value?.setErrors(fields)
    problem.value = message
  }
}

defineExpose({
  /** Reset the form draft and errors. */
  reset: resetDraft,
  /** The inner Form instance. */
  form,
})
</script>

<template>
  <Dialog v-model:open="open" :title="itemTitle" :description="description" :close-label="t.close">
    <template #body>
      <Form :key="formKey" ref="form" :state="draft" :schema="formSchema" @submit="onSubmit">
        <Stack :gap="4">
          <Alert v-if="problem" role="alert" tone="danger" :title="problem" />
          <slot v-if="$slots.default" :state="getDraft()" :record="record ?? null" :schema="schema" />
          <template v-else>
            <FormField
              v-for="entry in entries"
              :key="`${entry.key}-${entry.control}`"
              :name="entry.key"
              :label="entry.control === 'checkbox' || entry.control === 'switch' ? undefined : entry.label"
              :hint="entry.field.hint"
              :description="entry.field.description"
            >
              <slot
                v-if="$slots[`${entry.key}-field`]"
                :name="`${entry.key}-field`"
                :state="getDraft()"
                :record="record ?? null"
                :rule="entry.rule"
                :field="entry.field"
              />
              <RadioGroup
                v-else-if="entry.control === 'radio'"
                :model-value="getChoice(entry.key)"
                :items="entry.choices"
                orientation="horizontal"
                @update:model-value="setChoice(entry.key, $event)"
              />
              <Select
                v-else-if="entry.control === 'select'"
                :model-value="getChoice(entry.key)"
                :items="entry.choices"
                :placeholder="entry.field.placeholder"
                @update:model-value="setChoice(entry.key, $event)"
              />
              <Checkbox
                v-else-if="entry.control === 'checkbox'"
                :model-value="getBool(entry.key)"
                :label="entry.label"
                @update:model-value="setBool(entry.key, $event)"
              />
              <Switch
                v-else-if="entry.control === 'switch'"
                :model-value="getBool(entry.key)"
                :label="entry.label"
                @update:model-value="setBool(entry.key, $event)"
              />
              <InputNumber
                v-else-if="entry.control === 'number'"
                :model-value="getNum(entry.key)"
                :min="entry.rule.min"
                :max="entry.rule.max"
                :step="entry.field.step"
                :placeholder="entry.field.placeholder"
                @update:model-value="setNum(entry.key, $event)"
              />
              <Textarea
                v-else-if="entry.control === 'textarea'"
                :model-value="getStr(entry.key)"
                :rows="entry.field.rows"
                :placeholder="entry.field.placeholder"
                @update:model-value="setStr(entry.key, $event)"
              />
              <Input
                v-else
                :model-value="getStr(entry.key)"
                :type="entry.control"
                :placeholder="entry.field.placeholder"
                @update:model-value="setStr(entry.key, $event)"
              />
            </FormField>
          </template>
          <Stack direction="row" :gap="2" justify="end">
            <Button :label="t.cancel" variant="ghost" @click="open = false" />
            <Button type="submit" :label="record ? t.editSubmit : t.createSubmit" />
          </Stack>
        </Stack>
      </Form>
    </template>
  </Dialog>
</template>
