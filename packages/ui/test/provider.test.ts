import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { setMatchMedia } from '../src/adaptive/detect.ts'
import { fakeMedia } from './fake-media.ts'
import { mount } from './mount.ts'
import Bare from './fixtures/Bare.vue'
import ProviderCase from './fixtures/ProviderCase.vue'

let media: ReturnType<typeof fakeMedia>
beforeEach(() => {
  media = fakeMedia({ '(pointer: coarse)': false })
  setMatchMedia(media.mq)
})
afterEach(() => setMatchMedia(undefined))

const probe = (el: HTMLElement) => el.querySelector('.probe') as HTMLElement

describe('WeftProvider', () => {
  it('sets the surface and the detected input on its element', () => {
    const { el } = mount(ProviderCase, { outerSurface: 'landing' })
    const outer = el.querySelector('.outer') as HTMLElement
    expect(outer.classList.contains('weft-provider')).toBe(true)
    expect(outer.dataset.weftSurface).toBe('landing')
    expect(outer.dataset.weftInput).toBe('pointer')
    expect(probe(el).dataset.surface).toBe('landing')
  })

  it('lets an inner provider override one axis and inherit the other', () => {
    const { el } = mount(ProviderCase, { outerInput: 'touch', nested: true, innerSurface: 'app' })
    expect(probe(el).dataset.input).toBe('touch')
    expect(probe(el).dataset.surface).toBe('app')
  })

  it('passes live input changes through nested providers', async () => {
    const { el } = mount(ProviderCase, { outerSurface: 'landing', nested: true, innerSurface: 'app' })
    expect(probe(el).dataset.input).toBe('pointer')
    media.set('(pointer: coarse)', true)
    await nextTick()
    expect(probe(el).dataset.input).toBe('touch')
    expect((el.querySelector('.inner') as HTMLElement).dataset.weftInput).toBe('touch')
  })

  it('keeps live detection after the first component that used it unmounts', async () => {
    mount(Bare).unmount()
    const { el } = mount(Bare)
    media.set('(pointer: coarse)', true)
    await nextTick()
    expect(probe(el).dataset.input).toBe('touch')
  })

  it('defaults to the detected input and the app surface without a provider', () => {
    const { el } = mount(Bare)
    expect(probe(el).dataset.input).toBe('pointer')
    expect(probe(el).dataset.surface).toBe('app')
  })
})
