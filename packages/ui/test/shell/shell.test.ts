import { afterEach, describe, expect, it, vi } from 'vitest'
import HeaderCases from '../fixtures/shell/HeaderCases.vue'
import ShellCases from '../fixtures/shell/ShellCases.vue'
import { splitBar } from '../../src/shell/appShell.ts'
import { mount } from '../mount.ts'
import { press as pressKey } from '../keyboard.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { DEFAULT_VIEWPORT, setViewport } from '../viewport.ts'
import { loadStyles, mediaRule, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, active, overlays, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
  setViewport(DEFAULT_VIEWPORT)
  unmount?.()
  unmount = undefined
  ro?.restore()
  ro = undefined
  reset()
  localStorage.clear()
  vi.restoreAllMocks()
  history.replaceState(null, '', '/')
})
async function mountAt(component: typeof ShellCases, props: Record<string, unknown>, viewport?: number) {
  ro = installResizeObserver()
  const m = mount(component, props)
  unmount = m.unmount
  await settle()
  if (viewport) {
    setViewport(viewport)
    await settle()
  }
  return m
}
const shell = () => $('.weft-app-shell')
const toggle = () => $<HTMLButtonElement>('.weft-app-shell__collapse')
const sidebar = () => $('.weft-app-shell__sidebar')
const barItems = () => $$<HTMLElement>('.weft-app-shell__bar-item')

describe('AppShell, wide', () => {
  it('draws a sidebar of the navigation, a header and the main content', async () => {
    await mountAt(ShellCases, {})
    expect(shell().dataset.layout).toBe('sidebar')
    expect(sidebar().tagName).toBe('ASIDE')
    expect($('nav.weft-nav', sidebar()).dataset.orientation).toBe('vertical')
    expect($('header.weft-app-shell__header .account')).not.toBeNull()
    expect($('main.weft-app-shell__main .page')).not.toBeNull()
    expect($('.weft-app-shell__sidebar .user')!.textContent).toBe('Ada')
    const unstyle = loadStyles()
    try {
      expect(getComputedStyle(sidebar()).display).not.toBe('none')
      // Once mounted, the other layouts' parts are not rendered at all.
      expect($('.weft-app-shell__bar')).toBeNull()
      expect($('.weft-app-shell__menu')).toBeNull()
    } finally {
      unstyle()
    }
  })

  it('inside a WeftProvider marked inset (a preview of a page), draws its content in no main landmark', async () => {
    await mountAt(ShellCases, { inset: true })
    expect($('main')).toBeNull()
    expect($('div.weft-app-shell__main .page')).not.toBeNull()
  })

  it('collapses the sidebar to icons from a toggle button the keyboard reaches', async () => {
    await mountAt(ShellCases, {})
    expect(toggle().getAttribute('aria-label')).toBe('Sidebar')
    expect(toggle().getAttribute('aria-expanded')).toBe('true')
    expect(toggle().getAttribute('aria-controls')).toBe(sidebar().id)
    expect(pressKey(toggle(), 'Enter')).toBe(true)
    await settle()
    expect(shell().dataset.collapsed).toBe('true')
    expect(toggle().getAttribute('aria-label')).toBe('Sidebar') // the same name; aria-expanded says the state
    expect(toggle().getAttribute('aria-expanded')).toBe('false')
    expect($('nav.weft-nav', sidebar()).dataset.collapsed).toBe('true')
    expect($('.weft-app-shell__sidebar .user')!.textContent).toBe('A')
    expect(pressKey(toggle(), ' ')).toBe(true)
    await settle()
    expect(shell().hasAttribute('data-collapsed')).toBe(false)
  })

  it('keeps the collapsed state in localStorage, read after mounting', async () => {
    await mountAt(ShellCases, {})
    toggle().click()
    await settle()
    expect(localStorage.getItem('test:collapsed')).toBe('true')
    unmount!()
    await mountAt(ShellCases, {})
    expect(shell().dataset.collapsed).toBe('true')
  })

  it('inside a WeftProvider marked inset (a preview), neither keeps nor reads the collapsed state', async () => {
    await mountAt(ShellCases, { inset: true })
    toggle().click()
    await settle()
    expect(shell().dataset.collapsed).toBe('true')
    expect(localStorage.length).toBe(0)
    unmount!()
    localStorage.setItem('test:collapsed', 'true')
    await mountAt(ShellCases, { inset: true })
    expect(shell().dataset.collapsed).not.toBe('true')
  })

  it('keeps nothing with persist false, and survives storage that throws', async () => {
    await mountAt(ShellCases, { persist: false })
    toggle().click()
    await settle()
    expect(localStorage.length).toBe(0)
    unmount!()
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('denied')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('denied')
    })
    await mountAt(ShellCases, {})
    toggle().click()
    await settle()
    expect(shell().dataset.collapsed).toBe('true')
  })
})

