import { afterEach, describe, expect, it, vi } from 'vitest'
import HeaderCases from '../fixtures/shell/HeaderCases.vue'
import ShellCases from '../fixtures/shell/ShellCases.vue'
import { splitBar } from '../../src/shell/appShell.ts'
import { mount } from '../mount.ts'
import { press as pressKey } from '../keyboard.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { loadStyles } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, overlays, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
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
    ro.report(document.documentElement, viewport)
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
      expect(getComputedStyle($('.weft-app-shell__bar')).display).toBe('none')
      expect(getComputedStyle($('.weft-app-shell__menu')).display).toBe('none')
    } finally {
      unstyle()
    }
  })

  it('collapses the sidebar to icons from a toggle button the keyboard reaches', async () => {
    await mountAt(ShellCases, {})
    expect(toggle().getAttribute('aria-label')).toBe('Collapse sidebar')
    expect(toggle().getAttribute('aria-expanded')).toBe('true')
    expect(toggle().getAttribute('aria-controls')).toBe(sidebar().id)
    expect(pressKey(toggle(), 'Enter')).toBe(true)
    await settle()
    expect(shell().dataset.collapsed).toBe('true')
    expect(toggle().getAttribute('aria-label')).toBe('Expand sidebar')
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
      expect(getComputedStyle(sidebar()).display).toBe('none')
      expect(getComputedStyle($('.weft-app-shell__bar')).display).toBe('block')
      expect(getComputedStyle(toggle()).display).toBe('none')
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
    ro!.report(document.documentElement, 1200)
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
