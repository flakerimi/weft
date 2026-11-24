import { afterEach, describe, expect, it, vi } from 'vitest'
import ModalCases from '../fixtures/overlays/ModalCases.vue'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { px } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { $, $$, active, key, locked, overlays, press, reset, settle } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown>) {
  const m = mount(ModalCases, props)
  unmount = m.unmount
  await settle()
  return m
}
async function openFromTrigger() {
  $('.trigger').focus()
  $('.trigger').click()
  await settle()
}

describe('Dialog', () => {
  it('opens from its trigger as a modal dialog named by its title and described by its description', async () => {
    const m = await cases({ case: 'dialog' })
    const trigger = $('.trigger')
    expect(trigger.getAttribute('aria-haspopup')).toBe('dialog')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    await openFromTrigger()
    expect(m.exposed.open).toBe(true)
    const dialog = $('[role="dialog"]')
    expect(overlays()!.contains(dialog)).toBe(true)
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.getElementById(dialog.getAttribute('aria-labelledby')!)!.textContent).toBe('Rename file')
    expect(document.getElementById(dialog.getAttribute('aria-describedby')!)!.textContent).toBe('Pick a new name.')
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(trigger.getAttribute('aria-controls')).toBe(dialog.id)
    expect(locked()).toBe(true)
    // The first field, not the close button that comes first.
    expect(active()).toBe($('.field'))
  })

  it('opens from its trigger by Enter and by Space', async () => {
    const m = await cases({ case: 'dialog' })
    expect(pressKey($('.trigger'), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(true)
    key('Escape')
    await settle()
    expect(pressKey($('.trigger'), ' ')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(true)
  })

  it('stays named when the header slot replaces the title', async () => {
    await cases({ case: 'dialog', header: true })
    await openFromTrigger()
    const dialog = $('[role="dialog"]')
    expect(document.getElementById(dialog.getAttribute('aria-labelledby')!)!.textContent).toBe('My heading')
    // The slot replaced the description too: nothing to describe it by.
    expect(dialog.hasAttribute('aria-describedby')).toBe(false)
  })

  it('draws no backdrop with overlay off, still modal; scrollable lets the layer scroll', async () => {
    await cases({ case: 'dialog', overlay: false, scrollable: true })
    await openFromTrigger()
    expect($('.weft-overlay-backdrop')).toBeNull()
    expect($('[role="dialog"]').getAttribute('aria-modal')).toBe('true')
    expect(locked()).toBe(true)
    expect($('[role="dialog"]').parentElement!.dataset.scrollable).toBe('true')
  })

  it('plays its leave transition before it goes, and goes at once without one', async () => {
    const unstyle = loadStyles()
    try {
      const m = await cases({ case: 'dialog' })
      await openFromTrigger()
      const dialog = $('[role="dialog"]')
      dialog.style.transitionDuration = '40ms'
      key('Escape')
      await settle()
      expect(m.exposed.open).toBe(false)
      expect(dialog.isConnected).toBe(true)
      expect(dialog.parentElement!.dataset.state).toBe('closed')
      expect(dialog.inert).toBe(true) // not to be pressed or focused on its way out
      expect(active()).toBe($('.trigger')) // focus is home already
      await new Promise((r) => setTimeout(r, 120))
      expect(dialog.isConnected).toBe(false)
      await openFromTrigger()
      const again = $('[role="dialog"]')
      again.style.transitionDuration = '0s' // as under reduced motion
      key('Escape')
      await settle()
      expect(again.isConnected).toBe(false)
    } finally {
      unstyle()
    }
  })

  it('goes at once under reduced motion, whose base.css rule zeroes every transition', async () => {
    const settings = (window as unknown as { happyDOM: { settings: { device: { prefersReducedMotion: string } } } }).happyDOM.settings.device
    const unstyle = loadStyles()
    settings.prefersReducedMotion = 'reduce'
    try {
      await cases({ case: 'dialog' })
      await openFromTrigger()
      const dialog = $('[role="dialog"]')
      dialog.style.transitionDuration = '40ms' // base.css's !important rule wins over it
      expect(getComputedStyle(dialog).transitionDuration).toBe('0ms')
      key('Escape')
      await settle()
      expect(dialog.isConnected).toBe(false)
    } finally {
      settings.prefersReducedMotion = 'no-preference'
      unstyle()
    }
  })

  it('gives its close button a touch hit area and a visible focus ring', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ case: 'dialog' })
      await openFromTrigger()
      const before = ruleFor('.weft-close-button::before')!
      expect(before.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(before.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(ruleFor('.weft-close-button')!.style.getPropertyValue('outline')).toBe('')
      // The theme's focus ring (happy-dom keeps only part of the outline shorthand).
      const ring = ruleFor(':focus-visible')!.style
      expect(ring.getPropertyValue('outline')).toBe('var(--weft-focus-width, 2px) solid var(--weft-color-focus, Highlight)')
      expect(ring.getPropertyValue('outline-offset')).toBe('2px')
    } finally {
      unstyle()
    }
  })

  it('traps focus: Tab wraps and focus elsewhere comes back', async () => {
    await cases({ case: 'dialog' })
    await openFromTrigger()
    $('.save').focus()
    const tab = key('Tab')
    expect(tab.defaultPrevented).toBe(true)
    expect(active()).toBe($('.weft-close-button'))
    $('.after').focus()
    expect(overlays()!.contains(active())).toBe(true)
  })

  it('closes on Escape, returns focus to its trigger and lets the page scroll', async () => {
    const m = await cases({ case: 'dialog' })
    await openFromTrigger()
    key('Escape')
    await settle()
    expect(m.exposed.open).toBe(false)
    expect($('[role="dialog"]')).toBeNull()
    expect(active()).toBe($('.trigger'))
    expect(locked()).toBe(false)
  })

  it('closes on a press on the backdrop and returns focus to its trigger', async () => {
    const m = await cases({ case: 'dialog' })
    await openFromTrigger()
    press($('.weft-overlay-backdrop'))
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe($('.trigger'))
  })

  it('closes from its close button and from close() in a slot', async () => {
    const m = await cases({ case: 'dialog' })
    await openFromTrigger()
    const close = $('.weft-close-button')
    expect(close.getAttribute('aria-label')).toBe('Close')
    expect(close.getAttribute('type')).toBe('button')
    close.click()
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe($('.trigger'))
    await openFromTrigger()
    $('.cancel').click()
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('stays open on Escape and the backdrop when not dismissible', async () => {
    const m = await cases({ case: 'dialog', dismissible: false })
    await openFromTrigger()
    key('Escape')
    press($('.weft-overlay-backdrop'))
    await settle()
    expect(m.exposed.open).toBe(true)
  })

  it('opens with v-model:open and returns focus to the element that had it', async () => {
    const m = await cases({ case: 'dialog' })
    $('.before').focus()
    m.exposed.open = true
    await settle()
    expect(active()).toBe($('.field'))
    key('Escape')
    await settle()
    expect(active()).toBe($('.trigger')) // its own trigger, though it opened from elsewhere
  })

  it('goes full-screen when the viewport is narrow, or when asked', async () => {
    const ro = installResizeObserver()
    try {
      await cases({ case: 'dialog' })
      await openFromTrigger()
      const dialog = $('[role="dialog"]')
      expect(dialog.hasAttribute('data-fullscreen')).toBe(false)
      ro.report(document.documentElement, 500) // the viewport's width
      await settle()
      expect(dialog.dataset.fullscreen).toBe('true')
      ro.report(document.documentElement, 900)
      await settle()
      expect(dialog.hasAttribute('data-fullscreen')).toBe(false)
    } finally {
      ro.restore()
    }
    unmount?.()
    reset()
    await cases({ case: 'dialog', fullscreen: true })
    await openFromTrigger()
    expect($('[role="dialog"]').dataset.fullscreen).toBe('true')
  })
})

describe('Sheet', () => {
  it('slides in from the end by default, modal, with the shared close behaviour', async () => {
    const m = await cases({ case: 'sheet' })
    await openFromTrigger()
    const sheet = $('[role="dialog"]')
    expect(sheet.classList.contains('weft-sheet')).toBe(true)
    expect(sheet.dataset.side).toBe('end')
    expect(sheet.getAttribute('aria-modal')).toBe('true')
    // The shared close button, with its touch hit area (CloseButton's rule, checked under Dialog).
    expect($('.weft-close-button', sheet).getAttribute('aria-label')).toBe('Close')
    expect(active()).toBe($('.field'))
    expect(locked()).toBe(true)
    press($('.weft-overlay-backdrop'))
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe($('.trigger'))
    expect(locked()).toBe(false)
  })

  it('makes its leaving panel inert, and Drawer too', async () => {
    const unstyle = loadStyles()
    try {
      for (const kind of ['sheet', 'drawer']) {
        const m = await cases({ case: kind })
        await openFromTrigger()
        const panel = $('[role="dialog"]')
        panel.style.transitionDuration = '40ms'
        key('Escape')
        await settle()
        expect(m.exposed.open, kind).toBe(false)
        expect(panel.inert, kind).toBe(true)
        await new Promise((r) => setTimeout(r, 120))
        unmount?.()
        unmount = undefined
        reset()
      }
    } finally {
      unstyle()
    }
  })

  it.each(['start', 'top', 'bottom'])('opens from the %s side', async (side) => {
    await cases({ case: 'sheet', side })
    await openFromTrigger()
    expect($('[role="dialog"]').dataset.side).toBe(side)
  })
})

describe('Drawer', () => {
  const drag = (el: Element, type: string, y: number) =>
    el.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, pointerType: 'touch', pointerId: 1, clientX: 10, clientY: y }))

  it('gives its handle a touch-sized grip and its sheet close button the same hit area as Dialog', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ case: 'drawer' })
      $('.trigger').closest('div')
      document.querySelector('.trigger')!.closest('[data-weft-input]')
      await openFromTrigger()
      const layer = $('[role="dialog"]').parentElement!
      layer.dataset.weftInput = 'touch'
      expect(px(getComputedStyle($('.weft-drawer__handle')).blockSize)).toBe(44)
    } finally {
      unstyle()
    }
  })

  it('is a bottom sheet with a handle, closed by Escape', async () => {
    const m = await cases({ case: 'drawer' })
    await openFromTrigger()
    const drawer = $('[role="dialog"]')
    expect(drawer.classList.contains('weft-sheet')).toBe(true)
    expect(drawer.dataset.side).toBe('bottom')
    expect($('.weft-drawer__handle', drawer).getAttribute('aria-hidden')).toBe('true')
    key('Escape')
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('follows a drag on its handle, closes past a quarter of its height and springs back short of it', async () => {
    const m = await cases({ case: 'drawer' })
    await openFromTrigger()
    const drawer = $('[role="dialog"]')
    Object.defineProperty(drawer, 'offsetHeight', { value: 400, configurable: true })
    const handle = $('.weft-drawer__handle')
    const now = vi.spyOn(Date, 'now').mockReturnValue(1000)
    drag(handle, 'pointerdown', 100)
    drag(handle, 'pointermove', 150)
    expect(drawer.style.getPropertyValue('--weft-drawer-drag')).toBe('50px')
    now.mockReturnValue(1500) // slowly: not a flick
    drag(handle, 'pointerup', 150)
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(drawer.style.getPropertyValue('--weft-drawer-drag')).toBe('')
    drag(handle, 'pointerdown', 100)
    drag(handle, 'pointermove', 80) // up: it does not go past open
    expect(drawer.style.getPropertyValue('--weft-drawer-drag')).toBe('0px')
    drag(handle, 'pointermove', 220)
    now.mockReturnValue(3000)
    drag(handle, 'pointerup', 220)
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe($('.trigger'))
    now.mockRestore()
  })

  it('closes on a flick, however short', async () => {
    const m = await cases({ case: 'drawer' })
    await openFromTrigger()
    const drawer = $('[role="dialog"]')
    Object.defineProperty(drawer, 'offsetHeight', { value: 400, configurable: true })
    const now = vi.spyOn(Date, 'now').mockReturnValue(1000)
    drag(drawer.querySelector('.weft-drawer__handle')!, 'pointerdown', 100)
    drag(drawer.querySelector('.weft-drawer__handle')!, 'pointermove', 140)
    now.mockReturnValue(1040) // 40 px in 40 ms
    drag(drawer.querySelector('.weft-drawer__handle')!, 'pointerup', 140)
    await settle()
    expect(m.exposed.open).toBe(false)
    now.mockRestore()
  })
})

