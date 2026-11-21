import { afterEach, describe, expect, it } from 'vitest'
import FeedbackCases from '../fixtures/feedback/FeedbackCases.vue'
import { mount } from '../mount.ts'
import { loadStyles } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
  localStorage.clear()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(FeedbackCases, props)
  unmount = m.unmount
  await settle()
  return m as typeof m & { exposed: { open: boolean; log: string[] } }
}
const buttonIn = (root: Element, label: string) =>
  $$<HTMLElement>('button, a', root).find((b) => b.textContent!.trim() === label || b.getAttribute('aria-label') === label)!

describe('Alert', () => {
  const alert = () => $('.weft-alert')

  it('shows a title, a description, an icon and actions, in a tone, with no live role of its own', async () => {
    const m = await cases()
    expect(alert().dataset.tone).toBe('danger')
    expect(alert().dataset.variant).toBe('soft')
    expect(alert().hasAttribute('role')).toBe(false)
    expect($('.weft-alert__title', alert()).textContent).toBe('Payment failed')
    expect($('.weft-alert__description', alert()).textContent).toBe('The card was declined.')
    expect($('.weft-alert__leading svg.weft-icon', alert())).not.toBeNull()
    buttonIn(alert(), 'Update card').click()
    expect(m.exposed.log).toEqual(['update'])
  })

  it('closes with its close button, through v-model:open', async () => {
    const m = await cases()
    const close = buttonIn(alert(), 'Close')
    expect(close.tagName).toBe('BUTTON')
    close.click()
    await settle()
    expect(m.exposed.open).toBe(false)
    expect($$('.weft-alert')).toHaveLength(1)
    expect($('.weft-alert').classList.contains('plain')).toBe(true)
  })

  it('takes its text and leading from slots', async () => {
    await cases()
    const plain = $('.weft-alert.plain')
    expect($('.custom-leading', plain)).not.toBeNull()
    expect($('.weft-alert__description', plain).innerHTML).toContain('<strong>text</strong>')
    expect(buttonIn(plain, 'Close')).toBeUndefined()
  })

  it('puts the actions beside the text when horizontal', async () => {
    await cases({ orientation: 'horizontal' })
    expect(alert().dataset.orientation).toBe('horizontal')
  })
})

describe('Banner', () => {
  const banner = () => $('.weft-banner')

  it('is a strip with a title, an icon, actions and a close button that emits close', async () => {
    const m = await cases()
    expect($('.weft-banner__title', banner()).textContent!.trim()).toBe('We are moving offices on Friday.')
    expect(buttonIn(banner(), 'Read more').getAttribute('href')).toBe('/news')
    buttonIn(banner(), 'Close').click()
    await settle()
    expect(m.exposed.log).toEqual(['banner-closed'])
    expect($('.weft-banner')).toBeNull()
  })

  it('links its title with `to`', async () => {
    await cases({ bannerTo: '/moving' })
    expect($<HTMLAnchorElement>('a.weft-banner__title').getAttribute('href')).toBe('/moving')
  })

  it('remembers being closed when it has an id, once mounted', async () => {
    await cases({ bannerId: 'move-2026' })
    buttonIn(banner(), 'Close').click()
    await settle()
    expect(localStorage.getItem('weft-banner:move-2026')).toBe('closed')
    unmount!()
    await cases({ bannerId: 'move-2026' })
    expect($('.weft-banner')).toBeNull()
    unmount!()
    // Another id still shows.
    await cases({ bannerId: 'other' })
    expect($('.weft-banner')).not.toBeNull()
  })

  it('gives its close button a 44 px hit on touch', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      const close = buttonIn(banner(), 'Close')
      expect(close.classList.contains('weft-close-button')).toBe(true)
    } finally {
      unstyle()
    }
  })
})

describe('EmptyState', () => {
  const empty = () => $('.weft-empty-state')

  it('says there is nothing yet, with a heading at the level given, an icon, actions and a footer', async () => {
    const m = await cases()
    expect($('h2.weft-empty-state__title', empty()).textContent).toBe('No customers yet')
    expect($('.weft-empty-state__description', empty()).textContent).toBe('Add one to get started.')
    expect($('.weft-empty-state__icon svg.weft-icon', empty())).not.toBeNull()
    buttonIn(empty(), 'Add customer').click()
    expect(m.exposed.log).toEqual(['add'])
    expect($('.weft-empty-state__footer a', empty()).getAttribute('href')).toBe('/help')
  })

  it('shows a spinner in place of its icon while loading, busy', async () => {
    await cases({ loading: true })
    expect(empty().getAttribute('aria-busy')).toBe('true')
    expect($('.weft-spinner', empty())).not.toBeNull()
    expect($('.weft-empty-state__icon svg.weft-icon', empty())).toBeNull()
  })
})

describe('ErrorState', () => {
  const state = () => $('.weft-error-state')

  it('says something went wrong, as an alert, with a retry button that emits retry', async () => {
    const m = await cases()
    expect(state().getAttribute('role')).toBe('alert')
    expect($('.weft-empty-state__title', state()).textContent).toBe('Something went wrong')
    buttonIn(state(), 'Try again').click()
    expect(m.exposed.log).toEqual(['retry'])
  })

  it('shows the message of an Error, a string or a wire-format error, with its code', async () => {
    await cases({ error: new Error('Network down') })
    expect($('.weft-error-state__description', state()).textContent).toBe('Network down')
    unmount!()
    await cases({ error: 'Plain words' })
    expect($('.weft-error-state__description', state()).textContent).toBe('Plain words')
    unmount!()
    await cases({ error: { error: { code: 'not_found', message: 'No such customer' } } })
    expect($('.weft-error-state__description', state()).textContent).toBe('No such customer')
    expect($('.weft-error-state__code', state()).textContent).toBe('not_found')
  })

  it('blocks a second retry while retrying', async () => {
    const m = await cases({ retrying: true })
    const retry = buttonIn(state(), 'Try again')
    expect(retry.getAttribute('aria-busy')).toBe('true')
    retry.click()
    expect(m.exposed.log).toEqual([])
  })

  it('lays its parts out centred, with room around them', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      expect(getComputedStyle(state()).textAlign).toBe('center')
      expect(px(getComputedStyle(state()).paddingTop)).toBeGreaterThan(0)
    } finally {
      unstyle()
    }
  })
})
