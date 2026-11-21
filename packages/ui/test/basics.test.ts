import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { registerIcon } from '../src/basics/icons.ts'
import Badge from '../src/basics/Badge.vue'
import Kbd from '../src/basics/Kbd.vue'
import Progress from '../src/basics/Progress.vue'
import Skeleton from '../src/basics/Skeleton.vue'
import Spinner from '../src/basics/Spinner.vue'
import BasicsCases from './fixtures/basics/BasicsCases.vue'
import { mount } from './mount.ts'
import { loadStyles } from './styles.ts'

registerIcon('check', '<path d="M20 6 9 17l-5-5"></path>')
registerIcon('x', '<path d="M18 6 6 18"></path>')

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
})
function root(component: object, props: Record<string, unknown> = {}) {
  const m = mount(component as never, props)
  mounted.push(m)
  return m.el.firstElementChild as HTMLElement
}
const cases = (c: string) => root(BasicsCases, { case: c })

describe('Badge', () => {
  it('is a solid primary md label by default', () => {
    const b = root(Badge, { label: 'New' })
    expect(b.tagName).toBe('SPAN')
    expect(b.className).toBe('weft-badge')
    expect(b.dataset).toMatchObject({ variant: 'solid', tone: 'primary', size: 'md' })
    expect(b.textContent).toBe('New')
  })

  it('takes variant, tone and size', () => {
    const b = root(Badge, { label: 'Failed', variant: 'soft', tone: 'danger', size: 'sm' })
    expect(b.dataset).toMatchObject({ variant: 'soft', tone: 'danger', size: 'sm' })
  })

  it('draws decorative leading and trailing icons', () => {
    const b = root(Badge, { label: 'Paid', icon: 'check', trailingIcon: 'x' })
    const icons = b.querySelectorAll('svg.weft-icon')
    expect(icons).toHaveLength(2)
    expect(icons[0]!.innerHTML).toContain('M20 6')
    expect(icons[1]!.innerHTML).toContain('M18 6')
    expect(b.firstElementChild).toBe(icons[0])
    expect(b.lastElementChild).toBe(icons[1])
    for (const i of icons) expect(i.getAttribute('aria-hidden')).toBe('true')
  })

  it('is square with only an icon, and puts its slot in place of the label', () => {
    expect(root(Badge, { icon: 'check' }).dataset.square).toBe('true')
    expect(root(Badge, { label: 'x' }).hasAttribute('data-square')).toBe(false)
    const b = cases('badge-slot')
    expect(b.querySelector('.mine')?.textContent).toBe('New')
    expect(b.textContent).toBe('New')
  })
})

describe('Kbd', () => {
  it('is a kbd element showing its value', () => {
    const k = root(Kbd, { value: 'K' })
    expect(k.tagName).toBe('KBD')
    expect(k.className).toBe('weft-kbd')
    expect(k.dataset.size).toBe('md')
    expect(k.textContent).toBe('K')
  })

  it('takes a size, and its slot in place of the value', () => {
    expect(root(Kbd, { value: 'K', size: 'sm' }).dataset.size).toBe('sm')
    expect(cases('kbd-slot').querySelector('.mine')?.textContent).toBe('F1')
  })

  it('draws any other value as given, even one named like an object property', () => {
    expect(root(Kbd, { value: 'constructor' }).textContent).toBe('constructor')
    expect(root(Kbd, { value: 'Esc' }).textContent).toBe('Esc')
  })

  it.each([
    ['command', '⌘', 'Command'],
    ['Shift', '⇧', 'Shift'],
    ['option', '⌥', 'Option'],
    ['enter', '↵', 'Enter'],
    ['arrowup', '↑', 'Up arrow'],
  ])('draws %s as its symbol, named for assistive technology', (value, symbol, name) => {
    const k = root(Kbd, { value })
    const shown = k.querySelector('[aria-hidden="true"]')!
    expect(shown.textContent).toBe(symbol)
    expect(k.querySelector('.weft-visually-hidden')?.textContent).toBe(name)
  })
})

describe('Chip', () => {
  it('draws an indicator on the corner of what it wraps', () => {
    const c = cases('chip')
    expect(c.className).toBe('weft-chip')
    expect(c.querySelector('.anchor')?.textContent).toBe('bell')
    const dot = c.querySelector('.weft-chip__indicator') as HTMLElement
    expect(dot.textContent).toBe('3')
    expect(dot.dataset).toMatchObject({ tone: 'danger', size: 'lg', position: 'bottom-start', inset: 'true', text: 'true' })
  })

  it('defaults to a primary md dot at the top end', () => {
    const dot = cases('chip-dot').querySelector('.weft-chip__indicator') as HTMLElement
    expect(dot.textContent).toBe('')
    expect(dot.dataset).toMatchObject({ tone: 'primary', size: 'md', position: 'top-end' })
    expect(dot.hasAttribute('data-text')).toBe(false)
    expect(dot.hasAttribute('data-inset')).toBe(false)
  })

  it('names itself for assistive technology with a label, hiding the drawn indicator', () => {
    const c = cases('chip-label')
    expect(c.querySelector('.weft-chip__indicator')!.getAttribute('aria-hidden')).toBe('true')
    expect(c.querySelector('.weft-visually-hidden')?.textContent).toBe('3 unread messages')
  })

  it('hides the indicator with show false, keeping what it wraps', () => {
    const c = cases('chip-hidden')
    expect(c.querySelector('.weft-chip__indicator')).toBeNull()
    expect(c.querySelector('.anchor')).not.toBeNull()
  })
})

