import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useTypeahead } from '../../src/headless/useTypeahead.ts'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

const labels = ['Apple', 'Banana', 'blueberry', '', 'Cherry', 'New York', 'Newark']

function setup(list = labels) {
  const matches: number[] = []
  const scope = effectScope()
  const t = scope.run(() => useTypeahead(() => list, (i) => matches.push(i)))!
  let current = -1
  const type = (chars: string, init: KeyboardEventInit = {}) => {
    let handled = false
    for (const ch of chars) {
      handled = t.onKeydown(new KeyboardEvent('keydown', { key: ch, ...init }), current)
      if (matches.length) current = matches[matches.length - 1]!
    }
    return handled
  }
  return { t, matches, type, scope, at: (i: number) => (current = i) }
}

describe('useTypeahead', () => {
  it('jumps to the next label starting with the typed letter, any case', () => {
    const { type, matches, scope } = setup()
    expect(type('b')).toBe(true)
    expect(matches).toEqual([1])
    scope.stop()
  })

  it('cycles through labels sharing a letter when the letter repeats', () => {
    const { type, matches, scope } = setup()
    type('bbb')
    expect(matches).toEqual([1, 2, 1])
    scope.stop()
  })

  it('matches a typed word, keeping the current item while it still matches', () => {
    const { type, matches, scope } = setup()
    type('new')
    expect(matches).toEqual([5, 5, 5])
    type('a')
    expect(matches.at(-1)).toBe(6)
    scope.stop()
  })

  it('treats a space inside a word as part of it, and a first space as not its own', () => {
    const { type, matches, t, scope } = setup()
    expect(t.onKeydown(new KeyboardEvent('keydown', { key: ' ' }), -1)).toBe(false)
    type('new y')
    expect(matches.at(-1)).toBe(5)
    scope.stop()
  })

  it('starts a new word after half a second without typing', () => {
    const { type, matches, scope } = setup()
    type('c')
    vi.advanceTimersByTime(500)
    type('a')
    expect(matches).toEqual([4, 0])
    scope.stop()
  })

  it('searches from the item after the current one and wraps', () => {
    const { type, matches, at, scope } = setup()
    at(2)
    type('a')
    expect(matches).toEqual([0])
    scope.stop()
  })

  it('never matches an empty label and ignores keys with modifiers or names', () => {
    const { t, type, matches, scope } = setup(['', 'x'])
    expect(type('a')).toBe(true)
    expect(matches).toEqual([])
    expect(t.onKeydown(new KeyboardEvent('keydown', { key: 'x', ctrlKey: true }), -1)).toBe(false)
    expect(t.onKeydown(new KeyboardEvent('keydown', { key: 'ArrowDown' }), -1)).toBe(false)
    expect(matches).toEqual([])
    scope.stop()
  })

  it('clears its timer when its scope ends', () => {
    const { type, scope } = setup()
    type('a')
    expect(vi.getTimerCount()).toBe(1)
    scope.stop()
    expect(vi.getTimerCount()).toBe(0)
  })
})
