import { afterEach, describe, expect, it } from 'vitest'
import StepperCases from '../fixtures/navigation/StepperCases.vue'
import { mount } from '../mount.ts'
import { press as pressKey } from '../keyboard.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(StepperCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const steps = () => $$<HTMLButtonElement>('.weft-stepper__trigger')
const step = (title: string) => steps().find((s) => s.textContent!.includes(title))!
const states = () => $$('.weft-stepper__item').map((i) => i.dataset.state)

interface Api { next: () => void; prev: () => void; hasNext: boolean; hasPrev: boolean }

describe('Stepper', () => {
  it('is an ordered list of steps; the current one says aria-current="step"', async () => {
    await cases()
    expect($('.weft-stepper ol.weft-stepper__list')).not.toBeNull()
    expect(steps()).toHaveLength(4)
    expect(step('Address').getAttribute('aria-current')).toBe('step')
    expect(step('Shipping').hasAttribute('aria-current')).toBe(false)
    expect(states()).toEqual(['active', 'inactive', 'inactive', 'inactive'])
    expect(step('Address').textContent).toContain('Where to ship')
    expect($('.weft-stepper__content').textContent).toContain('Address form')
  })

  it('goes on with next() and back with prev(), marking the steps done', async () => {
    const m = await cases()
    const api = m.exposed.stepper as Api
    expect(api.hasPrev).toBe(false)
    expect(api.hasNext).toBe(true)
    api.next()
    await settle()
    expect(m.exposed.step).toBe('shipping')
    expect(states()).toEqual(['completed', 'active', 'inactive', 'inactive'])
    expect($('.weft-stepper__content .shipping')!.textContent).toBe('Pick Shipping')
    api.next()
    await settle()
    expect(m.exposed.step).toBe('payment')
    expect(api.hasNext).toBe(false) // Review is disabled
    api.prev()
    await settle()
    expect(m.exposed.step).toBe('shipping')
  })

  it('when linear, lets a press reach the steps done and the next one, not beyond', async () => {
    const m = await cases()
    expect(step('Shipping').disabled).toBe(false)
    expect(step('Payment').disabled).toBe(true)
    step('Shipping').click()
    await settle()
    expect(m.exposed.step).toBe('shipping')
    step('Address').click()
    await settle()
    expect(m.exposed.step).toBe('address')
  })

  it('when not linear, lets a press reach any step that is not disabled', async () => {
    const m = await cases({ linear: false })
    step('Payment').click()
    await settle()
    expect(m.exposed.step).toBe('payment')
    expect(step('Review').disabled).toBe(true)
  })

  it('moves focus between steps with the arrow keys, one tab stop; Enter chooses', async () => {
    const m = await cases({ linear: false })
    expect(steps().map((s) => s.getAttribute('tabindex'))).toEqual(['0', '-1', '-1', '-1'])
    step('Address').focus()
    key('ArrowRight')
    expect(active()).toBe(step('Shipping'))
    expect(m.exposed.step).toBe('address')
    expect(pressKey(step('Shipping'), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.step).toBe('shipping')
  })

  it('mirrors the arrow keys right to left', async () => {
    await cases({ linear: false, dir: 'rtl' })
    step('Address').focus()
    key('ArrowLeft')
    expect(active()).toBe(step('Shipping'))
  })

  it('stacks vertically when narrow', async () => {
    const ro = installResizeObserver()
    try {
      await cases()
      expect($('.weft-stepper').dataset.orientation).toBe('horizontal')
      ro.report($('.weft-stepper'), 400)
      await settle()
      expect($('.weft-stepper').dataset.orientation).toBe('vertical')
    } finally {
      ro.restore()
    }
  })

  it('shows a focus ring on the step buttons and gives them a touch hit area', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      expect(ruleFor('.weft-stepper__trigger:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      expect(ruleFor('.weft-stepper__trigger')!.style.getPropertyValue('min-block-size')).toContain('var(--weft-hit-min')
    } finally {
      unstyle()
    }
  })
})
