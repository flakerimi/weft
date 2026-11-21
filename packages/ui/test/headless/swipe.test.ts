import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSwipe } from '../../src/headless/swipe.ts'

afterEach(() => vi.restoreAllMocks())

const at = (y: number, x = 0, pointerId = 1) => new PointerEvent('pointermove', { pointerId, clientX: x, clientY: y })

describe('createSwipe', () => {
  it('measures the distance towards an edge, never below zero', () => {
    const s = createSwipe(() => ({ axis: 'y', sign: 1 }))
    vi.spyOn(Date, 'now').mockReturnValue(0)
    s.start(at(100))
    expect(s.move(at(150))).toBe(50)
    expect(s.move(at(80))).toBe(0)
    expect(s.move(at(100, 0, 2))).toBeNull() // another pointer
    const left = createSwipe(() => ({ axis: 'x', sign: -1 }))
    left.start(at(0, 200))
    expect(left.move(at(0, 150))).toBe(50)
  })

  it('reads a flick from the speed of the last moves, not the average', () => {
    const now = vi.spyOn(Date, 'now')
    const s = createSwipe(() => ({ axis: 'y', sign: 1 }))
    now.mockReturnValue(0)
    s.start(at(0))
    now.mockReturnValue(900)
    s.move(at(20)) // slow for most of the way
    now.mockReturnValue(940)
    s.move(at(60)) // then 40 px in 40 ms
    expect(s.end(at(60))).toEqual({ distance: 60, speed: 1 })
  })

  it('reads no flick when the last moves were slow, however fast it started', () => {
    const now = vi.spyOn(Date, 'now')
    const s = createSwipe(() => ({ axis: 'y', sign: 1 }))
    now.mockReturnValue(0)
    s.start(at(0))
    now.mockReturnValue(20)
    s.move(at(80)) // fast at first
    now.mockReturnValue(600)
    s.move(at(82))
    now.mockReturnValue(700)
    s.move(at(83))
    const end = s.end(at(83))!
    expect(end.distance).toBe(83)
    expect(end.speed).toBeLessThan(0.1)
    expect(s.end(at(83))).toBeNull() // over
  })
})
