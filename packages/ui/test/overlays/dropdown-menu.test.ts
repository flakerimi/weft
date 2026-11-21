import { afterEach, describe, expect, it } from 'vitest'
import MenuCases from '../fixtures/overlays/MenuCases.vue'
import InPlaceMenu from '../fixtures/overlays/InPlaceMenu.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, $$, active, key, locked, overlays, press, reset, settle } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(MenuCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const menus = () => $$('[role="menu"]')
const menu = () => menus()[0]!
const items = (el: Element = menu()) => $$('[role^="menuitem"]', el)
const named = (label: string, el: Element = menu()) => items(el).find((i) => i.querySelector('.weft-menu__text')!.textContent!.trim() === label)!
const trigger = () => $('.trigger')
async function openWith(k?: string) {
  trigger().focus()
  if (k) key(k)
  else trigger().click()
  await settle()
}

describe('DropdownMenu', () => {
  it('is a menu button: the trigger says it has a menu and whether it is open', async () => {
    await cases()
    expect(trigger().getAttribute('aria-haspopup')).toBe('menu')
    expect(trigger().getAttribute('aria-expanded')).toBe('false')
    await openWith()
    expect(trigger().getAttribute('aria-expanded')).toBe('true')
    expect(overlays()!.contains(menu())).toBe(true)
    expect(menu().getAttribute('aria-labelledby')).toBe(trigger().id)
    expect(trigger().getAttribute('aria-controls')).toBe(menu().id)
    expect(locked()).toBe(true)
  })

  it('opens on a press, Enter or ArrowDown with the first item focused, and on ArrowUp with the last', async () => {
    await cases()
    await openWith()
    expect(active()).toBe(named('New'))
    key('Escape')
    await settle()
    await openWith('ArrowDown')
    expect(active()).toBe(named('New'))
    key('Escape')
    await settle()
    await openWith('ArrowUp')
    expect(active()).toBe(named('Delete')) // the last that can be chosen: Sync is loading
  })

  it('draws groups, a group label, separators and roles for each kind of item', async () => {
    await cases()
    await openWith()
    const groups = $$('[role="group"]', menu())
    expect(groups).toHaveLength(3)
    expect(document.getElementById(groups[0]!.getAttribute('aria-labelledby')!)!.textContent).toBe('File')
    expect($$('[role="separator"]', menu())).toHaveLength(2)
    expect(items().map((i) => i.getAttribute('role'))).toEqual([
      'menuitem', 'menuitem', 'menuitem', 'menuitem',
      'menuitemcheckbox', 'menuitemradio', 'menuitemradio',
      'menuitem', 'menuitem', 'menuitem',
    ])
    expect(named('Open').getAttribute('aria-disabled')).toBe('true')
    expect(named('Show grid').getAttribute('aria-checked')).toBe('false')
    expect(named('By name').getAttribute('aria-checked')).toBe('true')
    expect(named('Share').getAttribute('aria-haspopup')).toBe('menu')
    expect(named('Share').getAttribute('aria-expanded')).toBe('false')
    expect(named('Delete').dataset.tone).toBe('danger')
    expect($('svg.weft-icon', named('New'))).not.toBeNull()
  })

  it('moves with the arrows, looping, Home and End, reaching a disabled item', async () => {
    await cases()
    await openWith()
    key('ArrowDown')
    expect(active()).toBe(named('Open')) // disabled, still focusable (APG)
    key('ArrowUp')
    key('ArrowUp')
    expect(active()).toBe(named('Sync'))
    key('ArrowDown')
    expect(active()).toBe(named('New'))
    key('End')
    expect(active()).toBe(named('Sync'))
    key('Home')
    expect(active()).toBe(named('New'))
  })

  it('jumps to an item by typing the start of its label', async () => {
    await cases()
    await openWith()
    key('s')
    expect(active()).toBe(named('Show grid'))
    key('h')
    expect(active()).toBe(named('Show grid'))
    await new Promise((r) => setTimeout(r, 600))
    key('d')
    expect(active()).toBe(named('Delete'))
  })

  it('activates an item with Enter, Space or a press, then closes and returns focus to the trigger', async () => {
    const m = await cases()
    await openWith()
    key('Enter')
    await settle()
    expect(m.exposed.log).toEqual(['new'])
    expect(menus()).toHaveLength(0)
    expect(active()).toBe(trigger())
    await openWith()
    key('End')
    key('ArrowUp')
    key(' ')
    await settle()
    await openWith()
    named('Rename').click()
    await settle()
    expect(m.exposed.log).toEqual(['new', 'delete', 'rename'])
    expect(m.exposed.open).toBe(false)
  })

  it('opens from its trigger by Enter and by Space, as a native button does', async () => {
    const m = await cases()
    expect(pressKey(trigger(), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(active()).toBe(named('New'))
    key('Escape')
    await settle()
    expect(pressKey(trigger(), ' ')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(true)
  })

  it('shows shortcuts with Kbd, a description, a loading item that cannot be chosen, and color as tone', async () => {
    const m = await cases()
    await openWith()
    const item = named('New')
    expect($$('.weft-kbd', item).map((k) => k.textContent!.trim())).toHaveLength(2)
    const description = $('.weft-menu__description', item)
    expect(description.textContent).toBe('An empty file')
    expect(item.getAttribute('aria-describedby')).toBe(description.id)
    const sync = named('Sync')
    expect(sync.getAttribute('aria-busy')).toBe('true')
    expect(sync.getAttribute('aria-disabled')).toBe('true')
    expect($('.weft-spinner', sync)).not.toBeNull()
    expect(sync.dataset.tone).toBe('warning')
    // A link while loading has no address: no Ctrl-click or middle click follows it.
    expect(sync.tagName).toBe('DIV')
    expect(sync.hasAttribute('href')).toBe(false)
    sync.click()
    await settle()
    expect(m.exposed.log).toEqual([])
    expect(m.exposed.open).toBe(true)
  })

  it('stays open after an item whose onSelect prevents the default, and ignores a disabled item', async () => {
    const m = await cases()
    await openWith()
    named('Open').click()
    named('Pin').click()
    await settle()
    expect(m.exposed.log).toEqual(['pin'])
    expect(m.exposed.open).toBe(true)
  })

  it('toggles a checkbox item and chooses a radio item through their callbacks', async () => {
    const m = await cases()
    await openWith()
    named('Show grid').click()
    await settle()
    expect(m.exposed.grid).toBe(true)
    await openWith()
    expect(named('Show grid').getAttribute('aria-checked')).toBe('true')
    named('By date').click()
    await settle()
    expect(m.exposed.sort).toBe('date')
  })

  it('closes on Escape and on Tab with focus back on the trigger, and on a press outside', async () => {
    const m = await cases()
    await openWith()
    key('Escape')
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe(trigger())
    await openWith()
    const tab = key('Tab')
    await settle()
    expect(tab.defaultPrevented).toBe(true)
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe(trigger())
    await openWith()
    press($('.after'))
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(locked()).toBe(false)
  })

  it('shows a focus ring on the focused item and fades out when it closes', async () => {
    const unstyle = loadStyles()
    try {
      const m = await cases()
      const ring = ruleFor('.weft-menu__item:focus-visible')!.style
      expect(ring.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      await openWith()
      const el = menu()
      el.style.transitionDuration = '40ms'
      key('Escape')
      await settle()
      expect(m.exposed.open).toBe(false)
      expect(el.parentElement!.dataset.state).toBe('closed')
      expect(active()).toBe(trigger())
      await new Promise((r) => setTimeout(r, 120))
      expect(el.isConnected).toBe(false)
    } finally {
      unstyle()
    }
  })

  describe('submenus', () => {
    async function toShare() {
      await openWith()
      named('Share').focus()
    }

    it('open with ArrowRight, Enter or a press, focusing their first item, and close with ArrowLeft back to the parent item', async () => {
      await cases()
      await toShare()
      key('ArrowRight')
      await settle()
      expect(menus()).toHaveLength(2)
      expect(named('Share').getAttribute('aria-expanded')).toBe('true')
      expect(active()).toBe(named('Copy link', menus()[1]))
      key('ArrowDown')
      expect(active()).toBe(named('Email', menus()[1]))
      key('ArrowLeft')
      await settle()
      expect(menus()).toHaveLength(1)
      expect(active()).toBe(named('Share'))
      key('Enter')
      await settle()
      expect(active()).toBe(named('Copy link', menus()[1]))
    })

    it('close alone on Escape; an item chosen in one closes every level', async () => {
      const m = await cases()
      await toShare()
      key('ArrowRight')
      await settle()
      key('Escape')
      await settle()
      expect(menus()).toHaveLength(1)
      expect(active()).toBe(named('Share'))
      key('ArrowRight')
      await settle()
      key('ArrowDown')
      key('Enter')
      await settle()
      expect(m.exposed.log).toEqual(['email'])
      expect(menus()).toHaveLength(0)
      expect(active()).toBe(trigger())
      expect(locked()).toBe(false)
    })

    it('open on hover without taking focus from the parent', async () => {
      await cases()
      await openWith()
      named('Share').dispatchEvent(new PointerEvent('pointermove', { bubbles: true, pointerType: 'mouse' }))
      await new Promise((r) => setTimeout(r, 200))
      await settle()
      expect(menus()).toHaveLength(2)
      expect(active()).toBe(named('Share'))
    })

    it('mirror right to left: ArrowLeft opens, ArrowRight closes', async () => {
      await cases({ dir: 'rtl' })
      await toShare()
      key('ArrowLeft')
      await settle()
      expect(menus()).toHaveLength(2)
      key('ArrowRight')
      await settle()
      expect(menus()).toHaveLength(1)
    })

    it('a press outside every level closes them all', async () => {
      const m = await cases()
      await toShare()
      key('ArrowRight')
      await settle()
      press($('.after'))
      await settle()
      expect(m.exposed.open).toBe(false)
      expect(menus()).toHaveLength(0)
      expect(active()).not.toBe(trigger()) // a press elsewhere: no focus comes back
    })
  })

  describe('on touch', () => {
    it('renders as a bottom sheet over a backdrop, with large items: 48 px, past the 44 px touch minimum', async () => {
      const unstyle = loadStyles()
      try {
        await cases({ input: 'touch' })
        await openWith()
        expect(menu().dataset.form).toBe('sheet')
        expect(menu().classList.contains('weft-sheet')).toBe(true)
        expect($('.weft-overlay-backdrop')).not.toBeNull()
        expect(active()).toBe(named('New'))
        expect(px(getComputedStyle(named('New')).minBlockSize)).toBe(48)
        expect(menu().getAttribute('aria-modal')).toBe('true')
      } finally {
        unstyle()
      }
    })

    it('opens a submenu as a sheet above, with a back item that returns to the parent', async () => {
      const m = await cases({ input: 'touch' })
      await openWith()
      named('Share').click()
      await settle()
      expect(menus()).toHaveLength(2)
      const sub = menus()[1]!
      expect(sub.dataset.form).toBe('sheet')
      expect(items(sub)[0]!.querySelector('.weft-menu__text')!.textContent!.trim()).toBe('Share')
      expect(items(sub)[0]!.classList.contains('weft-menu__back')).toBe(true)
      items(sub)[0]!.click()
      await settle()
      expect(menus()).toHaveLength(1)
      expect(active()).toBe(named('Share'))
      expect(m.exposed.open).toBe(true)
    })

    it('closes on Tab, as on pointer, though it traps focus otherwise', async () => {
      const m = await cases({ input: 'touch' })
      await openWith()
      const tab = key('Tab')
      await settle()
      expect(tab.defaultPrevented).toBe(true)
      expect(m.exposed.open).toBe(false)
      expect(active()).toBe(trigger())
    })

    it('closes on a press on the backdrop, returning focus to the trigger', async () => {
      const m = await cases({ input: 'touch' })
      await openWith()
      press($('.weft-overlay-backdrop'), 'touch')
      await settle()
      expect(m.exposed.open).toBe(false)
      expect(active()).toBe(trigger())
    })
  })
})

describe('DropdownMenu with :portal="false"', () => {
  it('opens where it stands, not in the overlay layer, and still works as a menu', async () => {
    const m = mount(InPlaceMenu)
    unmount = m.unmount
    await settle()
    $('.trigger').click()
    await settle()
    const panel = $('[role="menu"]')
    expect(panel).not.toBeNull()
    expect($('.place').contains(panel)).toBe(true)
    expect(overlays()).toBeNull()
    expect(panel.dataset.positioned).toBe('true')
    expect(active()?.textContent?.trim()).toBe('Open device')
    key('Escape')
    await settle()
    expect($('[role="menu"]')).toBeNull()
  })

  it('lines its items up from the start edge, whatever the place it is drawn in aligns', async () => {
    const unstyle = loadStyles()
    try {
      const m = mount(InPlaceMenu, { defaultOpen: true })
      unmount = m.unmount
      await settle()
      ;($('.place') as HTMLElement).style.textAlign = 'end'
      expect(getComputedStyle($('[role="menu"]')).textAlign).toBe('start')
    } finally {
      unstyle()
    }
  })

  it('as a touch sheet in place, keeps the page and itself usable: nothing around it inert, no scroll lock', async () => {
    const m = mount(InPlaceMenu, { input: 'touch' })
    unmount = m.unmount
    await settle()
    $('.trigger').click()
    await settle()
    await settle()
    const sheet = $('[role="menu"]')
    expect(sheet.dataset.form).toBe('sheet')
    expect($('.place').contains(sheet)).toBe(true)
    expect(sheet.closest('[inert]')).toBeNull()
    expect(document.querySelector('[inert]')).toBeNull()
    expect(locked()).toBe(false)
    named('Reboot', sheet).click()
    await settle()
    expect(m.exposed.chosen).toBe('reboot')
    expect($('[role="menu"]')).toBeNull()
  })

  it('starting open in place locks nothing either', async () => {
    const m = mount(InPlaceMenu, { input: 'touch', defaultOpen: true })
    unmount = m.unmount
    await settle()
    await settle()
    expect(document.querySelector('[inert]')).toBeNull()
    expect(locked()).toBe(false)
  })

  it('starts open in place when it starts open', async () => {
    const m = mount(InPlaceMenu, { defaultOpen: true })
    unmount = m.unmount
    await settle()
    expect($('.place').contains($('[role="menu"]'))).toBe(true)
  })
})
