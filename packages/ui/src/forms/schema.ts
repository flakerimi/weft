/**
 * The Weft field schema: what a Form checks before it submits. It is plain
 * JSON, so a server can send the same rules it enforces (the Go contract will
 * generate it), keyed by field name:
 *
 *     { email: { required: true, email: true }, age: { min: 18 } }
 *
 * - `required`: the value must not be empty. Empty is undefined, null, text
 *   that is only whitespace, an empty list, or `false` (an unticked box).
 * - `min`, `max`: for text, the length in characters; for a list, how many
 *   are chosen; for a number, its bounds (inclusive).
 * - `pattern`: a regular expression the whole text must match, as the HTML
 *   `pattern` attribute reads it (`[0-9]+` means digits only).
 * - `email`: the text must be an email address, by the rule browsers use for
 *   `type="email"`.
 * - `message`: one sentence to show in place of any of the sentences below.
 *
 * An empty value that is not required passes every other rule. A field's
 * first failing rule, in the order above, gives its error.
 */

/** The rules for one field. */
export interface FieldRule {
  required?: boolean
  min?: number
  max?: number
  pattern?: string
  email?: boolean
  message?: string
}

/** Rules by field name. */
export type FieldSchema<Name extends string = string> = { [K in Name]?: FieldRule }

// The WHATWG "valid email address" rule, which browsers apply to type="email".
const EMAIL =
  /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/

const patterns = new Map<string, RegExp>()
function compiled(pattern: string): RegExp {
  let re = patterns.get(pattern)
  if (!re) {
    try {
      re = new RegExp(`^(?:${pattern})$`, 'u')
    } catch {
      throw new Error(`Form: the pattern "${pattern}" is not a regular expression; write it as the HTML pattern attribute takes it.`)
    }
    patterns.set(pattern, re)
  }
  return re
}

function isEmpty(value: unknown): boolean {
  return (
    value === undefined ||
    value === null ||
    value === false ||
    (typeof value === 'string' && value.trim() === '') ||
    (Array.isArray(value) && value.length === 0)
  )
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`

/** The error for `value` under `rule`, or undefined when it passes. */
export function validateField(value: unknown, rule: FieldRule): string | undefined {
  const fail = (sentence: string) => rule.message ?? sentence
  if (isEmpty(value)) return rule.required ? fail('This field is required.') : undefined
  if (typeof value === 'number') {
    if (rule.min !== undefined && value < rule.min) return fail(`Enter ${rule.min} or more.`)
    if (rule.max !== undefined && value > rule.max) return fail(`Enter ${rule.max} or less.`)
  } else if (Array.isArray(value)) {
    if (rule.min !== undefined && value.length < rule.min) return fail(`Choose at least ${rule.min}.`)
    if (rule.max !== undefined && value.length > rule.max) return fail(`Choose at most ${rule.max}.`)
  } else if (typeof value === 'string') {
    const length = Array.from(value).length
    if (rule.min !== undefined && length < rule.min) return fail(`Use at least ${plural(rule.min, 'character', 'characters')}.`)
    if (rule.max !== undefined && length > rule.max) return fail(`Use at most ${plural(rule.max, 'character', 'characters')}.`)
    if (rule.pattern !== undefined && !compiled(rule.pattern).test(value)) return fail('Use the format asked for.')
    if (rule.email && !EMAIL.test(value)) return fail('Enter an email address.')
  }
  return undefined
}

/** The first error of each field in `schema` that fails, by name; `{}` when all pass. */
export function validateFields(schema: FieldSchema, values: Record<string, unknown>): Record<string, string> {
  const errors: Record<string, string> = {}
  for (const [name, rule] of Object.entries(schema)) {
    if (!rule) continue
    const error = validateField(values[name], rule)
    if (error !== undefined) errors[name] = error
  }
  return errors
}
