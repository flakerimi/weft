import { afterEach, describe, expect, it } from 'vitest'
import NavCases from '../fixtures/navigation/NavCases.vue'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { DEFAULT_VIEWPORT, setViewport } from '../viewport.ts'
import { loadStyles, mediaRule, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, active, key, moveFocus, overlays, press, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
  setViewport(DEFAULT_VIEWPORT)
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
    setViewport(viewport)
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

  it('opens an item with children as a disclosure (APG Disclosure Navigation): a button over a list of links', async () => {
    const m = await cases()
    const trigger = item('Components')
    expect(trigger.tagName).toBe('BUTTON')
    expect(trigger.hasAttribute('aria-haspopup')).toBe(false)
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    const list = document.getElementById(trigger.getAttribute('aria-controls')!)!
    expect(list.tagName).toBe('UL')
    expect(nav().contains(list)).toBe(true)
    expect(list.hidden).toBe(true)
    trigger.focus()
    trigger.click()
    await settle()
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(list.hidden).toBe(false)
    expect(active()).toBe(trigger) // focus stays on the button
    expect($('[role="menu"]')).toBeNull()
    expect($$('[role^="menuitem"]')).toHaveLength(0)
    const links = $$<HTMLAnchorElement>('a', list)
    expect(links.map((l) => l.getAttribute('href'))).toEqual(['/components/button', '/components/tabs'])
    links[1]!.addEventListener('click', (e) => e.preventDefault())
    links[1]!.click()
    await settle()
    expect(m.exposed.chosen).toEqual(['Tabs'])
    expect(list.hidden).toBe(true)
  })

  it('moves into an open list with ArrowDown, through it with the arrows, Home and End; Escape closes it back to its button', async () => {
    await cases()
    const trigger = item('Components')
    trigger.focus()
    key('ArrowDown')
    await settle()
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(active()).toBe(item('Button'))
    key('ArrowDown')
    expect(active()).toBe(item('Tabs'))
    key('ArrowDown')
    expect(active()).toBe(item('Button'))
    key('End')
    expect(active()).toBe(item('Tabs'))
    key('Home')
    expect(active()).toBe(item('Button'))
    key('Escape')
    await settle()
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    expect(active()).toBe(trigger)
  })

  it('closes an open list when focus leaves it or a press lands outside', async () => {
    await cases()
    const trigger = item('Components')
    trigger.focus()
    trigger.click()
    await settle()
    moveFocus(item('Inbox'))
    await settle()
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    trigger.focus()
    trigger.click()
    await settle()
    press(document.body)
    await settle()
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
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

  it('underlines the current page in a bar that is not collapsed, where its link is the entry\'s own child', async () => {
    history.replaceState(null, '', '/inbox')
    await cases()
    const unstyle = loadStyles()
    try {
      const underline = Array.from(document.styleSheets)
        .flatMap((sheet) => Array.from(sheet.cssRules))
        .filter((rule): rule is CSSStyleRule => rule instanceof CSSStyleRule && rule.style.getPropertyValue('content') !== '')
        .flatMap((rule) => rule.selectorText.split(/\s*,\s*/))
        .filter((selector) => /\[aria-current="page"\]::after$/.test(selector) && selector.includes('[data-orientation="horizontal"]'))
        .map((selector) => selector.slice(0, -'::after'.length))
      const underlined = (el: Element) => underline.some((selector) => el.matches(selector))
      expect(item('Inbox').parentElement!.classList.contains('weft-nav__entry')).toBe(true)
      expect(underlined(item('Inbox'))).toBe(true)
      expect(underlined(item('Home'))).toBe(false)
    } finally {
      unstyle()
    }
  })

  it('stays a bar on a narrow pointer screen, scrolling sideways when it overflows', async () => {
    await cases({}, 400)
    expect(nav().dataset.form).toBe('bar')
    expect(item('Home')).toBeDefined()
    const unstyle = loadStyles()
    try {
      expect(getComputedStyle(nav()).overflowX).toBe('auto')
    } finally {
      unstyle()
    }
  })

  it('before mounting, leaves the form to CSS: the bar, or the menu button on a narrow touch screen', () => {
    const unstyle = loadStyles()
    try {
      const media = '(max-width: 639.98px) and (pointer: coarse)'
      expect(mediaRule(media, '.weft-nav[data-orientation="horizontal"]:not([data-form]) > .weft-nav__list')?.style.getPropertyValue('display')).toBe('none')
      expect(mediaRule(media, '.weft-nav[data-orientation="horizontal"]:not([data-form]) > .weft-nav__menu-wrap')?.style.getPropertyValue('display')).toBe('contents')
      expect(ruleFor('.weft-nav__menu-wrap')!.style.getPropertyValue('display')).toBe('none')
    } finally {
      unstyle()
    }
  })
})

describe('NavigationMenu on a narrow touch screen', () => {
  it('is one menu button that opens the items as a vertical list in a sheet', async () => {
    const m = await cases({ input: 'touch' }, 400)
    expect(nav().dataset.form).toBe('sheet')
    expect($$('.weft-nav__list', nav())).toHaveLength(0)
    const button = $<HTMLButtonElement>('.weft-nav__menu', nav())
    expect(button.getAttribute('aria-label')).toBe('Menu')
    expect(button.getAttribute('aria-haspopup')).toBe('dialog')
    button.click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    expect(sheet.dataset.side).toBe('start')
    const inner = $('nav.weft-nav', sheet)
    expect(inner.dataset.form).toBe('column')
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
    expect(nav().dataset.form).toBe('bar')
  })

  it('with the sheet open, turning to pointer closes it and moves focus to the same item in the bar; it does not reopen', async () => {
    const m = await cases({ input: 'touch' }, 400)
    $<HTMLButtonElement>('.weft-nav__menu', nav()).click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    item('Inbox', sheet).focus()
    ;(m.exposed as { input: string }).input = 'pointer'
    await settle()
    await settle()
    expect($('[role="dialog"]')).toBeNull()
    expect(nav().dataset.form).toBe('bar')
    expect(active()).toBe(item('Inbox'))
    ;(m.exposed as { input: string }).input = 'touch'
    await settle()
    await settle()
    expect(nav().dataset.form).toBe('sheet')
    expect($('[role="dialog"]')).toBeNull()
  })

  it('with a list open in the bar, turning to touch closes it and moves focus to the menu button', async () => {
    const m = await cases({}, 400)
    item('Components').focus()
    key('ArrowDown')
    await settle()
    expect(active()).toBe(item('Button'))
    ;(m.exposed as { input: string }).input = 'touch'
    await settle()
    await settle()
    expect(nav().dataset.form).toBe('sheet')
    expect(active()).toBe($('.weft-nav__menu', nav()))
    expect($('[role="dialog"]')).toBeNull()
    ;(m.exposed as { input: string }).input = 'pointer'
    await settle()
    expect(item('Components').getAttribute('aria-expanded')).toBe('false')
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
    expect(nav().dataset.form).toBe('column')
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
    expect(trigger.hasAttribute('aria-haspopup')).toBe(false)
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    trigger.click()
    await settle()
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    const list = document.getElementById(trigger.getAttribute('aria-controls')!)!
    expect(list.hidden).toBe(false)
    expect(list.dataset.float).toBe('true')
    expect($('[role="menu"]')).toBeNull()
  })

  it('opens the group of the current page, by the address after mounting, and follows new items', async () => {
    history.replaceState(null, '', '/components/tabs')
    const m = await cases({ orientation: 'vertical' })
    expect(item('Components').getAttribute('aria-expanded')).toBe('true')
    const items = m.exposed.items as unknown[][]
    ;(m.exposed as { items: unknown }).items = [[...items[0]!, { label: 'Guides', defaultOpen: true, children: [{ label: 'Intro', to: '/intro' }] }], items[1]!]
    await settle()
    expect(item('Guides').getAttribute('aria-expanded')).toBe('true')
  })

  it('shows a focus ring on items', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ orientation: 'vertical' })
      expect(ruleFor('.weft-nav .weft-nav__item:focus-visible')!.style.getPropertyValue('outline')).toBe('var(--weft-focus-width, 2px) solid var(--weft-color-focus, Highlight)')
    } finally {
      unstyle()
    }
  })

  it('leaves room inside the scrolling bar for focus rings and the current mark, without moving the bar', async () => {
    const unstyle = loadStyles()
    try {
      // overflow-x: auto clips at the padding box: a ring or a mark at an item's edge sits in the padding.
      const bar = ruleFor('.weft-nav[data-orientation="horizontal"]')!.style
      expect(bar.getPropertyValue('overflow-x')).toBe('auto')
      expect(bar.getPropertyValue('padding')).toBe('4px')
      expect(bar.getPropertyValue('margin')).toBe('-4px')
    } finally {
      unstyle()
    }
  })
})
