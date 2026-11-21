import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { setMatchMedia } from '../src/adaptive/detect.ts'
import { fakeMedia } from './fake-media.ts'
import { mount } from './mount.ts'
import Bare from './fixtures/Bare.vue'
import LateProbe from './fixtures/LateProbe.vue'
import { seen } from './fixtures/probe-log.ts'
import ProviderCase from './fixtures/ProviderCase.vue'

let media: ReturnType<typeof fakeMedia>
beforeEach(() => {
  media = fakeMedia({ '(pointer: coarse)': false })
  setMatchMedia(media.mq)
})
afterEach(() => setMatchMedia(undefined))

const probe = (el: HTMLElement) => el.querySelector('.probe') as HTMLElement

describe('WeftProvider', () => {
  it('sets the surface on its element, and the input only when its input prop is set', () => {
    const { el } = mount(ProviderCase, { outerSurface: 'landing' })
    const outer = el.querySelector('.outer') as HTMLElement
    expect(outer.classList.contains('weft-provider')).toBe(true)
    expect(outer.dataset.weftSurface).toBe('landing')
    // The detected input is left to CSS's (pointer: coarse), so a page that is never hydrated sizes right.
    expect(outer.hasAttribute('data-weft-input')).toBe(false)
    expect(probe(el).dataset.surface).toBe('landing')
    const forced = mount(ProviderCase, { outerInput: 'touch' })
    expect((forced.el.querySelector('.outer') as HTMLElement).dataset.weftInput).toBe('touch')
  })

  it('reports the pointer until mounted, then the detected touch, with and without a provider', async () => {
    media.set('(pointer: coarse)', true)
    seen.length = 0
    const bare = mount(Bare)
    const wrapped = mount(ProviderCase, { outerSurface: 'landing' })
    // What the server drew, so a phone hydrates the same markup.
    expect(seen).toEqual(['pointer', 'pointer'])
    await nextTick()
    expect(probe(bare.el).dataset.input).toBe('touch')
    expect(probe(wrapped.el).dataset.input).toBe('touch')
  })

  it('gives a part created after mounting the detected input at once', async () => {
    media.set('(pointer: coarse)', true)
    const { el, exposed } = mount(LateProbe)
    await nextTick()
    expect((el.querySelector('.host') as HTMLElement).dataset.input).toBe('touch')
    seen.length = 0
    ;(exposed.show as () => void)()
    await nextTick()
    expect(seen).toEqual(['touch'])
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
