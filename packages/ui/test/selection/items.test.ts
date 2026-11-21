// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { fuzzyMatch, segments } from '../../src/selection/fuzzy.ts'
import { filterGroups, fold, groupsOf, labelOfValue, sameValue, score, valueOf } from '../../src/selection/items.ts'

describe('selection items', () => {
  it('reads values, labels and groups', () => {
    expect(groupsOf(['a', 'b'])).toEqual([['a', 'b']])
    expect(groupsOf([['a'], ['b']])).toEqual([['a'], ['b']])
    expect(valueOf({ label: 'A', value: 1 }, 'value')).toBe(1)
    const object = { label: 'A', id: 1 }
    expect(valueOf(object)).toBe(object)
    expect(labelOfValue([{ label: 'One', value: 1 }], 1, { valueKey: 'value' })).toBe('One')
    expect(labelOfValue([{ user: { name: 'Ada' }, id: 7 }], 7, { valueKey: 'id', labelKey: 'user.name' })).toBe('Ada')
    expect(labelOfValue([], 'free', {})).toBe('free')
  })

  it('compares values in depth, or by a key or a function', () => {
    expect(sameValue({ id: 1, n: 'a' }, { id: 1, n: 'a' })).toBe(true)
    expect(sameValue({ id: 1, n: 'a' }, { id: 1, n: 'b' })).toBe(false)
    expect(sameValue({ id: 1, n: 'a' }, { id: 1, n: 'b' }, 'id')).toBe(true)
    expect(sameValue(2, 3, (a, b) => (a as number) % 2 === (b as number) % 2)).toBe(false)
  })

  it('folds case and accents, keeping indices', () => {
    expect(fold('Crème Brûlée')).toBe('creme brulee')
    expect(fold('İstanbul')).toHaveLength('İstanbul'.length)
    expect(score('Zürich', 'zur')).toBe(1)
    expect(score('Bern', 'bern')).toBe(0)
    expect(score('Lucerne', 'cer')).toBe(2)
    expect(score('Basel', 'x')).toBeNull()
  })

  it('filters groups: the exact match, then those starting with it, then the rest; headings kept, empty groups dropped', () => {
    const groups = groupsOf([
      [{ type: 'label', label: 'Fruit' }, 'Pineapple', 'Apple', 'Apricot', 'Banana'],
      [{ type: 'label', label: 'Veg' }, 'Carrot'],
    ])
    expect(filterGroups(groups, 'ap', ['label'])).toEqual([[{ type: 'label', label: 'Fruit' }, 'Apple', 'Apricot', 'Pineapple']])
    expect(filterGroups(groups, 'apple', ['label'])[0]!.slice(1)).toEqual(['Apple', 'Pineapple'])
    expect(filterGroups(groups, '', ['label'])).toBe(groups)
    const people = [[{ name: 'Ada', role: 'admin' }, { name: 'Bob', role: 'guest' }]]
    expect(filterGroups(people, 'gue', ['name', 'role'])).toEqual([[{ name: 'Bob', role: 'guest' }]])
  })
})

describe('fuzzyMatch', () => {
  it('matches letters in order with gaps, and where', () => {
    const m = fuzzyMatch('Git: Stash', 'gtst')!
    expect(m).not.toBeNull()
    expect(segments('Git: Stash', m.ranges).filter((s) => s.match).map((s) => s.text).join('')).toBe('GtSt')
    expect(fuzzyMatch('Git: Stash', 'gx')).toBeNull()
    expect(fuzzyMatch('Settings', '')).toEqual({ score: 0, ranges: [] })
  })

  it('prefers word starts and runs over scattered letters', () => {
    const a = fuzzyMatch('Open file', 'of')!.score
    const b = fuzzyMatch('Profile', 'of')!.score
    expect(a).toBeGreaterThan(b)
    const run = fuzzyMatch('New window', 'new')!.score
    const scattered = fuzzyMatch('Narrow view window', 'new')!.score
    expect(run).toBeGreaterThan(scattered)
    expect(fuzzyMatch('newWindow', 'w')!.ranges).toEqual([[3, 4]])
  })

  it('matches each word of the query, in any order, ignoring case and accents', () => {
    expect(fuzzyMatch('Toggle dark mode', 'mode dark')).not.toBeNull()
    expect(fuzzyMatch('Café settings', 'cafe')!.ranges).toEqual([[0, 4]])
    expect(fuzzyMatch('Toggle dark mode', 'mode light')).toBeNull()
  })

  it('cuts text into matched and unmatched runs', () => {
    expect(segments('Profile', [[0, 1], [3, 5]])).toEqual([
      { text: 'P', match: true },
      { text: 'ro', match: false },
      { text: 'fi', match: true },
      { text: 'le', match: false },
    ])
    expect(segments('', [])).toEqual([{ text: '', match: false }])
  })
})
