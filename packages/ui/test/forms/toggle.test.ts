import { afterEach, describe, expect, it } from 'vitest'
import ToggleCases from '../fixtures/forms/ToggleCases.vue'
import { mount } from '../mount.ts'
import { press } from '../keyboard.ts'
import { loadStyles } from '../styles.ts'
import { px } from './helpers.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'
import { registerIcon } from '../../src/basics/icons.ts'

for (const name of ['align-left']) registerIcon(name, '<circle cx="12" cy="12" r="10"></circle>')

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(ToggleCases, props)
  unmount = m.unmount
  await settle()
  return m as typeof m & { exposed: { bold: boolean; align: unknown; styles: unknown[] } }
}
const form = () => $<HTMLFormElement>('form')
const posted = (name: string) => new FormData(form()).getAll(name)

describe('Toggle', () => {
  const bold = () => $<HTMLButtonElement>('button.bold')

  it('is a button that says whether it is pressed, toggled by a press, Enter and Space', async () => {
    const m = await cases()
    expect(bold().getAttribute('type')).toBe('button')
    expect(bold().getAttribute('aria-pressed')).toBe('false')
    expect(bold().textContent!.trim()).toBe('Bold')
    bold().click()
    await settle()
    expect(m.exposed.bold).toBe(true)
    expect(bold().getAttribute('aria-pressed')).toBe('true')
    expect(bold().dataset.active).toBe('true')
    expect(press(bold(), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.bold).toBe(false)
    expect(press(bold(), ' ')).toBe(true)
    await settle()
    expect(m.exposed.bold).toBe(true)
  })

  it('posts its name with the form while pressed, nothing while not', async () => {
    await cases()
    expect(posted('bold')).toEqual([])
    bold().click()
    await settle()
    expect(posted('bold')).toEqual(['on'])
  })

  it('is drawn ghost and neutral, soft and primary while pressed', async () => {
    await cases()
    expect(bold().dataset.variant).toBe('ghost')
    expect(bold().dataset.tone).toBe('neutral')
    bold().click()
    await settle()
    expect(bold().dataset.variant).toBe('soft')
    expect(bold().dataset.tone).toBe('primary')
  })

  it('does not toggle while loading or disabled, by press or key', async () => {
    const m = await cases({ loading: true })
    expect(bold().getAttribute('aria-busy')).toBe('true')
    bold().click()
    press(bold(), 'Enter')
    await settle()
    expect(m.exposed.bold).toBe(false)
    unmount!()
    const n = await cases({ disabled: true })
    expect(bold().disabled).toBe(true)
    bold().click()
    await settle()
    expect(n.exposed.bold).toBe(false)
  })

  it('is at least 44 px tall on touch', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      expect(px(getComputedStyle(bold()).blockSize)).toBe(44)
    } finally {
      unstyle()
    }
  })
})

describe('ToggleGroup', () => {
  const group = () => $('.align')
  const item = (label: string) => $$<HTMLButtonElement>('button', group()).find((b) => b.textContent!.trim() === label)!

  it('is a named group of pressed buttons, one tab stop, choosing one or none', async () => {
    const m = await cases()
    expect(group().getAttribute('role')).toBe('group')
    expect(group().getAttribute('aria-label')).toBe('Alignment')
    const buttons = $$<HTMLButtonElement>('button', group())
    expect(buttons.map((b) => b.getAttribute('aria-pressed'))).toEqual(['false', 'false', 'false', 'false'])
    expect(buttons.map((b) => b.getAttribute('tabindex'))).toEqual(['0', '-1', '-1', '-1'])
    expect(item('Right').disabled).toBe(true)
    item('Center').click()
    await settle()
    expect(m.exposed.align).toBe('center')
    expect(item('Center').getAttribute('aria-pressed')).toBe('true')
    expect(item('Center').getAttribute('tabindex')).toBe('0')
    item('Left').click()
    await settle()
    expect(m.exposed.align).toBe('left')
    expect(item('Center').getAttribute('aria-pressed')).toBe('false')
    // Pressing the pressed one again leaves none.
    item('Left').click()
    await settle()
    expect(m.exposed.align).toBeNull()
  })

  it('moves with the arrows over the enabled items, wrapping, mirrored right to left', async () => {
    await cases()
    item('Left').focus()
    key('ArrowRight')
    expect(active()).toBe(item('Center'))
    key('ArrowRight')
    expect(active()).toBe(item('Justify'))
    key('ArrowRight')
    expect(active()).toBe(item('Left'))
    key('End')
    expect(active()).toBe(item('Justify'))
    unmount!()
    await cases({ dir: 'rtl' })
    item('Left').focus()
    key('ArrowLeft')
    expect(active()).toBe(item('Center'))
  })

  it('moves with Up and Down when vertical', async () => {
    await cases({ orientation: 'vertical' })
    expect(group().dataset.orientation).toBe('vertical')
    item('Left').focus()
    key('ArrowDown')
    expect(active()).toBe(item('Center'))
  })

  it('with multiple, presses any number', async () => {
    const m = await cases({ multiple: true })
    const g = $('.styles')
    const b = (label: string) => $$<HTMLButtonElement>('button', g).find((x) => x.textContent!.trim() === label)!
    expect(b('Italic').getAttribute('aria-pressed')).toBe('true')
    b('Bold').click()
    await settle()
    expect(m.exposed.styles).toEqual(['italic', 'bold'])
    b('Italic').click()
    await settle()
    expect(m.exposed.styles).toEqual(['bold'])
    expect(posted('styles')).toEqual(['bold'])
  })

  it('posts the pressed value with the form', async () => {
    await cases()
    item('Justify').click()
    await settle()
    expect(posted('align')).toEqual(['justify'])
  })

  it('makes each button at least 44 px tall on touch', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      for (const b of $$('button', group())) expect(px(getComputedStyle(b).blockSize)).toBe(44)
      unmount!()
      await cases({ input: 'pointer' })
      expect(px(getComputedStyle(item('Left')).blockSize)).toBeLessThan(44)
    } finally {
      unstyle()
    }
  })

  it('disables every item when disabled', async () => {
    await cases({ disabled: true })
    expect($$<HTMLButtonElement>('button', group()).every((b) => b.disabled)).toBe(true)
  })
})
