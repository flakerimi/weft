import { afterEach, describe, expect, it, vi } from 'vitest'
import ModalCases from '../fixtures/overlays/ModalCases.vue'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
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

  it('traps focus: Tab wraps and focus elsewhere comes back', async () => {
    await cases({ case: 'dialog' })
    await openFromTrigger()
    $('.save').focus()
    const tab = key('Tab')
    expect(tab.defaultPrevented).toBe(true)
    expect(active()).toBe($('.weft-overlay__close'))
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
    const close = $('.weft-overlay__close')
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
      ro.report($('.weft-overlay-backdrop'), 500)
      await settle()
      expect(dialog.dataset.fullscreen).toBe('true')
      ro.report($('.weft-overlay-backdrop'), 900)
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
    expect(active()).toBe($('.field'))
    expect(locked()).toBe(true)
    press($('.weft-overlay-backdrop'))
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe($('.trigger'))
    expect(locked()).toBe(false)
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
    expect($('.weft-overlay__close', dialog)).toBeNull()
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
  })
})