describe('AppShell on a narrow touch screen', () => {
  it('puts the navigation in a bottom bar: four links and More for the rest', async () => {
    history.replaceState(null, '', '/inbox')
    await mountAt(ShellCases, { input: 'touch' }, 400)
    expect(shell().dataset.layout).toBe('bar')
    const bar = $('nav.weft-app-shell__bar')
    expect(bar.getAttribute('aria-label')).toBe('Main')
    expect(barItems().map((b) => b.textContent!.trim())).toEqual(['Home', 'Inbox', 'Calendar', 'Reports', 'More'])
    expect(barItems()[1]!.getAttribute('aria-current')).toBe('page')
    const more = barItems()[4]!
    expect(more.getAttribute('aria-haspopup')).toBe('dialog')
    more.click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    expect(sheet.dataset.side).toBe('bottom')
    const labels = $$('.weft-nav__item:not(.weft-nav__child) .weft-nav__label', sheet).map((l) => l.textContent!.trim())
    expect(labels).toEqual(['Projects', 'Settings'])
    const settings = $$<HTMLElement>('.weft-nav__item', sheet).find((l) => l.textContent!.includes('Settings'))!
    settings.addEventListener('click', (e) => e.preventDefault())
    settings.click()
    await settle()
    expect($('[role="dialog"]')).toBeNull()
  })

  it('shows the bar and hides the sidebar, with items at least 56 px tall', async () => {
    const unstyle = loadStyles()
    try {
      await mountAt(ShellCases, { input: 'touch' }, 400)
      expect(sidebar()).toBeNull() // not rendered once the layout has none
      expect(getComputedStyle($('.weft-app-shell__bar')).display).toBe('block')
      expect(toggle()).toBeNull()
      expect(px(getComputedStyle(barItems()[0]!).minBlockSize)).toBe(56)
    } finally {
      unstyle()
    }
  })

  it('needs no More button for five links or fewer', async () => {
    await mountAt(ShellCases, { input: 'touch', few: true }, 400)
    expect(barItems().map((b) => b.textContent!.trim())).toEqual(['Home', 'Inbox', 'Settings'])
  })

  it('goes back to the sidebar when the input turns to pointer and the viewport widens', async () => {
    const m = await mountAt(ShellCases, { input: 'touch' }, 400)
    ;(m.exposed as { input: string }).input = 'pointer'
    await settle()
    expect(shell().dataset.layout).toBe('drawer')
    setViewport(1200)
    await settle()
    expect(shell().dataset.layout).toBe('sidebar')
  })
})

describe('AppShell on a narrow pointer screen', () => {
  it('opens the navigation in a sheet from the menu button, closing it when an item is chosen', async () => {
    const m = await mountAt(ShellCases, {}, 400)
    expect(shell().dataset.layout).toBe('drawer')
    const button = $<HTMLButtonElement>('.weft-app-shell__menu button')
    expect(button.getAttribute('aria-label')).toBe('Menu')
    button.click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    expect(sheet.dataset.side).toBe('start')
    expect($('nav.weft-nav', sheet).dataset.orientation).toBe('vertical')
    expect($('.user', sheet)).not.toBeNull()
    const home = $$<HTMLElement>('.weft-nav__item', sheet)[0]!
    home.addEventListener('click', (e) => e.preventDefault())
    home.click()
    await settle()
    expect($('[role="dialog"]')).toBeNull()
    expect(m.exposed.chosen).toEqual(['Home'])
  })
})

