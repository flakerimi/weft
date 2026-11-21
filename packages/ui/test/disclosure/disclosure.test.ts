import { afterEach, describe, expect, it } from 'vitest'
import DisclosureCases from '../fixtures/disclosure/DisclosureCases.vue'
import { mount } from '../mount.ts'
import { press as pressKey } from '../keyboard.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(DisclosureCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const trigger = () => $('.trigger')
const content = (el: Element) => document.getElementById(el.getAttribute('aria-controls')!)!
const headers = () => $$<HTMLButtonElement>('.weft-accordion__trigger')
const header = (label: string) => headers().find((h) => h.textContent!.includes(label))!

describe('Collapsible', () => {
  it('is a disclosure: the trigger says whether it is open and what it controls', async () => {
    const m = await cases()
    expect(trigger().getAttribute('aria-expanded')).toBe('false')
    const panel = content(trigger())
    expect(panel.contains($('.inside'))).toBe(true)
    // Hidden until found: rendered, so find-in-page reaches it, but not shown.
    expect(panel.getAttribute('hidden')).toBe('until-found')
    trigger().click()
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(trigger().getAttribute('aria-expanded')).toBe('true')
    expect(panel.hasAttribute('hidden')).toBe(false)
    expect($('.weft-collapsible.slotted').dataset.state).toBe('open')
  })

  it('opens and closes from the keyboard, as its trigger is a button', async () => {
    const m = await cases()
    expect(pressKey(trigger(), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(pressKey(trigger(), ' ')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('opens when find-in-page reveals its content (beforematch)', async () => {
    const m = await cases()
    content(trigger()).dispatchEvent(new Event('beforematch', { bubbles: true }))
    await settle()
    expect(m.exposed.open).toBe(true)
  })

  it('draws a trigger button from label, which a disabled collapsible keeps shut', async () => {
    await cases()
    const button = $<HTMLButtonElement>('.labelled .weft-collapsible__trigger')
    expect(button.textContent).toContain('More')
    expect(button.getAttribute('aria-expanded')).toBe('false')
    expect(button.disabled).toBe(true)
    expect(content(button).textContent).toContain('Labelled content')
  })

  it('animates its height with CSS, which reduced motion turns off', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      const rule = ruleFor('.weft-collapsible__content')!.style
      expect(rule.getPropertyValue('transition-property')).toContain('block-size')
      expect(ruleFor('.weft-collapsible__content[hidden]')!.style.getPropertyValue('block-size')).toBe('0px')
      expect(ruleFor('.weft-collapsible__trigger:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
    } finally {
      unstyle()
    }
  })
})

describe('Accordion', () => {
  it('follows the APG accordion: heading > button, expanded state, a region named by its header', async () => {
    await cases()
    const h = header('Shipping')
    expect(h.parentElement!.tagName).toBe('H3')
    expect(h.getAttribute('aria-expanded')).toBe('false')
    const region = content(h)
    expect(region.getAttribute('role')).toBe('region')
    expect(region.getAttribute('aria-labelledby')).toBe(h.id)
    expect(region.getAttribute('hidden')).toBe('until-found')
    expect(region.textContent).toContain('Ships in two days.')
    expect(content(header('Support')).querySelector('.support')!.textContent).toBe('Mail us about Support')
    expect(header('Warranty').disabled).toBe(true)
  })

  it('opens one panel at a time when single, and can close it again (collapsible)', async () => {
    const m = await cases()
    header('Shipping').click()
    await settle()
    expect(m.exposed.section).toBe('shipping')
    header('Returns').click()
    await settle()
    expect(m.exposed.section).toBe('returns')
    expect(header('Shipping').getAttribute('aria-expanded')).toBe('false')
    expect(content(header('Shipping')).getAttribute('hidden')).toBe('until-found')
    header('Returns').click()
    await settle()
    expect(m.exposed.section).toBeNull()
  })

  it('keeps the open panel open when not collapsible, and says so with aria-disabled', async () => {
    const m = await cases({ collapsible: false })
    header('Shipping').click()
    await settle()
    expect(header('Shipping').getAttribute('aria-disabled')).toBe('true')
    header('Shipping').click()
    await settle()
    expect(m.exposed.section).toBe('shipping')
  })

  it('opens several at once when multiple', async () => {
    const m = await cases({ type: 'multiple' })
    header('Shipping').click()
    header('Support').click()
    await settle()
    expect(m.exposed.section).toEqual(['shipping', 'support'])
    header('Shipping').click()
    await settle()
    expect(m.exposed.section).toEqual(['support'])
  })

  it('moves between headers with the arrows, Home and End, every header staying in the tab order', async () => {
    await cases()
    header('Shipping').focus()
    expect(key('ArrowDown').defaultPrevented).toBe(true)
    expect(active()).toBe(header('Returns'))
    key('ArrowDown') // Warranty is disabled: skipped
    expect(active()).toBe(header('Support'))
    key('ArrowDown')
    expect(active()).toBe(header('Shipping'))
    key('End')
    expect(active()).toBe(header('Support'))
    key('Home')
    expect(active()).toBe(header('Shipping'))
    expect(headers().map((h) => h.getAttribute('tabindex'))).toEqual([null, null, null, null])
  })

  it('toggles with Enter and Space', async () => {
    const m = await cases()
    expect(pressKey(header('Returns'), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.section).toBe('returns')
    expect(pressKey(header('Returns'), ' ')).toBe(true)
    await settle()
    expect(m.exposed.section).toBeNull()
  })

  it('opens the panel find-in-page reveals', async () => {
    const m = await cases()
    content(header('Returns')).dispatchEvent(new Event('beforematch', { bubbles: true }))
    await settle()
    expect(m.exposed.section).toBe('returns')
  })

  it('gives headers a touch-sized hit area and a focus ring', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      expect(ruleFor('.weft-accordion__trigger')!.style.getPropertyValue('min-block-size')).toContain('var(--weft-hit-min')
      expect(ruleFor('.weft-accordion__trigger:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
    } finally {
      unstyle()
    }
  })
})
