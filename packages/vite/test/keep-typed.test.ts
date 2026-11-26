// @vitest-environment happy-dom
// What a person typed, ticked or chose in an island's server HTML before it
// hydrated survives the hydration: the control keeps it, its model gets it,
// and a submit hands it to the handler. A field nobody touched keeps the
// value the app gave its model. (Hydration writes each control's model value
// into the element, which alone would wipe what was typed.)
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Forms from './fixtures/forms-island/components/Forms.vue'
import { formsIsland } from './fixtures/forms-island/markup.ts'
import { sent } from './fixtures/forms-island/sent.ts'
import Selection from './fixtures/selection-island/components/Selection.vue'
import { selectionIsland } from './fixtures/selection-island/markup.ts'
import Dates from './fixtures/dates-island/components/Dates.vue'
import { datesIsland } from './fixtures/dates-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => {
  document.body.innerHTML = ''
  sent.length = 0
  vi.restoreAllMocks()
})

/** The Forms island's server HTML, with `touch` run on it before it hydrates. */
async function hydrate(touch: (island: Element) => void) {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${formsIsland}</main>`
  const island = document.querySelector('weft-island')!
  touch(island)
  startIslands({ Forms: () => Promise.resolve(Forms as Component) })
  await settle()
  await settle()
  expect(island.querySelector('script'), 'hydrated').toBeNull()
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
  const q = <E extends Element = HTMLInputElement>(sel: string) => island.querySelector<E>(sel)!
  const submit = async () => {
    q<HTMLButtonElement>('button[type="submit"]').click()
    await settle()
  }
  return { q, submit, output: () => q('output').textContent }
}

it('keeps text typed into a field before hydration, gives it to the model, and submits it', async () => {
  const f = await hydrate((island) => {
    island.querySelector<HTMLInputElement>('input[type="email"]')!.value = 'ada@weft.dev'
    island.querySelector<HTMLTextAreaElement>('textarea')!.value = 'Typed early'
    island.querySelector<HTMLInputElement>('.weft-checkbox__input[name="terms"]')!.checked = true
  })
  expect(f.q('input[type="email"]').value).toBe('ada@weft.dev')
  expect(f.q<HTMLTextAreaElement>('textarea').value).toBe('Typed early')
  // The model has it: the island's output draws state.email.
  expect(f.output()).toBe('ada@weft.dev|pro|30')
  await f.submit()
  expect(sent).toHaveLength(1)
  expect(sent[0]).toMatchObject({ email: 'ada@weft.dev', bio: 'Typed early', terms: true })
})

it('keeps a checkbox, a switch and a radio set before hydration', async () => {
  const f = await hydrate((island) => {
    island.querySelector<HTMLInputElement>('input[type="email"]')!.value = 'ada@weft.dev'
    island.querySelector<HTMLInputElement>('.weft-checkbox__input[name="terms"]')!.checked = true
    island.querySelector<HTMLInputElement>('.weft-checkbox__input[value="pear"]')!.checked = true
    island.querySelector<HTMLInputElement>('.weft-switch__input')!.checked = false
    island.querySelector<HTMLInputElement>('.weft-radio__input[value="free"]')!.checked = true
  })
  expect(f.q('.weft-checkbox__input[name="terms"]').checked).toBe(true)
  expect(f.q('.weft-checkbox__input[value="pear"]').checked).toBe(true)
  expect(f.q('.weft-switch__input').checked).toBe(false)
  expect(f.q('.weft-radio__input[value="free"]').checked).toBe(true)
  expect(f.q('.weft-radio__input[value="pro"]').checked).toBe(false)
  expect(f.output()).toBe('ada@weft.dev|free|30')
  await f.submit()
  expect(sent).toHaveLength(1)
  expect(sent[0]).toMatchObject({ terms: true, fruit: ['apple', 'pear'], alerts: false, plan: 'free' })
})

it('leaves a field nobody touched with the value the app gave its model', async () => {
  const f = await hydrate((island) => {
    island.querySelector<HTMLInputElement>('input[type="email"]')!.value = 'ada@weft.dev'
    island.querySelector<HTMLInputElement>('.weft-checkbox__input[name="terms"]')!.checked = true
  })
  expect(f.q<HTMLTextAreaElement>('textarea').value).toBe('Hello')
  expect(f.q('input[name="seats"]').value).toBe('2')
  expect(f.q('.weft-checkbox__input[value="apple"]').checked).toBe(true)
  expect(f.q('.weft-switch__input').checked).toBe(true)
  expect(f.q('.weft-radio__input[value="pro"]').checked).toBe(true)
  await f.submit()
  expect(sent[0]).toMatchObject({ bio: 'Hello', seats: 2, fruit: ['apple'], alerts: true, plan: 'pro', site: '' })
})

it('keeps a number, and a code typed into the pin boxes', async () => {
  const f = await hydrate((island) => {
    island.querySelector<HTMLInputElement>('input[type="email"]')!.value = 'ada@weft.dev'
    island.querySelector<HTMLInputElement>('.weft-checkbox__input[name="terms"]')!.checked = true
    island.querySelector<HTMLInputElement>('input[name="seats"]')!.value = '5'
    island.querySelector<HTMLInputElement>('.weft-pin-input__box')!.value = 'a'
  })
  expect(f.q('input[name="seats"]').value).toBe('5')
  expect(f.q('.weft-pin-input__box').value).toBe('a')
  await f.submit()
  expect(sent[0]).toMatchObject({ seats: 5, code: 'a' })
})

/** An island of `component` from `markup`, with `touch` run on it before it hydrates. */
async function hydrateOther(markup: string, name: string, component: Component, touch: (island: Element) => void) {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${markup}</main>`
  const island = document.querySelector('weft-island')!
  touch(island)
  startIslands({ [name]: () => Promise.resolve(component) })
  await settle()
  await settle()
  expect(island.querySelector('script'), 'hydrated').toBeNull()
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
  return island
}

it('keeps the text typed into a Combobox, an InputTags and a CommandPalette', async () => {
  const island = await hydrateOther(selectionIsland, 'Selection', Selection as Component, (island) => {
    island.querySelector<HTMLInputElement>('[role="combobox"].weft-input__field')!.value = 'Ada'
    island.querySelector<HTMLInputElement>('.weft-input-tags .weft-input__field, .weft-input-tags input:not([type="hidden"])')!.value = 'go'
    island.querySelector<HTMLInputElement>('.weft-command-palette__input')!.value = 'Ho'
  })
  expect(island.querySelector<HTMLInputElement>('[role="combobox"].weft-input__field')!.value).toBe('Ada')
  expect(island.querySelector<HTMLInputElement>('.weft-input-tags input:not([type="hidden"])')!.value).toBe('go')
  expect(island.querySelector<HTMLInputElement>('.weft-command-palette__input')!.value).toBe('Ho')
})

it('keeps a date typed into a DatePicker, and its model reads it', async () => {
  const island = await hydrateOther(datesIsland, 'Dates', Dates as Component, (island) => {
    island.querySelector<HTMLInputElement>('.weft-date-picker input.weft-input__field, input[placeholder="dd/mm/yyyy"]')!.value = '15/03/2024'
  })
  expect(island.querySelector<HTMLInputElement>('input[placeholder="dd/mm/yyyy"]')!.value).toBe('15/03/2024')
  expect(island.querySelector('output')!.textContent).toContain('|2024-03-15|')
})
