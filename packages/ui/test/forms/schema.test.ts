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
      expect(validateField(empty, { required: true }), JSON.stringify(empty)).toBe('This is required.')
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

  it('checks enum: the value must be one of them, in the order given', () => {
    const rule = { enum: ['draft', 'live', 'gone'] }
    expect(validateField('live', rule)).toBeUndefined()
    expect(validateField('Live', rule)).toBe('Choose one of: draft, live, gone.')
    expect(validateField('other', { enum: ['a'] })).toBe('Choose one of: a.')
    expect(validateField('other', { enum: ['a'], message: 'Pick one.' })).toBe('Pick one.')
  })

  it('checks url as Go does: no blanks, http(s)://, a host, and a port of one to five digits', () => {
    const sentence = 'Enter a web address that starts with http:// or https://.'
    const accept = ['http://x.com', 'HTTPS://X.COM/a?b#c', 'http://u:p@x.com', 'http://x.com:8080', 'http://[::1]:8080/']
    const refuse = ['ftp://x', 'http://', '//x.com', 'http:x.com', 'http://a b.com', 'http://x.com/a b', 'http://:8080', 'https:///x.com', 'http://x.com ', 'http://x.com/a\tb', 'http://x.com/a\nb', 'http://x.com\\a', 'http://x:999999']
    for (const good of accept) expect(validateField(good, { url: true }), good).toBeUndefined()
    for (const bad of refuse) expect(validateField(bad, { url: true }), JSON.stringify(bad)).toBe(sentence)
    expect(validateField('nope', { url: true, message: 'Bad.' })).toBe('Bad.')
  })

  it('lets an empty list skip every rule but required', () => {
    expect(validateField([], { min: 1 })).toBeUndefined()
    expect(validateField([], { max: 0 })).toBeUndefined()
    expect(validateField([], { required: true })).toBe('This is required.')
    expect(validateField(['a'], { min: 2 })).toBe('Choose at least 2.')
  })

  it('checks a blank value (only spaces) against every rule but required', () => {
    expect(validateField('  ', { url: true })).toBe('Enter a web address that starts with http:// or https://.')
    expect(validateField('  ', { enum: ['a', 'b'] })).toBe('Choose one of: a, b.')
    expect(validateField('  ', { email: true })).toBe('Enter an email address.')
    expect(validateField('  ', { min: 3 })).toBe('Use at least 3 characters.')
    expect(validateField('  ', { pattern: '[0-9]+' })).toBe('Use the format asked for.')
    expect(validateField('  ', { required: true, email: true })).toBe('This is required.')
  })

  it('checks enum exactly: an empty string in the list is allowed, spaces around a value are not ignored', () => {
    expect(validateField('', { enum: ['', 'a'] })).toBeUndefined()
    expect(validateField('a', { enum: ['', 'a'] })).toBeUndefined()
    expect(validateField('b', { enum: ['', 'a'] })).toBe('Choose one of: , a.')
    expect(validateField(' live ', { enum: ['draft', 'live'] })).toBe('Choose one of: draft, live.')
  })

  it('lets an empty optional value pass url and enum, and still requires when asked', () => {
    for (const empty of ['', undefined, null]) {
      expect(validateField(empty, { url: true, enum: ['a'], email: true, min: 2, pattern: 'x' })).toBeUndefined()
    }
    expect(validateField('', { url: true, enum: ['a'], required: true })).toBe('This is required.')
  })

  it('gives the first failing rule: email, then url, then enum', () => {
    expect(validateField('x', { email: true, url: true, enum: ['a'] })).toBe('Enter an email address.')
    expect(validateField('x', { url: true, enum: ['a'] })).toBe('Enter a web address that starts with http:// or https://.')
    expect(validateField('x', { min: 3, url: true })).toBe('Use at least 3 characters.')
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
    expect(validateFields({ terms: { required: true } }, {})).toEqual({ terms: 'This is required.' })
  })
})