describe('ConfirmDialog', () => {
  it('is an alert dialog that starts on Cancel, the safe answer', async () => {
    await cases({ case: 'confirm' })
    await openFromTrigger()
    const dialog = $('[role="alertdialog"]')
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.getElementById(dialog.getAttribute('aria-labelledby')!)!.textContent).toBe('Delete the file?')
    const buttons = $$('button', dialog)
    expect(buttons.map((b) => b.textContent!.trim())).toEqual(['Cancel', 'Delete'])
    expect(active()).toBe(buttons[0])
    expect(buttons[1]!.dataset.tone).toBe('danger')
    expect($('.weft-close-button', dialog)).toBeNull()
  })

  it('emits confirm and closes on its confirm button', async () => {
    const m = await cases({ case: 'confirm' })
    await openFromTrigger()
    $$('button', $('[role="alertdialog"]'))[1]!.click()
    await settle()
    expect(m.exposed.confirms).toBe(1)
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe($('.trigger'))
  })

  it('emits cancel on Cancel, on Escape and on the backdrop', async () => {
    const m = await cases({ case: 'confirm' })
    await openFromTrigger()
    $$('button', $('[role="alertdialog"]'))[0]!.click()
    await settle()
    await openFromTrigger()
    key('Escape')
    await settle()
    await openFromTrigger()
    press($('.weft-overlay-backdrop'))
    await settle()
    expect(m.exposed.cancels).toBe(3)
    expect(m.exposed.confirms).toBe(0)
    expect(m.exposed.open).toBe(false)
  })

  it('loads while the work its confirm handler returns runs, then closes', async () => {
    const m = await cases({ case: 'confirm', work: 'resolve' })
    await openFromTrigger()
    const confirm = $$('button', $('[role="alertdialog"]'))[1]!
    confirm.click()
    await settle()
    expect(confirm.getAttribute('aria-busy')).toBe('true')
    key('Escape') // nothing closes it mid-work
    await settle()
    expect(m.exposed.open).toBe(true)
    ;(m.exposed.finish as () => void)()
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(m.exposed.cancels).toBe(0)
  })

  it('stays open, ready again, when that work fails', async () => {
    const m = await cases({ case: 'confirm', work: 'reject' })
    await openFromTrigger()
    const confirm = $$('button', $('[role="alertdialog"]'))[1]!
    confirm.click()
    await settle()
    ;(m.exposed.finish as () => void)()
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(confirm.hasAttribute('aria-busy')).toBe(false)
    // Not swallowed: the error reaches the page.
    expect((m.exposed.errors as unknown[]).map((e) => (e as Error).message)).toEqual(['no'])
  })

  it('takes a handler that throws at once down the same failure path', async () => {
    const m = await cases({ case: 'confirm', work: 'throw' })
    await openFromTrigger()
    const confirm = $$('button', $('[role="alertdialog"]'))[1]!
    confirm.click()
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(confirm.hasAttribute('aria-busy')).toBe(false)
    expect((m.exposed.errors as unknown[]).map((e) => (e as Error).message)).toEqual(['at once'])
  })

  it('waits on a thenable that is not a native Promise, and fails the same way', async () => {
    const m = await cases({ case: 'confirm', work: 'thenable' })
    await openFromTrigger()
    const confirm = $$('button', $('[role="alertdialog"]'))[1]!
    confirm.click()
    await settle()
    expect(confirm.getAttribute('aria-busy')).toBe('true')
    ;(m.exposed.finish as () => void)()
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(confirm.hasAttribute('aria-busy')).toBe(false)
    expect((m.exposed.errors as unknown[]).map((e) => (e as Error).message)).toEqual(['later'])
  })
})
