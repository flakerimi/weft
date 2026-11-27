import { afterEach, describe, expect, it } from 'vitest'
import { registerIcon } from '../../src/basics/icons.ts'
import CommandPalette from '../../src/selection/CommandPalette.vue'
import CommandCases from '../fixtures/selection/CommandCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px, type } from '../forms/helpers.ts'
import { navigations } from '../keyboard.ts'
import { $, $$, active, key, overlays, reset, settle } from '../overlays/helpers.ts'

registerIcon('plus', '<path d="M5 12h14"></path>')

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
  localStorage.clear()
})
type Exposed = { log: string[]; term: string; open: boolean; closed: number }
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(CommandCases, props)
  unmount = m.unmount
  await settle()
  return m.exposed as unknown as Exposed
}
const root = () => overlays() ?? document
const input = () => $<HTMLInputElement>('.weft-command-palette__input', root())
const list = () => $('[role="listbox"]', root())
const options = () => $$('[role="option"]', root())
const text = (o: Element) => (o.querySelector('.weft-command-palette__label') ?? o.querySelector('.weft-listbox__text'))!.textContent!.trim()
const labels = () => options().map(text)
const named = (label: string) => options().find((o) => text(o) === label)!
const current = () => document.getElementById(input().getAttribute('aria-activedescendant') ?? '')
const headings = () => $$('[role="group"]', list()).map((g) => document.getElementById(g.getAttribute('aria-labelledby')!)!.textContent)
async function search(t: string) {
  input().focus()
  type(input(), t)
  await settle()
}