describe('AppShell before mounting', () => {
  it('draws the collapsed rail when the pre-paint script marked the root', () => {
    const unstyle = loadStyles()
    try {
      const rail = ruleFor(':root[data-weft-sidebar="collapsed"] .weft-app-shell:not([data-layout]) .weft-app-shell__sidebar')
      expect(rail!.style.getPropertyValue('inline-size')).toBe('var(--weft-app-shell-rail, 4.5rem)')
      expect(ruleFor(':root[data-weft-sidebar="collapsed"] .weft-app-shell:not([data-layout]) .weft-nav__label')).toBeDefined()
    } finally {
      unstyle()
    }
  })

  it('writes no script in the browser: the page build writes the pre-paint snippet', async () => {
    const before = document.head.innerHTML
    await mountAt(ShellCases, {})
    expect(document.head.innerHTML).toBe(before)
    expect(document.querySelector('script')).toBeNull()
  })
})

describe('AppShell across a change of layout', () => {
  it('closes the drawer when the viewport widens, focusing the same item in the sidebar; it does not reopen', async () => {
    await mountAt(ShellCases, {}, 400)
    $<HTMLButtonElement>('.weft-app-shell__menu button').click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    const inbox = $$<HTMLElement>('.weft-nav__item', sheet).find((el) => el.textContent!.includes('Inbox'))!
    inbox.focus()
    setViewport(1200)
    await settle()
    await settle()
    expect(shell().dataset.layout).toBe('sidebar')
    expect($('[role="dialog"]')).toBeNull()
    expect(active()!.closest('.weft-app-shell__sidebar')).not.toBeNull()
    expect(active()!.textContent).toContain('Inbox')
    setViewport(400)
    await settle()
    await settle()
    expect(shell().dataset.layout).toBe('drawer')
    expect($('[role="dialog"]')).toBeNull()
  })

  it('closes the More sheet when the input turns to pointer, focusing the menu button', async () => {
    const m = await mountAt(ShellCases, { input: 'touch' }, 400)
    const more = barItems().find((b) => b.textContent!.includes('More'))!
    more.click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    $$<HTMLElement>('.weft-nav__item', sheet)[0]!.focus()
    ;(m.exposed as { input: string }).input = 'pointer'
    await settle()
    await settle()
    expect(shell().dataset.layout).toBe('drawer')
    expect($('[role="dialog"]')).toBeNull()
    expect(active()).toBe($('.weft-app-shell__menu button'))
  })

  it('moves focus from the collapse toggle to the menu button when the viewport narrows', async () => {
    await mountAt(ShellCases, {})
    toggle().focus()
    setViewport(400)
    await settle()
    await settle()
    expect(active()).toBe($('.weft-app-shell__menu button'))
  })

  it('renders the brand, sidebar and footer slots once in the layout drawn, once mounted', async () => {
    await mountAt(ShellCases, {})
    expect($$('.weft-app-shell__title')).toHaveLength(1)
    expect($$('.user')).toHaveLength(1)
    setViewport(400)
    await settle()
    expect(sidebar()).toBeNull()
    expect($$('.weft-app-shell__title')).toHaveLength(1)
    $<HTMLButtonElement>('.weft-app-shell__menu button').click()
    await settle()
    expect($$('.user')).toHaveLength(1)
    expect($$('nav.weft-nav')).toHaveLength(1) // the sheet's, the sidebar's gone
  })
})

