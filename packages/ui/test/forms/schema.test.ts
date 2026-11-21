// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { validateField, validateFields, type FieldSchema } from '../../src/forms/schema.ts'

describe('validateField', () => {
  it('passes anything when the rule is empty', () => {
    expect(validateField(undefined, {})).toBeUndefined()
    expect(validateField('x', {})).toBeUndefined()
  })

  it('requires a value: not undefined, null, blank text, an empty list or an unticked box', () => {
    for (const empty of [undefined, null, '', '   ', [], false]) {
      expect(validateField(empty, { required: true }), JSON.stringify(empty)).toBe('This field is required.')
    }
    for (const value of ['a', 0, true, ['x']]) expect(validateField(value, { required: true })).toBeUndefined()
  })

  it('skips every other rule for an empty value that is not required', () => {
    expect(validateField('', { min: 3, email: true, pattern: '[0-9]+' })).toBeUndefined()
    expect(validateField(null, { min: 3 })).toBeUndefined()
  })

  it('reads min and max as a length for text and lists, and as bounds for numbers', () => {
    expect(validateField('ab', { min: 3 })).toBe('Use at least 3 characters.')
    expect(validateField('abcd', { max: 3 })).toBe('Use at most 3 characters.')
    expect(validateField('a', { min: 1 })).toBeUndefined()
    expect(validateField(['a'], { min: 2 })).toBe('Choose at least 2.')
    expect(validateField(['a', 'b', 'c'], { max: 2 })).toBe('Choose at most 2.')
    expect(validateField(2, { min: 3 })).toBe('Enter 3 or more.')
    expect(validateField(11, { max: 10 })).toBe('Enter 10 or less.')
    expect(validateField(3, { min: 3, max: 3 })).toBeUndefined()
  })

  it('counts characters, not UTF-16 units', () => {
    expect(validateField('😀😀', { max: 2 })).toBeUndefined()
  })

  it('matches pattern against the whole value, as the HTML pattern attribute does', () => {
    expect(validateField('123', { pattern: '[0-9]+' })).toBeUndefined()
    expect(validateField('12a', { pattern: '[0-9]+' })).toBe('Use the format asked for.')
    expect(validateField('a1|b', { pattern: 'a1|b' })).toBe('Use the format asked for.')
    expect(validateField('b', { pattern: 'a1|b' })).toBeUndefined()
  })

  it('matches pattern against a number written out, and checks a number as an email address the same way', () => {
    expect(validateField(123, { pattern: '[0-9]{3}' })).toBeUndefined()
    expect(validateField(12, { pattern: '[0-9]{3}' })).toBe('Use the format asked for.')
    expect(validateField(-1.5, { pattern: '-?[0-9]+' })).toBe('Use the format asked for.')
    expect(validateField(42, { email: true })).toBe('Enter an email address.')
  })

  it('applies only required to a value that is neither text, number nor list', () => {
    expect(validateField(true, { min: 3, max: 0, pattern: 'x', email: true })).toBeUndefined()
    expect(validateField({ a: 1 }, { min: 3, pattern: 'x', email: true })).toBeUndefined()
    expect(validateField(['a'], { pattern: 'x', email: true })).toBeUndefined()
  })

  it('checks an email address as a browser does for type="email"', () => {
    expect(validateField('ada@example.com', { email: true })).toBeUndefined()
    expect(validateField('ada@localhost', { email: true })).toBeUndefined()
    for (const bad of ['ada', 'ada@', '@example.com', 'a da@example.com', 'ada@exa_mple.com']) {
      expect(validateField(bad, { email: true }), bad).toBe('Enter an email address.')
    }
  })

  it('says message in place of any of its own sentences', () => {
    const rule = { required: true, min: 8, message: 'Use 8 or more characters.' }
    expect(validateField('', rule)).toBe('Use 8 or more characters.')
    expect(validateField('short', rule)).toBe('Use 8 or more characters.')
    expect(validateField('long enough', rule)).toBeUndefined()
  })
})

describe('validateFields', () => {
  it('gives the first error of each field that fails, by name', () => {
    const schema: FieldSchema = { name: { required: true }, email: { required: true, email: true }, age: { min: 18 } }
    expect(validateFields(schema, { name: 'Ada', email: 'nope', age: 12 })).toEqual({
      email: 'Enter an email address.',
      age: 'Enter 18 or more.',
    })
    expect(validateFields(schema, { name: 'Ada', email: 'ada@example.com', age: 36 })).toEqual({})
  })

  it('checks a field the values do not have as empty', () => {
    expect(validateFields({ terms: { required: true } }, {})).toEqual({ terms: 'This field is required.' })
  })
})
