import { describe, expect, it } from 'vitest'
import { effectScope } from 'vue'
import { detectInput, widthOf } from '../src/adaptive/detect.ts'
import { fakeMedia } from './fake-media.ts'

describe('detectInput', () => {
  it('is touch when the primary pointer is coarse', () => {
    expect(detectInput(fakeMedia({ '(pointer: coarse)': true }).mq).value).toBe('touch')
  })
  it('is pointer when the primary pointer is fine', () => {
    expect(detectInput(fakeMedia({ '(pointer: coarse)': false }).mq).value).toBe('pointer')
  })
  it('follows a change without a reload', () => {
    const media = fakeMedia({ '(pointer: coarse)': false })
    const input = detectInput(media.mq)
    media.set('(pointer: coarse)', true)
    expect(input.value).toBe('touch')
    media.set('(pointer: coarse)', false)
    expect(input.value).toBe('pointer')
  })
})

describe('detectInput cleanup', () => {
  it('stops listening when its scope ends', () => {
    const media = fakeMedia({ '(pointer: coarse)': false })
    const scope = effectScope()
    scope.run(() => detectInput(media.mq))
    expect(media.listenerCount('(pointer: coarse)')).toBe(1)
    scope.stop()
    expect(media.listenerCount('(pointer: coarse)')).toBe(0)
  })
})

describe('widthOf', () => {
  it.each([
    [0, 'narrow'], [639, 'narrow'], [640, 'regular'], [1023, 'regular'], [1024, 'wide'], [2000, 'wide'],
  ])('%i px is %s', (px, width) => {
    expect(widthOf(px)).toBe(width)
  })
})
