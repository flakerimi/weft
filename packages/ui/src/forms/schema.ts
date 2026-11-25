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
 * - `url`: the text must be an absolute web address: `http:` or `https:`
 *   with a host.
 * - `enum`: the text must be one of these values (`enum: ['draft', 'live']`).
 * - `message`: one sentence to show in place of any of the sentences below.
 *
 * By the value's type at run time:
 *
 * | value          | required | min, max          | pattern, email      | url, enum   |
 * |----------------|----------|-------------------|---------------------|-------------|
 * | text           | yes      | length, in chars  | the text            | the text    |
 * | number         | yes      | the number itself | the number written  | not checked |
 * |                |          |                   | out (`String(n)`)   |             |
 * | list (array)   | yes      | how many          | not checked         | not checked |
 * | anything else  | yes      | not checked       | not checked         | not checked |
 *
 * (A boolean, a date, a file: only `required` applies.)
 *
 * An empty value that is not required passes every other rule. A field's
 * first failing rule, in the order above (`url` and `enum` last, `email` before them), gives its error.
 */

/** The rules for one field. */
export interface FieldRule {
  required?: boolean
  min?: number
  max?: number
  pattern?: string
  email?: boolean
  /** The text must be an absolute `http:` or `https:` address with a host. */
  url?: boolean
  /** The text must be one of these values. */
  enum?: string[]
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
    return textRules(String(value), rule, fail)
  } else if (Array.isArray(value)) {
    if (rule.min !== undefined && value.length < rule.min) return fail(`Choose at least ${rule.min}.`)
    if (rule.max !== undefined && value.length > rule.max) return fail(`Choose at most ${rule.max}.`)
  } else if (typeof value === 'string') {
    const length = Array.from(value).length
    if (rule.min !== undefined && length < rule.min) return fail(`Use at least ${plural(rule.min, 'character', 'characters')}.`)
    if (rule.max !== undefined && length > rule.max) return fail(`Use at most ${plural(rule.max, 'character', 'characters')}.`)
    return textRules(value, rule, fail) ?? webRules(value, rule, fail)
  }
  return undefined
}

/** `url` and `enum`, for text only (as the Go side checks them). */
function webRules(text: string, rule: FieldRule, fail: (sentence: string) => string): string | undefined {
  if (rule.url && !isWebAddress(text)) return fail('Enter a web address that starts with http:// or https://.')
  if (rule.enum && !rule.enum.includes(text)) return fail(`Choose one of: ${rule.enum.join(', ')}.`)
  return undefined
}

function isWebAddress(text: string): boolean {
  if (!/^https?:\/\//i.test(text)) return false
  try {
    return new URL(text).hostname !== ''
  } catch {
    return false
  }
}

/** `pattern` and `email`, for text and numbers written out. */
function textRules(text: string, rule: FieldRule, fail: (sentence: string) => string): string | undefined {
  if (rule.pattern !== undefined && !compiled(rule.pattern).test(text)) return fail('Use the format asked for.')
  if (rule.email && !EMAIL.test(text)) return fail('Enter an email address.')
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