describe('Spinner', () => {
  it('is decorative without a label', () => {
    const s = root(Spinner)
    expect(s.className).toBe('weft-spinner')
    expect(s.getAttribute('aria-hidden')).toBe('true')
    expect(s.hasAttribute('role')).toBe(false)
    expect(s.hasAttribute('data-size')).toBe(false)
  })

  it('is a status announcing its label when given one', () => {
    const s = root(Spinner, { label: 'Loading invoices', size: 'lg' })
    expect(s.getAttribute('role')).toBe('status')
    expect(s.hasAttribute('aria-hidden')).toBe(false)
    expect(s.dataset.size).toBe('lg')
    expect(s.querySelector('.weft-visually-hidden')?.textContent).toBe('Loading invoices')
  })
})

describe('Progress', () => {
  it('is a determinate progressbar for a value', () => {
    const p = root(Progress, { modelValue: 40, label: 'Upload' })
    expect(p.getAttribute('role')).toBe('progressbar')
    expect(p.getAttribute('aria-label')).toBe('Upload')
    expect(p.getAttribute('aria-valuemin')).toBe('0')
    expect(p.getAttribute('aria-valuemax')).toBe('100')
    expect(p.getAttribute('aria-valuenow')).toBe('40')
    expect(p.dataset).toMatchObject({ state: 'determinate', tone: 'primary', size: 'md' })
    expect((p.querySelector('.weft-progress__bar') as HTMLElement).style.inlineSize).toBe('40%')
  })

  it('is indeterminate without a value: no aria-valuenow, no width', () => {
    const p = root(Progress, { label: 'Working' })
    expect(p.getAttribute('role')).toBe('progressbar')
    expect(p.hasAttribute('aria-valuenow')).toBe(false)
    expect(p.dataset.state).toBe('indeterminate')
    expect((p.querySelector('.weft-progress__bar') as HTMLElement).style.inlineSize).toBe('')
    expect(root(Progress, { modelValue: null }).dataset.state).toBe('indeterminate')
  })

  it('measures against max, clamped to the range', () => {
    const p = root(Progress, { modelValue: 3, max: 4 })
    expect(p.getAttribute('aria-valuemax')).toBe('4')
    expect(p.getAttribute('aria-valuenow')).toBe('3')
    expect((p.querySelector('.weft-progress__bar') as HTMLElement).style.inlineSize).toBe('75%')
    const over = root(Progress, { modelValue: 140 })
    expect(over.getAttribute('aria-valuenow')).toBe('100')
    expect((over.querySelector('.weft-progress__bar') as HTMLElement).style.inlineSize).toBe('100%')
    expect(root(Progress, { modelValue: -5 }).getAttribute('aria-valuenow')).toBe('0')
  })

  it('takes tone and size, and shows the percentage with status', () => {
    const p = root(Progress, { modelValue: 2, max: 3, tone: 'success', size: 'lg', status: true })
    expect(p.dataset).toMatchObject({ tone: 'success', size: 'lg' })
    expect(p.querySelector('.weft-progress__status')?.textContent).toBe('67%')
    expect(root(Progress, { modelValue: 2 }).querySelector('.weft-progress__status')).toBeNull()
    expect(root(Progress, { status: true }).querySelector('.weft-progress__status')).toBeNull()
  })

  it('follows its value live', async () => {
    const value = ref<number | null>(10)
    const p = root(Progress, { modelValue: () => value.value })
    value.value = 90
    await nextTick()
    expect(p.getAttribute('aria-valuenow')).toBe('90')
    value.value = null
    await nextTick()
    expect(p.hasAttribute('aria-valuenow')).toBe(false)
    expect(p.dataset.state).toBe('indeterminate')
  })
})

describe('Skeleton', () => {
  it('is a decorative placeholder sized by width and height', () => {
    const s = root(Skeleton, { width: '12rem', height: '1.5rem' })
    expect(s.className).toBe('weft-skeleton')
    expect(s.getAttribute('aria-hidden')).toBe('true')
    expect(s.style.inlineSize).toBe('12rem')
    expect(s.style.blockSize).toBe('1.5rem')
    expect(s.hasAttribute('data-circle')).toBe(false)
    expect(root(Skeleton, { circle: true }).dataset.circle).toBe('true')
  })
})

describe('motion', () => {
  let unload: () => void
  const settings = () => (window as unknown as { happyDOM: { settings: { device: { prefersReducedMotion: string } } } }).happyDOM.settings.device
  beforeEach(() => { unload = loadStyles() })
  afterEach(() => {
    settings().prefersReducedMotion = 'no-preference'
    unload()
  })
  const animation = (el: Element) => getComputedStyle(el).getPropertyValue('animation-name') || getComputedStyle(el).getPropertyValue('animation')

  it('pulses a Skeleton and sweeps an indeterminate Progress', () => {
    expect(animation(root(Skeleton))).toContain('weft-pulse')
    expect(animation(root(Progress).querySelector('.weft-progress__bar')!)).toContain('weft-sweep')
  })

  it('stills both under prefers-reduced-motion', () => {
    settings().prefersReducedMotion = 'reduce'
    expect(matchMedia('(prefers-reduced-motion: reduce)').matches).toBe(true)
    expect(animation(root(Skeleton))).toMatch(/^none\b/)
    expect(animation(root(Progress).querySelector('.weft-progress__bar')!)).toMatch(/^none\b/)
  })
})
