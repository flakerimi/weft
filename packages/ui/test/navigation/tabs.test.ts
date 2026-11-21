import { afterEach, describe, expect, it } from 'vitest'
import TabsCases from '../fixtures/navigation/TabsCases.vue'
import { mount } from '../mount.ts'
import { press as pressKey } from '../keyboard.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
  history.replaceState(null, '', '/')
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(TabsCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const tabs = () => $$<HTMLButtonElement>('[role="tab"]')
const tab = (label: string) => tabs().find((t) => t.textContent!.includes(label))!
const panelOf = (t: Element) => document.getElementById(t.getAttribute('aria-controls')!)!
const shown = () => $$('[role="tabpanel"]').filter((p) => !p.hasAttribute('hidden'))

describe('Tabs', () => {
  it('follows the APG tabs pattern: tablist, tabs that control panels, panels named by their tab', async () => {
    await cases()
    const list = $('[role="tablist"]')
    expect(list.getAttribute('aria-label')).toBe('Settings')
    expect(list.getAttribute('aria-orientation')).toBe('horizontal')
    expect(tabs()).toHaveLength(4)
    expect(tab('Account').getAttribute('aria-selected')).toBe('true')
    expect(tab('Password').getAttribute('aria-selected')).toBe('false')
    for (const t of tabs()) {
      const panel = panelOf(t)
      expect(panel.getAttribute('role')).toBe('tabpanel')
      expect(panel.getAttribute('aria-labelledby')).toBe(t.id)
    }
    expect(shown()).toEqual([panelOf(tab('Account'))])
    expect(shown()[0]!.getAttribute('tabindex')).toBe('0')
    expect(shown()[0]!.textContent).toContain('Your account.')
    expect(tab('Billing').disabled).toBe(true)
    expect(tab('Team').querySelector('.weft-badge')!.textContent).toBe('3')
  })

  it('is one tab stop, on the selected tab', async () => {
    await cases()
    expect(tabs().map((t) => t.getAttribute('tabindex'))).toEqual(['0', '-1', '-1', '-1'])
  })

  it('activates automatically: an arrow key moves focus and selects, skipping disabled tabs and wrapping', async () => {
    const m = await cases()
    tab('Account').focus()
    key('ArrowRight')
    await settle()
    expect(active()).toBe(tab('Password'))
    expect(m.exposed.tab).toBe('password')
    expect(shown()).toEqual([panelOf(tab('Password'))])
    key('ArrowRight')
    await settle()
    expect(active()).toBe(tab('Team'))
    key('ArrowRight')
    await settle()
    expect(active()).toBe(tab('Account'))
    key('End')
    await settle()
    expect(m.exposed.tab).toBe('team')
    key('Home')
    await settle()
    expect(m.exposed.tab).toBe('account')
  })

  it('activates manually: arrows move focus only, Enter or Space selects', async () => {
    const m = await cases({ activation: 'manual' })
    tab('Account').focus()
    key('ArrowRight')
    await settle()
    expect(active()).toBe(tab('Password'))
    expect(m.exposed.tab).toBe('account')
    expect(pressKey(tab('Password'), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.tab).toBe('password')
    key('ArrowRight')
    await settle()
    expect(pressKey(tab('Team'), ' ')).toBe(true)
    await settle()
    expect(m.exposed.tab).toBe('team')
  })

  it('mirrors Left and Right in a right-to-left document', async () => {
    const m = await cases({ dir: 'rtl' })
    tab('Account').focus()
    key('ArrowLeft')
    await settle()
    expect(m.exposed.tab).toBe('password')
  })

  it('moves with Up and Down when vertical', async () => {
    const m = await cases({ orientation: 'vertical' })
    expect($('[role="tablist"]').getAttribute('aria-orientation')).toBe('vertical')
    tab('Account').focus()
    key('ArrowDown')
    await settle()
    expect(m.exposed.tab).toBe('password')
    expect(key('ArrowRight').defaultPrevented).toBe(false)
  })

  it('selects a tab pressed with the mouse', async () => {
    const m = await cases()
    tab('Team').click()
    await settle()
    expect(m.exposed.tab).toBe('team')
    expect(tab('Team').getAttribute('tabindex')).toBe('0')
  })

  it('renders only the selected panel content by default (lazy)', async () => {
    const m = await cases()
    const log = m.exposed.mounts as string[]
    expect(log).toEqual([])
    tab('Password').click()
    await settle()
    expect(log).toEqual(['+password'])
    tab('Account').click()
    await settle()
    expect(log).toEqual(['+password', '-password'])
    expect(panelOf(tab('Password')).querySelector('.spy')).toBeNull()
  })

  it('keeps a visited panel mounted, hidden, with keep-alive', async () => {
    const m = await cases({ keepAlive: true })
    const log = m.exposed.mounts as string[]
    tab('Password').click()
    await settle()
    tab('Account').click()
    await settle()
    expect(log).toEqual(['+password'])
    const panel = panelOf(tab('Password'))
    expect(panel.hasAttribute('hidden')).toBe(true)
    expect(panel.querySelector('.spy')).not.toBeNull()
  })

  it('leaves the address alone unless hash is set', async () => {
    await cases()
    tab('Team').click()
    await settle()
    expect(location.hash).toBe('')
  })

  it('with hash, selects the tab the address names and writes the selected one into it', async () => {
    history.replaceState(null, '', '/#team')
    const m = await cases({ hash: true })
    expect(m.exposed.tab).toBe('team')
    tab('Password').click()
    await settle()
    expect(location.hash).toBe('#password')
    location.hash = '#account'
    window.dispatchEvent(new HashChangeEvent('hashchange'))
    await settle()
    expect(m.exposed.tab).toBe('account')
  })

  it('puts a vertical tab list above its panels when narrow', async () => {
    const ro = installResizeObserver()
    try {
      await cases({ orientation: 'vertical' })
      ro.report($('.weft-tabs'), 400)
      await settle()
      expect($('[role="tablist"]').getAttribute('aria-orientation')).toBe('horizontal')
      expect($('.weft-tabs').dataset.orientation).toBe('horizontal')
      ro.report($('.weft-tabs'), 900)
      await settle()
      expect($('[role="tablist"]').getAttribute('aria-orientation')).toBe('vertical')
    } finally {
      ro.restore()
    }
  })

  it('gives tabs a 44 px hit area on touch, and a focus ring', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      expect(px(getComputedStyle(tab('Account')).minBlockSize)).toBe(44)
      expect(ruleFor('.weft-tabs__tab:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
    } finally {
      unstyle()
    }
  })
})