describe('CommandPalette', () => {
  it('is a search field that controls a listbox of grouped commands, the first one active', async () => {
    await cases()
    const field = input()
    expect(field.getAttribute('role')).toBe('combobox')
    expect(field.getAttribute('aria-expanded')).toBe('true')
    expect(field.getAttribute('aria-autocomplete')).toBe('list')
    expect(field.getAttribute('aria-controls')).toBe(list().id)
    expect(field.id).toBe(`${list().id}-input`)
    expect(field.getAttribute('placeholder')).toBe('Type a command or search…')
    expect(headings()).toEqual(['Actions', 'People', 'Pinned'])
    expect(current()).toBe(named('New file'))
    expect(named('New file').getAttribute('aria-selected')).toBe('true') // selection follows the highlight
    expect(named('New window').getAttribute('aria-selected')).toBe('false')
    expect($$('.weft-kbd', named('New file'))).toHaveLength(2)
    expect($('.weft-command-palette__suffix', named('New window')).textContent).toBe('Opens a window')
    expect($('.weft-spinner', named('Sync'))).not.toBeNull()
    expect(named('Sync').getAttribute('aria-disabled')).toBe('true')
    expect($('.weft-command-palette__chevron', named('Change theme'))).not.toBeNull()
  })

  it('shows a focus ring under the search field while it has focus, and marks the active command', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      const ring = ruleFor('.weft-command-palette__search:focus-within')!.style
      expect(ring.getPropertyValue('box-shadow')).toContain('var(--weft-color-focus, Highlight)')
      // happy-dom neither matches nor computes :focus-within, so this ring is checked by its rule only.
      expect(ruleFor('.weft-listbox__option[data-active]')).toBeDefined()
    } finally {
      unstyle()
    }
  })

  it('matches fuzzily, marks the matched letters, and orders groups by their best match', async () => {
    const e = await cases()
    await search('grc hpr')
    expect(e.term).toBe('grc hpr')
    expect(labels()).toEqual(['Grace Hopper', 'Always here'])
    expect($$('mark', named('Grace Hopper')).map((m) => m.textContent).join('')).toBe('GrcHpr')
    await search('navy') // a keyword
    expect(labels()[0]).toBe('Grace Hopper')
    await search('math') // the description
    expect(labels()[0]).toBe('Ada Lovelace')
    await search('nw')
    expect(labels().slice(0, 2)).toEqual(['New window', 'New file'])
    expect(current()).toBe(named('New window'))
    await search('ada')
    expect(headings()).toEqual(['People', 'Pinned'])
  })

  it('keeps the groups in their order with preserve-group-order', async () => {
    await cases({ preserveGroupOrder: true })
    await search('a')
    expect(headings()).toEqual(['Actions', 'People', 'Pinned'])
  })

  it('shows a group with ignore-filter whatever the search, through its postFilter', async () => {
    await cases()
    await search('zzzz')
    expect(labels()).toEqual(['Always here'])
  })

  it('limits each group to `limit` results while searching', async () => {
    await cases({ many: 30, limit: 5 })
    expect(options().filter((o) => text(o).startsWith('Person')).length).toBe(27)
    await search('person')
    expect(options().filter((o) => text(o).startsWith('Person')).length).toBe(5)
  })

  it('moves with the arrows across groups past disabled and loading commands, and Enter runs one', async () => {
    const e = await cases()
    input().focus()
    key('ArrowDown')
    key('ArrowDown')
    await settle()
    expect(current()).toBe(named('Change theme')) // past Save and Sync
    expect(key('Home').defaultPrevented).toBe(false) // the caret's
    key('ArrowUp')
    key('ArrowUp')
    key('ArrowDown')
    key('ArrowDown')
    key('ArrowDown')
    key('ArrowDown')
    await settle()
    expect(current()).toBe(named('Ada Lovelace'))
    const enter = key('Enter')
    await settle()
    expect(enter.defaultPrevented).toBe(true)
    expect(e.log).toEqual(['p0', 'select:p0'])
    named('Sync').click()
    named('Save').click()
    await settle()
    expect(e.log).toEqual(['p0', 'select:p0'])
    named('Grace Hopper').click()
    await settle()
    expect(e.log).toEqual(['p0', 'select:p0', 'p1', 'select:p1'])
  })

  it('opens a command with children as a page of its own, with a back button and Backspace back', async () => {
    const e = await cases()
    await search('theme')
    key('Enter')
    await settle()
    expect(e.log).toEqual([])
    expect(e.term).toBe('')
    expect(labels()).toEqual(['Light', 'Dark'])
    expect(headings()).toEqual(['Change theme'])
    expect(input().getAttribute('placeholder')).toBe('Pick a theme…')
    expect(current()).toBe(named('Light'))
    const back = $<HTMLButtonElement>('.weft-command-palette__back', root())
    expect(back.getAttribute('aria-label')).toBe('Back')
    key('ArrowDown')
    key('Enter')
    await settle()
    expect(e.log).toEqual(['dark', 'select:dark'])
    key('Backspace')
    await settle()
    expect(headings()).toEqual(['Actions', 'People', 'Pinned'])
    expect($('.weft-command-palette__back', root())).toBeNull()
    expect(active()).toBe(input())
    await search('theme')
    key('Enter')
    await settle()
    $<HTMLButtonElement>('.weft-command-palette__back', root()).click()
    await settle()
    expect(headings()).toContain('Actions')
    expect(active()).toBe(input())
  })

  it('follows a command with `to`', async () => {
    await cases()
    const followed = navigations(document.body)
    try {
      await search('docs')
      key('Enter')
      await settle()
      expect(followed.urls).toEqual([new URL('/docs', location.href).href])
    } finally {
      followed.restore()
    }
  })

  it('lists recent commands first while the search is empty, kept under recent-key', async () => {
    const e = await cases({ recent: 2, recentKey: 'test' })
    expect(headings()[0]).toBe('Actions')
    for (const t of ['grace', 'ada', 'new win']) {
      await search(t)
      key('Enter')
      await settle()
    }
    await search('')
    expect(headings()[0]).toBe('Recent')
    const recent = $$('[role="option"]', $$('[role="group"]', list())[0])
    expect(recent.map(text)).toEqual(['New window', 'Ada Lovelace'])
    expect(JSON.parse(localStorage.getItem('weft:recent:test')!)).toEqual(['new-window', 'p0'])
    expect(e.log.filter((l) => l.startsWith('select'))).toHaveLength(3)
    unmount!()
    reset()
    await cases({ recent: 2, recentKey: 'test' })
    expect(headings()[0]).toBe('Recent')
    await search('a')
    expect(headings()).not.toContain('Recent')
  })

  it('says "No results", or shows the empty slot with the search, when nothing matches', async () => {
    await cases({ bare: true })
    await search('qqqq')
    expect(options()).toHaveLength(0)
    expect($('[role="status"]', root()).textContent!.trim()).toBe('No results')
    expect(input().hasAttribute('aria-activedescendant')).toBe(false)
    unmount!()
    reset()
    await cases({ emptySlot: true })
    await search('qqqq')
    expect($('.none', root()).textContent).toBe('Nothing for qqqq')
  })

  it('is busy while loading, and offers a close button with close', async () => {
    const e = await cases({ loading: true, close: true })
    expect(list().getAttribute('aria-busy')).toBe('true')
    expect($('.weft-command-palette__search .weft-spinner')).not.toBeNull()
    const close = $<HTMLButtonElement>('.weft-command-palette__search .weft-close-button')
    expect(close.getAttribute('aria-label')).toBe('Close')
    close.click()
    await settle()
    expect(e.closed).toBe(1)
  })

  it('on touch draws its commands at least --weft-hit-min tall, long lists too', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch', many: 300 })
      expect($('.weft-listbox', root()).dataset.touch).toBe('true')
      expect(list().dataset.virtual).toBe('true')
      const option = named('Ada Lovelace')
      expect(px(getComputedStyle(option).minBlockSize)).toBeGreaterThanOrEqual(44)
    } finally {
      unstyle()
    }
  })

  it('gives its attributes to its root, but the name to the search field, and draws no empty style', async () => {
    const m = mount(CommandPalette, { groups: [], class: 'mine', id: 'palette', 'data-x': '1', 'aria-label': 'Jump to' })
    unmount = m.unmount
    await settle()
    const el = $('.weft-command-palette')
    expect(el.id).toBe('palette')
    expect(el.classList.contains('mine')).toBe(true)
    expect(el.dataset.x).toBe('1')
    expect(el.hasAttribute('style')).toBe(false)
    expect(el.hasAttribute('aria-label')).toBe(false)
    expect(input().getAttribute('aria-label')).toBe('Jump to')
    unmount!()
    reset()
    const n = mount(CommandPalette, { groups: [], 'aria-labelledby': 'heading', placeholder: 'Find…' })
    unmount = n.unmount
    await settle()
    expect(input().getAttribute('aria-labelledby')).toBe('heading')
    expect(input().hasAttribute('aria-label')).toBe(false)
  })

  it('reads labels, descriptions and keys at label-key, description-key and value-key', async () => {
    const log: string[] = []
    const m = mount(CommandPalette, {
      groups: [{ id: 'g', label: 'Places', items: [{ name: 'Lisbon', info: 'Portugal', code: 'lis' }, { name: 'Oslo', info: 'Norway', code: 'osl' }] }],
      labelKey: 'name',
      descriptionKey: 'info',
      valueKey: 'code',
      recent: true,
      recentKey: 'keys',
      // Root props that are functions are read as getters: the handler is what this one returns.
      onSelect: () => (item: { code: string }) => log.push(item.code),
    })
    unmount = m.unmount
    await settle()
    expect(labels()).toEqual(['Lisbon', 'Oslo'])
    expect($('.weft-listbox__description', named('Oslo')).textContent).toBe('Norway')
    await search('norw')
    expect(labels()).toEqual(['Oslo'])
    key('Enter')
    await settle()
    expect(log).toEqual(['osl'])
    expect(JSON.parse(localStorage.getItem('weft:recent:keys')!)).toEqual(['osl'])
  })

  it('runs nothing while disabled', async () => {
    const e = await cases({ disabled: true })
    expect(input().disabled).toBe(true)
    expect(list().getAttribute('aria-disabled')).toBe('true')
    named('Ada Lovelace').click()
    key('Enter', {}, input())
    await settle()
    expect(e.log).toEqual([])
  })

  describe('in a Dialog (a Cmd+K launcher)', () => {
    it('takes focus when the dialog opens, runs a command, and the dialog closes back to its opener', async () => {
      const e = await cases({ inDialog: true })
      const opener = $<HTMLButtonElement>('.opener')
      opener.focus()
      opener.click()
      await settle()
      expect(e.open).toBe(true)
      expect(active()).toBe(input())
      expect($('[role="dialog"]').contains(input())).toBe(true)
      type(input(), 'alan')
      await settle()
      expect(current()).toBe(named('Alan Turing'))
      key('Enter')
      await settle()
      expect(e.log).toEqual(['p2', 'select:p2'])
      expect(e.open).toBe(false)
      expect(active()).toBe(opener)
    })

    it('closes on Escape, focus back on the opener', async () => {
      const e = await cases({ inDialog: true })
      const opener = $<HTMLButtonElement>('.opener')
      opener.focus()
      opener.click()
      await settle()
      type(input(), 'ad')
      await settle()
      key('Escape')
      await settle()
      expect(e.open).toBe(false)
      expect(active()).toBe(opener)
    })
  })
})
