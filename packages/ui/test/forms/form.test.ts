import { afterEach, describe, expect, it } from 'vitest'
import FormCases from '../fixtures/forms/FormCases.vue'
import { press } from '../keyboard.ts'
import { mount } from '../mount.ts'
import { settle, type } from './helpers.ts'

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
})
function render(props: Record<string, unknown> = {}) {
  const m = mount(FormCases, props)
  mounted.push(m)
  const q = <E extends Element = HTMLInputElement>(sel: string) => m.el.querySelector<E>(sel)!
  const error = (field: string) => m.el.querySelector(`.${field} .weft-form-field__error`)?.textContent?.trim()
  const submit = async () => {
    q<HTMLButtonElement>('button[type="submit"]').click()
    await settle()
  }
  return { ...m, q, error, submit, state: m.exposed.state as Record<string, unknown> }
}

describe('Form', () => {
  it('is a form the browser does not check, so Weft shows the errors', () => {
    const form = render().q<HTMLFormElement>('form')
    expect(form.className).toBe('weft-form')
    expect(form.noValidate).toBe(true)
  })

  it('marks fields the schema requires', () => {
    const f = render()
    expect(f.q('.email .weft-form-field__required')).not.toBeNull()
    expect(f.q('.age .weft-form-field__required')).toBeNull()
    expect(f.q('.email input').getAttribute('aria-required')).toBe('true')
  })

  it('shows no error before the first submit, even when a field is left empty', async () => {
    const f = render()
    f.q('.email input').focus()
    f.q('.email input').blur()
    await settle()
    expect(f.error('email')).toBeUndefined()
  })

  it('on submit with errors shows each, marks the controls invalid, emits error and focuses the first invalid control', async () => {
    const f = render()
    await f.submit()
    expect(f.error('email')).toBe('This field is required.')
    expect(f.error('terms')).toBe('This field is required.')
    expect(f.error('plan')).toBe('This field is required.')
    expect(f.error('age')).toBeUndefined()
    expect(f.error('nick')).toBeUndefined()
    const email = f.q('.email input')
    expect(email.getAttribute('aria-invalid')).toBe('true')
    expect(email.getAttribute('aria-describedby')).toBe(f.q('.email .weft-form-field__error').id)
    expect(f.q('.plan [role="radiogroup"]').getAttribute('aria-invalid')).toBe('true')
    expect(f.exposed.submitted).toEqual([])
    expect(f.exposed.failed).toEqual([{ email: 'This field is required.', terms: 'This field is required.', plan: 'This field is required.' }])
    expect(document.activeElement).toBe(email)
  })

  it('submits typed values once every field passes', async () => {
    const f = render()
    type(f.q('.email input'), 'ada@example.com')
    f.q('.age input').focus()
    type(f.q('.age input'), '36')
    f.q('.age input').blur()
    f.q('.terms input').click()
    f.q('.plan input').click()
    await settle()
    await f.submit()
    expect(f.exposed.failed).toEqual([])
    expect(f.exposed.submitted).toEqual([{ email: 'ada@example.com', age: 36, terms: true, plan: 'free', nick: '' }])
    expect(typeof (f.exposed.submitted as { age: unknown }[])[0]!.age).toBe('number')
  })

  it('submits with Enter in a field, as a form does', async () => {
    const f = render()
    expect(press(f.q('button[type="submit"]'), 'Enter')).toBe(true)
    await settle()
    expect(f.exposed.failed).toHaveLength(1)
  })

  it('after the first submit checks a field again when it is left', async () => {
    const f = render()
    await f.submit()
    const age = f.q('.age input')
    age.focus()
    type(age, '12')
    age.blur()
    await settle()
    expect(f.error('age')).toBe('You must be 18 or over.')
    age.focus()
    type(age, '20')
    age.blur()
    await settle()
    expect(f.error('age')).toBeUndefined()
  })

  it('clears a field\'s error as soon as it is fixed, without adding errors while typing', async () => {
    const f = render()
    await f.submit()
    const email = f.q('.email input')
    type(email, 'ada')
    await settle()
    expect(f.error('email')).toBe('Enter an email address.')
    type(email, 'ada@example.com')
    await settle()
    expect(f.error('email')).toBeUndefined()
    // A field with no error is not checked while typing, only when left.
    type(email, 'nope')
    await settle()
    expect(f.error('email')).toBeUndefined()
  })

  it('shows errors from a server with setErrors, until the field is checked again', async () => {
    const f = render()
    ;(f.exposed.setErrors as (e: Record<string, string>) => void)({ email: 'That address is taken.', nick: 'Taken too.' })
    await settle()
    expect(f.error('email')).toBe('That address is taken.')
    expect(f.error('nick')).toBe('Taken too.')
    expect(document.activeElement).toBe(f.q('.email input'))
    type(f.q('.nick input'), 'adal')
    await settle()
    expect(f.error('nick')).toBeUndefined()
    expect(f.error('email')).toBe('That address is taken.')
    ;(f.exposed.clear as () => void)()
    await settle()
    expect(f.error('email')).toBeUndefined()
  })

  it('validates on demand with validate()', async () => {
    const f = render()
    expect((f.exposed.validate as () => boolean)()).toBe(false)
    await settle()
    expect(f.error('email')).toBe('This field is required.')
  })

  it('while disabled disables its fields and ignores a submit', async () => {
    const f = render({ disabled: true })
    expect(f.q('.email input').disabled).toBe(true)
    expect(f.q('.age input').disabled).toBe(true)
    f.q<HTMLFormElement>('form').requestSubmit()
    await settle()
    expect(f.exposed.failed).toEqual([])
    expect(f.exposed.submitted).toEqual([])
    expect(f.error('email')).toBeUndefined()
  })
})
