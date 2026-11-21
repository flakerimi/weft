import { afterEach, describe, expect, it } from 'vitest'
import { reactive, ref, toRaw } from 'vue'
import FormProbe from '../../src/forms/Form.vue'
import FormCases from '../fixtures/forms/FormCases.vue'
import SearchForm from '../fixtures/forms/SearchForm.vue'
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

  it('submits a form with one text field and no button when Enter is pressed in the field', async () => {
    const m = mount(SearchForm)
    mounted.push(m)
    const input = m.el.querySelector('input')!
    type(input, 'weft')
    expect(press(input, 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.sent).toEqual(['weft'])
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
    // Links are not form controls, so the fieldset leaves them: they are disabled as a disabled Link is.
    for (const sel of ['a.cancel', 'a.help']) {
      const link = f.q<HTMLAnchorElement>(sel)
      expect(link.hasAttribute('href'), sel).toBe(false)
      expect(link.getAttribute('aria-disabled'), sel).toBe('true')
      expect(link.getAttribute('role'), sel).toBe('link')
      const click = new MouseEvent('click', { bubbles: true, cancelable: true })
      link.dispatchEvent(click)
      expect(click.defaultPrevented, sel).toBe(true)
      // A middle click (auxclick) opens nothing in a new tab either.
      const aux = new MouseEvent('auxclick', { bubbles: true, cancelable: true, button: 1 })
      link.dispatchEvent(aux)
      expect(aux.defaultPrevented, sel).toBe(true)
      expect(press(link, 'Enter'), sel).toBe(false)
    }
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
    // Links are disabled while it is busy: a Cancel cannot navigate mid-submit.
    expect(f.q('a.cancel').hasAttribute('href')).toBe(false)
    expect(f.q('a.help').getAttribute('aria-disabled')).toBe('true')
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

  it('keeps focus on the submit button through an async submit: busy and unavailable, not disabled', async () => {
    const f = render({ slow: true })
    type(f.q('.email input'), 'ada@example.com')
    f.q('.terms input').click()
    f.q('.plan input').click()
    await settle()
    const button = f.q<HTMLButtonElement>('button[type="submit"]')
    expect(press(button, 'Enter')).toBe(true)
    await settle()
    expect(f.exposed.submitted).toHaveLength(1)
    expect(document.activeElement).toBe(button)
    expect(actuallyDisabled(button)).toBe(false)
    expect(button.getAttribute('aria-busy')).toBe('true')
    expect(button.getAttribute('aria-disabled')).toBe('true')
    expect(button.querySelector('.weft-spinner')).not.toBeNull()
    // A second press does nothing while it runs.
    press(button, 'Enter')
    press(button, ' ')
    await settle()
    expect(f.exposed.submitted).toHaveLength(1)
    // Other buttons work while it is busy (as in Nuxt UI), so a Stop or Cancel can abort it.
    for (const [sel, count] of [['.joined button', 'checks'], ['button.stop', 'stops']] as const) {
      const other = f.q<HTMLButtonElement>(sel)
      expect(actuallyDisabled(other), sel).toBe(false)
      expect(other.hasAttribute('aria-disabled'), sel).toBe(false)
      expect(other.querySelector('.weft-spinner'), sel).toBeNull()
      other.click()
      await settle()
      expect(f.exposed[count], sel).toBe(1)
    }
    ;(f.exposed.finish as () => void)()
    await settle()
    expect(document.activeElement).toBe(button)
    expect(button.hasAttribute('aria-busy')).toBe(false)
    expect(button.hasAttribute('aria-disabled')).toBe(false)
    expect(f.q('a.cancel').getAttribute('href')).toBe('/cancel')
  })

  it('gives focus back to the field Enter was pressed in, once an async submit is done', async () => {
    const f = render({ slow: true })
    type(f.q('.email input'), 'ada@example.com')
    f.q('.terms input').click()
    f.q('.plan input').click()
    await settle()
    const nick = f.q('.nick input')
    expect(press(nick, 'Enter')).toBe(true)
    await settle()
    expect(nick.disabled).toBe(true)
    // A browser drops focus to the body when the focused control is disabled;
    // happy-dom keeps it, so drop it here as a browser would.
    const away = document.body.appendChild(document.createElement('button'))
    away.focus()
    away.remove()
    expect(document.activeElement).toBe(document.body)
    ;(f.exposed.finish as () => void)()
    await settle()
    expect(nick.disabled).toBe(false)
    expect(document.activeElement).toBe(nick)
  })

  it('gives focus back to the field it was in when a controlled loading ends', async () => {
    const on = ref(false)
    const f = render({ loading: () => on.value })
    const nick = f.q('.nick input')
    nick.focus()
    on.value = true
    await settle()
    expect(nick.disabled).toBe(true)
    // A browser drops focus to the body when the focused control is disabled; happy-dom keeps it.
    const away = document.body.appendChild(document.createElement('button'))
    away.focus()
    away.remove()
    on.value = false
    await settle()
    expect(nick.disabled).toBe(false)
    expect(document.activeElement).toBe(nick)
  })

  it('gives focus back to the field Enter was pressed in even when the handler moves it at once', async () => {
    const f = render({ slow: true, blurs: true })
    type(f.q('.email input'), 'ada@example.com')
    f.q('.terms input').click()
    f.q('.plan input').click()
    await settle()
    const nick = f.q('.nick input')
    expect(press(nick, 'Enter')).toBe(true)
    await settle()
    expect(document.activeElement).toBe(document.body)
    ;(f.exposed.finish as () => void)()
    await settle()
    expect(document.activeElement).toBe(nick)
  })

  it('submits a copy that keeps cycles, Maps and Sets, without sharing them with the state', async () => {
    const shared = { tag: 'x' }
    const loop: Record<string, unknown> = { name: 'loop' }
    loop.self = loop
    const state = reactive({ loop, map: new Map([['a', shared]]), set: new Set([shared]), when: new Date(0) })
    const sent: (typeof state)[] = []
    const m = mount(FormProbe, { state, onSubmit: () => (v: typeof state) => void sent.push(v) })
    mounted.push(m)
    m.el.querySelector('form')!.requestSubmit()
    await settle()
    const [copy] = sent
    expect(copy!.loop).not.toBe(loop)
    expect(copy!.loop.self).toBe(copy!.loop)
    expect(copy!.map).toBeInstanceOf(Map)
    expect(copy!.map).not.toBe(state.map)
    expect(copy!.map.get('a')).toEqual({ tag: 'x' })
    expect(copy!.map.get('a')).not.toBe(shared)
    expect(copy!.set).toBeInstanceOf(Set)
    // One object reached twice is one object in the copy.
    expect([...copy!.set][0]).toBe(copy!.map.get('a'))
    expect(copy!.when).toBe(toRaw(state).when)
  })
})
