import { afterEach, describe, expect, it } from 'vitest'
import FeedbackCases from '../fixtures/feedback/FeedbackCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { bannerSnippet } from '../../src/feedback/banner.ts'
import { registerIcon } from '../../src/basics/icons.ts'

for (const name of ['circle-alert', 'info', 'users']) registerIcon(name, '<circle cx="12" cy="12" r="10"></circle>')
import { px } from '../forms/helpers.ts'
import { $, $$, active, reset, settle } from '../overlays/helpers.ts'

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

/**
 * How large a CloseButton's press area is, as drawn: its ::before is
 * max(100%, --weft-hit-min) each way. happy-dom computes no pseudo-elements,
 * so the rule is read and resolved against the button.
 */
function hit(el: HTMLElement): number {
  const before = ruleFor('.weft-close-button::before')!.style
  expect(before.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
  expect(before.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
  // happy-dom does not inherit custom properties into getComputedStyle: read it where it is set.
  let min = ''
  for (let at: Element | null = el; at && !min; at = at.parentElement) min = getComputedStyle(at).getPropertyValue('--weft-hit-min')
  return Math.max(px(getComputedStyle(el).blockSize), px(min || getComputedStyle(document.documentElement).getPropertyValue('--weft-hit-min')))
}

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

  it('moves focus on when closed from its button, to what follows it', async () => {
    const m = await cases()
    const close = buttonIn(alert(), 'Close')
    close.focus()
    close.click()
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe(buttonIn($('.weft-banner'), 'Read more'))
  })

  it('leaves focus alone when closed from outside', async () => {
    const m = await cases()
    const other = buttonIn($('.weft-empty-state'), 'Add customer')
    other.focus()
    ;(m.exposed as { open: boolean }).open = false
    await settle()
    expect(active()).toBe(other)
  })

  it('gives its close button a 44 px hit on touch', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      expect(hit(buttonIn(alert(), 'Close'))).toBe(44)
      unmount!()
      await cases({ input: 'pointer' })
      expect(hit(buttonIn(alert(), 'Close'))).toBeLessThan(44)
    } finally {
      unstyle()
    }
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
      expect(hit(buttonIn(banner(), 'Close'))).toBe(44)
      unmount!()
      await cases({ input: 'pointer' })
      expect(hit(buttonIn(banner(), 'Close'))).toBeLessThan(44)
    } finally {
      unstyle()
    }
  })

  it('moves focus on when closed from its button, to what follows it', async () => {
    await cases()
    const close = buttonIn(banner(), 'Close')
    close.focus()
    close.click()
    await settle()
    expect(active()).toBe(buttonIn($('.weft-empty-state'), 'Add customer'))
  })
})

describe('bannerSnippet', () => {
  it('hides the banner with that id before paint when it was closed, and survives storage that throws', () => {
    const code = bannerSnippet('a"b</script>')
    expect(code).not.toContain('</script>')
    const run = (getItem: (k: string) => string | null) => {
      const added: { textContent: string }[] = []
      new Function('localStorage', 'document', code)({ getItem }, { createElement: () => ({ textContent: '' }), head: { appendChild: (s: { textContent: string }) => added.push(s) } })
      return added.map((s) => s.textContent)
    }
    expect(run((k) => (k === 'weft-banner:a"b</script>' ? 'closed' : null))).toEqual(['.weft-banner[data-id="a\\"b</script>"]{display:none}'])
    expect(run(() => 'open')).toEqual([])
    expect(run(() => { throw new Error('denied') })).toEqual([])
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
