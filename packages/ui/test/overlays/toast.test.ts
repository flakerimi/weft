import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useToast } from '../../src/overlays/toast.ts'
import ToastCases from '../fixtures/overlays/ToastCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { $, $$, active, key, reset } from './helpers.ts'

let unmount: (() => void) | undefined
let unstyle: (() => void) | undefined
beforeEach(() => {
  vi.useFakeTimers()
  unstyle = loadStyles()
})
afterEach(() => {
  useToast().clear()
  unmount?.()
  unmount = undefined
  unstyle?.()
  vi.useRealTimers()
  reset()
})
async function toaster(props: Record<string, unknown> = {}) {
  const m = mount(ToastCases, props)
  unmount = m.unmount
  await vi.advanceTimersByTimeAsync(0)
  return m
}
const flush = () => vi.advanceTimersByTimeAsync(0)
const toasts = () => $$('.weft-toast')
const region = () => $('.weft-toaster')
const hover = (el: Element, type: 'pointerenter' | 'pointerleave') => el.dispatchEvent(new PointerEvent(type, { pointerType: 'mouse' }))

describe('Toaster and useToast', () => {
  it('shows toasts in a labelled live region in the overlay root', async () => {
    await toaster()
    expect(region().getAttribute('role')).toBe('region')
    expect(region().getAttribute('aria-label')).toBe('Notifications')
    // Two announcers, polite and assertive, there before any toast so each is heard.
    expect($('[aria-live="polite"]', region()).textContent).toBe('')
    expect($('[aria-live="assertive"]', region()).textContent).toBe('')
    expect(document.getElementById('weft-overlays')!.contains(region())).toBe(true)
    useToast().add({ title: 'Saved', description: 'The draft is safe.', tone: 'success' })
    await flush()
    expect(toasts()).toHaveLength(1)
    expect(toasts()[0]!.textContent).toContain('Saved')
    expect(toasts()[0]!.textContent).toContain('The draft is safe.')
    expect(toasts()[0]!.dataset.tone).toBe('success')
    expect(region().dataset.position).toBe('bottom-end')
  })

  it('announces background toasts politely and foreground ones (danger by default) assertively', async () => {
    await toaster()
    useToast().add({ title: 'Saved', description: 'Draft kept.' })
    await flush()
    expect($('.weft-toaster [aria-live="polite"]').textContent).toBe('Saved Draft kept.')
    useToast().add({ title: 'Payment failed', tone: 'danger' })
    await flush()
    expect($('.weft-toaster [aria-live="assertive"]').textContent).toBe('Payment failed')
    useToast().add({ title: 'Sent', type: 'foreground' })
    await flush()
    expect($('.weft-toaster [aria-live="assertive"]').textContent).toBe('Sent')
    expect($('ol', region()).hasAttribute('aria-live')).toBe(false) // one announcement each
  })

  it('restarts its time when update() or a new add() with its id changes the duration', async () => {
    await toaster()
    const t = useToast()
    const one = t.add({ title: 'Uploading', duration: 1000 })
    await flush()
    await vi.advanceTimersByTimeAsync(900)
    t.update(one.id, { title: 'Uploaded', duration: 500 })
    await vi.advanceTimersByTimeAsync(400)
    expect(toasts()).toHaveLength(1)
    t.add({ id: one.id, title: 'Uploaded', duration: 300 })
    await vi.advanceTimersByTimeAsync(250)
    expect(toasts()).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(100)
    expect(toasts()).toHaveLength(0)
  })

  it('shows its remaining time as a progress bar that stops while paused', async () => {
    await toaster()
    useToast().add({ title: 'Saved' })
    useToast().add({ title: 'No bar', progress: false })
    useToast().add({ title: 'Stays', duration: 0 })
    await flush()
    const bars = toasts().map((t) => t.querySelector<HTMLElement>('.weft-toast__progress'))
    expect(bars[0]!.getAttribute('aria-hidden')).toBe('true')
    expect(bars[0]!.style.getPropertyValue('--weft-toast-time')).toBe('4000ms')
    expect(bars.slice(1)).toEqual([null, null])
    hover(region(), 'pointerenter')
    await flush()
    expect(toasts()[0]!.dataset.paused).toBe('true')
  })

  it('leaves after --weft-duration-toast, the theme’s 4000 ms', async () => {
    expect(getComputedStyle(document.documentElement).getPropertyValue('--weft-duration-toast')).toBe('4000ms')
    await toaster()
    useToast().add({ title: 'Saved' })
    await flush()
    await vi.advanceTimersByTimeAsync(3990)
    expect(toasts()).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(20)
    expect(toasts()).toHaveLength(0)
  })

  it('takes the timeout a theme or page sets', async () => {
    const page = document.createElement('style')
    page.textContent = ':root { --weft-duration-toast: 1.5s; }'
    document.head.appendChild(page)
    await toaster()
    useToast().add({ title: 'Saved' })
    await flush()
    await vi.advanceTimersByTimeAsync(1490)
    expect(toasts()).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(20)
    expect(toasts()).toHaveLength(0)
    page.remove()
  })

  it('takes a duration per toast; 0 stays until closed', async () => {
    await toaster()
    useToast().add({ title: 'Quick', duration: 100 })
    useToast().add({ title: 'Stays', duration: 0 })
    await flush()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(toasts().map((t) => t.textContent!.includes('Stays'))).toEqual([true])
  })

  it('pauses while hovered and resumes with the time it had left', async () => {
    await toaster()
    useToast().add({ title: 'Saved' })
    await flush()
    await vi.advanceTimersByTimeAsync(3000)
    hover(region(), 'pointerenter')
    await vi.advanceTimersByTimeAsync(10_000)
    expect(toasts()).toHaveLength(1)
    hover(region(), 'pointerleave')
    await vi.advanceTimersByTimeAsync(990)
    expect(toasts()).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(20)
    expect(toasts()).toHaveLength(0)
  })

  it('pauses while focus is inside, and F8 takes focus there', async () => {
    await toaster()
    useToast().add({ title: 'Saved', actions: [{ label: 'Undo' }] })
    await flush()
    $('.elsewhere').focus()
    key('F8')
    expect(active()).toBe(region())
    $('.weft-toast button').focus()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(toasts()).toHaveLength(1)
    $('.weft-toast button').dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: $('.elsewhere') }))
    $('.elsewhere').focus()
    await vi.advanceTimersByTimeAsync(4010)
    expect(toasts()).toHaveLength(0)
  })

  it('closes from its close button, and runs an action then closes', async () => {
    await toaster()
    const undo = vi.fn()
    const onClose = vi.fn()
    useToast().add({ title: 'Deleted', actions: [{ label: 'Undo', onClick: undo }], onClose })
    useToast().add({ title: 'Other' })
    await flush()
    const [first] = toasts()
    const buttons = $$('button', first)
    expect(buttons.map((b) => b.getAttribute('aria-label') ?? b.textContent!.trim())).toEqual(['Undo', 'Close'])
    buttons[0]!.click()
    await flush()
    expect(undo).toHaveBeenCalledOnce()
    expect(onClose).toHaveBeenCalledOnce()
    expect(toasts()).toHaveLength(1)
    $('.weft-toast .weft-toast__close').click()
    await flush()
    expect(toasts()).toHaveLength(0)
  })

  it('closes the focused toast on Escape, and keeps focus in the region', async () => {
    await toaster()
    useToast().add({ title: 'Saved', actions: [{ label: 'Undo' }] })
    await flush()
    $('.weft-toast button').focus()
    const esc = key('Escape')
    await flush()
    expect(esc.defaultPrevented).toBe(true)
    expect(toasts()).toHaveLength(0)
    expect(active()).toBe(region())
  })

  it('shows the newest `max`, and updates or removes a toast by id', async () => {
    await toaster({ max: 2 })
    const t = useToast()
    const gone = vi.fn()
    t.add({ title: 'One', onClose: gone })
    t.add({ title: 'Two' })
    const three = t.add({ title: 'Three' })
    await flush()
    // The oldest is dropped, as Nuxt UI does, not kept waiting.
    expect(toasts().map((e) => $('.weft-toast__title', e).textContent)).toEqual(['Two', 'Three'])
    expect(t.toasts.value).toHaveLength(2)
    expect(gone).toHaveBeenCalledOnce()
    t.update(three.id, { title: 'Third' })
    await flush()
    expect(toasts().map((e) => $('.weft-toast__title', e).textContent)).toEqual(['Two', 'Third'])
    t.remove(three.id)
    await flush()
    expect(toasts().map((e) => $('.weft-toast__title', e).textContent)).toEqual(['Two'])
  })

  describe('on touch', () => {
    const swipe = (el: Element, type: string, x: number) =>
      el.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, pointerType: 'touch', pointerId: 7, clientX: x, clientY: 10 }))

    it('swipes away towards the edge it sits on, and springs back from a short slow drag', async () => {
      await toaster()
      useToast().add({ title: 'Saved', duration: 0 })
      await flush()
      const toast = toasts()[0]!
      Object.defineProperty(toast, 'offsetWidth', { value: 300, configurable: true })
      const now = vi.spyOn(Date, 'now').mockReturnValue(1000)
      swipe(toast, 'pointerdown', 100)
      swipe(toast, 'pointermove', 140)
      expect(toast.style.getPropertyValue('--weft-toast-swipe')).toBe('40px')
      now.mockReturnValue(2000)
      swipe(toast, 'pointerup', 140)
      await flush()
      expect(toasts()).toHaveLength(1)
      expect(toast.style.getPropertyValue('--weft-toast-swipe')).toBe('')
      now.mockReturnValue(3000)
      swipe(toast, 'pointerdown', 100)
      swipe(toast, 'pointermove', 120)
      now.mockReturnValue(3030)
      swipe(toast, 'pointermove', 160) // a flick at the end
      swipe(toast, 'pointerup', 160)
      await flush()
      expect(toasts()).toHaveLength(0)
      now.mockRestore()
    })

    it('gives its close button the touch hit area', () => {
      const before = ruleFor('.weft-close-button::before')!
      expect(before.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    })
  })
})