describe('splitBar', () => {
  const link = (label: string) => ({ label, to: `/${label}` })
  it('keeps five links in the bar, and moves the rest of six, or any group, under More', () => {
    const five = ['a', 'b', 'c', 'd', 'e'].map(link)
    expect(splitBar(five)).toEqual({ shown: five, more: [] })
    const six = [...five, link('f')]
    expect(splitBar(six).shown.map((i) => i.label)).toEqual(['a', 'b', 'c', 'd'])
    expect(splitBar(six).more.map((i) => i.label)).toEqual(['e', 'f'])
    const group = { label: 'g', children: [link('h')] }
    expect(splitBar([link('a'), group]).more).toEqual([group])
  })
})

describe('Header', () => {
  it('is the page banner: the title links home, the middle and the actions follow', async () => {
    await mountAt(HeaderCases as typeof ShellCases, { menu: true })
    const header = $('header.weft-header')
    expect($('.weft-header__title', header).getAttribute('href')).toBe('/')
    expect($('.weft-header__center .center-link', header)).not.toBeNull()
    expect($('.weft-header__actions .action', header)).not.toBeNull()
  })

  it('when narrow with a menu slot, trades the middle for a menu button opening a sheet', async () => {
    await mountAt(HeaderCases as typeof ShellCases, { menu: true }, 400)
    expect($('.weft-header__center')).toBeNull()
    const button = $<HTMLButtonElement>('.weft-header__menu button')
    expect(button.getAttribute('aria-label')).toBe('Menu')
    button.click()
    await settle()
    const sheet = $('[role="dialog"]', overlays()!)
    $('.menu-link', sheet).click()
    await settle()
    expect($('[role="dialog"]')).toBeNull()
  })

  it('closes its menu when the viewport widens, focusing the middle; it does not reopen', async () => {
    await mountAt(HeaderCases as typeof ShellCases, { menu: true }, 400)
    $<HTMLButtonElement>('.weft-header__menu button').click()
    await settle()
    $('.menu-link', overlays()!).focus()
    setViewport(1200)
    await settle()
    await settle()
    expect($('[role="dialog"]')).toBeNull()
    expect(active()).toBe($('.weft-header__center .center-link'))
    setViewport(400)
    await settle()
    await settle()
    expect($('[role="dialog"]')).toBeNull()
  })

  it('moves focus from the middle to the menu button when the viewport narrows', async () => {
    await mountAt(HeaderCases as typeof ShellCases, { menu: true })
    $('.weft-header__center .center-link').focus()
    setViewport(400)
    await settle()
    await settle()
    expect(active()).toBe($('.weft-header__menu button'))
  })

  it('before mounting, leaves the narrow form to CSS when it has a menu', () => {
    const unstyle = loadStyles()
    try {
      expect(ruleFor('.weft-header__menu')!.style.getPropertyValue('display')).toBe('none')
      expect(mediaRule('(max-width: 639.98px)', '.weft-header[data-menu]:not([data-layout]) .weft-header__center')?.style.getPropertyValue('display')).toBe('none')
      expect(mediaRule('(max-width: 639.98px)', '.weft-header[data-menu]:not([data-layout]) .weft-header__menu')?.style.getPropertyValue('display')).toBe('contents')
    } finally {
      unstyle()
    }
  })

  it('when narrow without a menu slot, keeps the middle', async () => {
    await mountAt(HeaderCases as typeof ShellCases, {}, 400)
    expect($('.weft-header__center .center-link')).not.toBeNull()
    expect($('.weft-header__menu')).toBeNull()
  })
})

describe('Footer', () => {
  it('is the page footer, its row stacking when its own width is narrow', async () => {
    await mountAt(HeaderCases as typeof ShellCases, {})
    const footer = $('footer.weft-footer')
    expect($('.weft-footer__leading .copy', footer)).not.toBeNull()
    expect($('.weft-footer__center a', footer).getAttribute('href')).toBe('/privacy')
    expect($('.weft-footer__trailing a', footer)).not.toBeNull()
    ro!.report(footer, 400)
    await settle()
    expect(footer.dataset.width).toBe('narrow')
    const unstyle = loadStyles()
    try {
      expect(getComputedStyle($('.weft-footer__row', footer)).flexDirection).toBe('column')
    } finally {
      unstyle()
    }
  })
})
