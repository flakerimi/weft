import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { overlayStack } from '../../src/headless/overlayStack.ts'
import StackCase from '../fixtures/headless/StackCase.vue'
import { mount } from '../mount.ts'
import { $, active, key, press, reset } from './dom.ts'

afterEach(reset)

const settle = async () => {
  await nextTick()
  await new Promise((r) => setTimeout(r, 0))
}
const locked = () => document.documentElement.style.overflow === 'hidden'

describe('three overlays stacked: dialog, menu, tooltip', () => {
  it('closes one layer per Escape, returns focus to each opener and unlocks scroll with the last', async () => {
    const { el, unmount } = mount(StackCase)
    const q = (s: string) => $(s, el)

    q('.open-dialog').focus()
    q('.open-dialog').click()
    await settle()
    expect(active()).toBe(q('.open-menu')) // the dialog's trap took focus
    expect(locked()).toBe(true)

    q('.open-menu').click()
    await settle()
    q('.item').focus() // into the menu, outside the dialog's element: the trap allows it
    await settle()
    expect(active()).toBe(q('.item'))
    expect(q('.tip')).not.toBeNull()
    expect(overlayStack.ids()).toHaveLength(3)

    key('Escape')
    await settle()
    expect(q('.tip')).toBeNull()
    expect(q('.menu')).not.toBeNull()
    expect(active()).toBe(q('.item'))

    key('Escape')
    await settle()
    expect(q('.menu')).toBeNull()
    expect(q('.dialog')).not.toBeNull()
    expect(active()).toBe(q('.open-menu'))
    expect(locked()).toBe(true)

    key('Escape')
    await settle()
    expect(q('.dialog')).toBeNull()
    expect(active()).toBe(q('.open-dialog'))
    expect(locked()).toBe(false)
    expect(overlayStack.ids()).toEqual([])
    unmount()
  })

  it('a press outside closes only the top layer; the trigger toggles without reopening', async () => {
    const { el, unmount } = mount(StackCase)
    const q = (s: string) => $(s, el)
    q('.open-dialog').click()
    await settle()
    q('.open-menu').click()
    await settle()
    press(q('.other')) // inside the dialog, outside the menu
    await settle()
    expect(q('.menu')).toBeNull()
    expect(q('.dialog')).not.toBeNull()

    q('.open-menu').click()
    await settle()
    press(q('.open-menu')) // the trigger is ignored as outside; its click toggles it shut
    await settle()
    expect(q('.menu')).toBeNull()
    unmount()
  })

  it('leaves the stack and unlocks when the whole thing unmounts with overlays open', async () => {
    const { el, unmount } = mount(StackCase)
    $('.open-dialog', el).click()
    await settle()
    $('.open-menu', el).click()
    await settle()
    unmount()
    await settle()
    expect(overlayStack.ids()).toEqual([])
    expect(locked()).toBe(false)
  })
})
