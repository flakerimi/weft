import { afterEach, describe, expect, it } from 'vitest'
import NavCases from '../fixtures/navigation/NavCases.vue'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, active, key, overlays, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  ro?.restore()
  ro = undefined
  reset()
  history.replaceState(null, '', '/')
})
async function cases(props: Record<string, unknown> = {}, viewport?: number) {
  ro = installResizeObserver()
  const m = mount(NavCases, props)
  unmount = m.unmount
  await settle()
  if (viewport) {
    ro.report(document.documentElement, viewport)
    await settle()
  }
  return m
}
const nav = () => $('nav.weft-nav')
const item = (label: string, from: ParentNode = nav()) =>
  $$<HTMLElement>('.weft-nav__item', from).find((el) => el.querySelector('.weft-nav__label')?.textContent!.trim() === label)!

describe('NavigationMenu, horizontal on pointer', () => {
  it('is a navigation landmark of lists of links; the current page says aria-current="page"', async () => {
    history.replaceState(null, '', '/inbox')
    await cases()
    expect(nav().getAttribute('aria-label')).toBe('Site')
    expect(nav().dataset.orientation).toBe('horizontal')
    expect($$('ul.weft-nav__list', nav())).toHaveLength(2)
    expect(item('Home').getAttribute('href')).toBe('/')
    expect(item('Docs').getAttribute('aria-current')).toBe('page') // active
    expect(item('Inbox').getAttribute('aria-current')).toBe('page') // the browser's address
    expect(item('Home').hasAttribute('aria-current')).toBe(false)
    expect(item('Old').getAttribute('aria-disabled')).toBe('true')
    expect(item('Old').hasAttribute('href')).toBe(false)
    expect(item('Inbox').querySelector('.weft-badge')!.textContent).toBe('4')
  })

  it('opens an item with children as a dropdown menu of links', async () => {
    const m = await cases()
    const trigger = item('Components')
    expect(trigger.tagName).toBe('BUTTON')
    expect(trigger.getAttribute('aria-haspopup')).toBe('menu')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    trigger.focus()
    trigger.click()
    await settle()
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    const menu = $('[role="menu"]', overlays()!)
    const links = $$('[role="menuitem"]', menu)
    expect(links.map((l) => l.getAttribute('href'))).toEqual(['/components/button', '/components/tabs'])
    expect(active()).toBe(links[0])
    links[1]!.addEventListener('click', (e) => e.preventDefault())
    links[1]!.click()
    await settle()
    expect(m.exposed.chosen).toEqual(['Tabs'])
    expect($('[role="menu"]')).toBeNull()
  })

  it('keeps every item in the tab order and moves between them with Left and Right, Home and End', async () => {
    await cases()
    expect($$('[data-weft-item]', nav()).every((el) => !el.hasAttribute('tabindex'))).toBe(true)
    item('Home').focus()
    key('ArrowRight')
    expect(active()).toBe(item('Docs'))
    key('ArrowRight')
    expect(active()).toBe(item('Components'))
    key('ArrowRight') // Old is disabled: skipped
    expect(active()).toBe(item('Inbox'))
    key('End')
    expect(active()).toBe(item('Sign out'))
    key('Home')
    expect(active()).toBe(item('Home'))
  })

  it('mirrors Left and Right right to left', async () => {
    await cases({ dir: 'rtl' })
    item('Home').focus()
    key('ArrowLeft')
    expect(active()).toBe(item('Docs'))
  })

  it('calls onSelect for a button item and emits select', async () => {
    const m = await cases()
    item('Sign out').click()
    await settle()
    expect(m.exposed.chosen).toEqual(['sign out', 'Sign out'])
  })

  it('stays a bar on a narrow pointer screen', async () => {
    await cases({}, 400)
    expect(nav().dataset.orientation).toBe('horizontal')
    expect(item('Home')).toBeDefined()
  })
})

describe('NavigationMenu on a narrow touch screen', () => {
  it('is one menu button that opens the items as a vertical list in a sheet', async () => {
    const m = await cases({ input: 'touch' }, 400)
    expect(nav().dataset.orientation).toBe('sheet')
    expect($$('.weft-nav__list', nav())).toHaveLength(0)
    const button = $<HTMLButtonElement>('.weft-nav__menu', nav())
    expect(button.getAttribute('aria-label')).toBe('Menu')
    expect(button.getAttribute('aria-haspopup')).toBe('dialog')
    button.click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    expect(sheet.dataset.side).toBe('start')
    const inner = $('nav.weft-nav', sheet)
    expect(inner.dataset.orientation).toBe('vertical')
    expect(item('Components', inner).getAttribute('aria-expanded')).toBe('false')
    // Choosing an item closes the sheet.
    const home = item('Home', inner)
    home.addEventListener('click', (e) => e.preventDefault())
    home.click()
    await settle()
    expect(m.exposed.chosen).toEqual(['Home'])
    expect($('[role="dialog"]')).toBeNull()
  })

  it('switches back to a bar when the input turns to pointer', async () => {
    const m = await cases({ input: 'touch' }, 400)
    ;(m.exposed as { input: string }).input = 'pointer'
    await settle()
    expect(nav().dataset.orientation).toBe('horizontal')
  })

  it('gives every item a 44 px hit area on touch', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      expect(px(getComputedStyle(item('Home')).minBlockSize)).toBe(44)
    } finally {
      unstyle()
    }
  })
})

describe('NavigationMenu, vertical', () => {
  it('draws children as a disclosure list, hidden until found, and headings over groups', async () => {
    await cases({ orientation: 'vertical' })
    expect(nav().dataset.orientation).toBe('vertical')
    expect($('.weft-nav__heading', nav()).textContent).toBe('More')
    const toggle = item('Components')
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    const list = document.getElementById(toggle.getAttribute('aria-controls')!)!
    expect(list.getAttribute('hidden')).toBe('until-found')
    expect(list.querySelector('a[href="/components/button"]')).not.toBeNull()
    toggle.click()
    await settle()
    expect(toggle.getAttribute('aria-expanded')).toBe('true')
    expect(list.hasAttribute('hidden')).toBe(false)
    toggle.click()
    await settle()
    expect(list.getAttribute('hidden')).toBe('until-found')
    // Find-in-page found text inside: it opens.
    list.dispatchEvent(new Event('beforematch'))
    await settle()
    expect(toggle.getAttribute('aria-expanded')).toBe('true')
  })

  it('moves with Up and Down over the visible items only', async () => {
    await cases({ orientation: 'vertical' })
    item('Docs').focus()
    key('ArrowDown')
    expect(active()).toBe(item('Components'))
    key('ArrowDown') // the children are hidden, Old is disabled
    expect(active()).toBe(item('Inbox'))
    key('ArrowUp')
    item('Components').click()
    await settle()
    item('Components').focus()
    key('ArrowDown')
    expect(active()).toBe(item('Button'))
  })

  it('collapsed, shows icons alone, keeps the labels for assistive technology, and opens children beside it', async () => {
    await cases({ orientation: 'vertical', collapsed: true })
    expect(nav().dataset.collapsed).toBe('true')
    const home = item('Home')
    expect(home.querySelector('.weft-nav__label')!.classList.contains('weft-visually-hidden')).toBe(true)
    const trigger = item('Components')
    expect(trigger.getAttribute('aria-haspopup')).toBe('menu')
    trigger.click()
    await settle()
    expect($('[role="menu"]', overlays()!)).not.toBeNull()
  })

  it('shows a focus ring on items', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ orientation: 'vertical' })
      expect(ruleFor('.weft-nav .weft-nav__item:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
    } finally {
      unstyle()
    }
  })
})
