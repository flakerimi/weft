import { afterEach, describe, expect, it } from 'vitest'
import FormCases from '../fixtures/forms/FormCases.vue'
import { actuallyDisabled, press } from '../keyboard.ts'
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
    expect(f.exposed.submitted).toEqual([{ email: 'ada@example.com', age: 36, terms: true, plan: 'free', nick: '', fruit: ['apple'], loose: '' }])
    expect(typeof (f.exposed.submitted as { age: unknown }[])[0]!.age).toBe('number')
  })

  it('submits when its submit button is pressed with Enter', async () => {
    const f = render()
    expect(press(f.q('button[type="submit"]'), 'Enter')).toBe(true)
    await settle()
    expect(f.exposed.failed).toHaveLength(1)
  })

  it('submits when Enter is pressed in a text field, with the number typed there committed first', async () => {
    const f = render()
    type(f.q('.email input'), 'ada@example.com')
    f.q('.terms input').click()
    f.q('.plan input').click()
    await settle()
    const age = f.q('.age input')
    age.focus()
    type(age, '40')
    expect(press(age, 'Enter')).toBe(true)
    await settle()
    expect(f.exposed.failed).toEqual([])
    expect((f.exposed.submitted as { age: unknown }[]).map((v) => v.age)).toEqual([40])
  })

  it('submits a copy of the values, lists and objects included, that later edits do not change', async () => {
    const f = render()
    type(f.q('.email input'), 'ada@example.com')
    f.q('.terms input').click()
    f.q('.plan input').click()
    await settle()
    await f.submit()
    const [sent] = f.exposed.submitted as { fruit: string[] }[]
    expect(sent!.fruit).toEqual(['apple'])
    ;(f.state.fruit as string[]).push('fig')
    expect(sent!.fruit).toEqual(['apple'])
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

  it('keeps a server error through the blur that follows, and checks that field again only as it changes', async () => {
    const f = render()
    ;(f.exposed.setErrors as (e: Record<string, string>) => void)({ email: 'That address is taken.' })
    await settle()
    const email = f.q('.email input')
    expect(document.activeElement).toBe(email)
    email.blur()
    await settle()
    expect(f.error('email')).toBe('That address is taken.')
    email.focus()
    type(email, 'other@example.com')
    await settle()
    expect(f.error('email')).toBeUndefined()
    // From then on it is an ordinary field again: leaving it checks it.
    type(email, 'nope')
    email.blur()
    await settle()
    expect(f.error('email')).toBe('Enter an email address.')
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

  it('while disabled disables every control in it, in a FormField or not, buttons too, and ignores a submit', async () => {
    const f = render({ disabled: true })
    expect(f.q('.email input').disabled).toBe(true)
    expect(f.q('.age input').disabled).toBe(true)
    expect(f.q('.bare input').disabled).toBe(true)
    expect(f.q('.bare').getAttribute('data-disabled')).toBe('true')
    expect(f.q('.joined input').disabled).toBe(true)
    const check = f.q<HTMLButtonElement>('.joined button')
    // Buttons are disabled by the form's disabled fieldset, as a browser has it.
    expect(check.closest('fieldset')!.disabled).toBe(true)
    expect(actuallyDisabled(check)).toBe(true)
    expect(press(check, 'Enter')).toBe(false)
    expect(actuallyDisabled(f.q<HTMLButtonElement>('button[type="submit"]'))).toBe(true)
    expect(press(f.q('.email input'), 'Enter')).toBe(false)
    f.q<HTMLFormElement>('form').requestSubmit()
    await settle()
    expect(f.exposed.failed).toEqual([])
    expect(f.exposed.submitted).toEqual([])
    expect(f.error('email')).toBeUndefined()
  })

  it('while loading ignores a submit and is busy', async () => {
    const f = render({ loading: true })
    expect(f.q('form').getAttribute('aria-busy')).toBe('true')
    f.q<HTMLFormElement>('form').requestSubmit()
    await settle()
    expect(f.exposed.failed).toEqual([])
    expect(f.error('email')).toBeUndefined()
  })

  it('stays loading while an async submit handler runs, so a second submit does nothing', async () => {
    const f = render({ slow: true })
    type(f.q('.email input'), 'ada@example.com')
    f.q('.terms input').click()
    f.q('.plan input').click()
    await settle()
    await f.submit()
    expect(f.exposed.submitted).toHaveLength(1)
    expect(f.q('form').getAttribute('aria-busy')).toBe('true')
    expect(f.q('.email input').disabled).toBe(true)
    f.q<HTMLFormElement>('form').requestSubmit()
    await settle()
    expect(f.exposed.submitted).toHaveLength(1)
    ;(f.exposed.finish as () => void)()
    await settle()
    expect(f.q('form').hasAttribute('aria-busy')).toBe(false)
    expect(f.q('.email input').disabled).toBe(false)
    await f.submit()
    expect(f.exposed.submitted).toHaveLength(2)
  })
})
