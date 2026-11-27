import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { registerIcon } from '../src/basics/icons.ts'
import LandingCases from './fixtures/LandingCases.vue'
import { mount } from './mount.ts'
import { installResizeObserver } from './resize-observer.ts'
import { loadStyles } from './styles.ts'

registerIcon('zap', '<path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"></path>')
registerIcon('shield', '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>')
registerIcon('check', '<path d="M20 6L9 17l-5-5"></path>')

describe('Hero', () => {
  let ro: ReturnType<typeof installResizeObserver>
  beforeEach(() => {
    ro = installResizeObserver()
  })
  afterEach(() => ro.restore())

  it('renders an h1 by default with headline, description and solid/outline action buttons', () => {
    const m = mount(LandingCases, { case: 'hero-default' })
    const hero = m.el.querySelector('section.weft-hero') as HTMLElement
    expect(hero.dataset).toMatchObject({ orientation: 'vertical', align: 'center' })
    expect(hero.querySelector('.weft-hero__headline')?.textContent).toBe('Pre-alpha')
    expect(hero.querySelector('h1.weft-hero__title')?.textContent).toBe('One UI system')
    expect(hero.querySelector('.weft-hero__description')?.textContent).toBe('Go and Vue in one binary.')
    const btns = hero.querySelectorAll<HTMLElement>('.weft-hero__actions .weft-button')
    expect(btns).toHaveLength(2)
    expect(btns[0]!.dataset).toMatchObject({ tone: 'primary', variant: 'solid', size: 'lg' })
    expect(btns[1]!.dataset).toMatchObject({ tone: 'neutral', variant: 'outline', size: 'lg' })
    expect(btns[1]!.getAttribute('href')).toBe('/docs')
    btns[0]!.click()
    expect(m.exposed.clicks).toEqual(['install'])
    m.unmount()
  })

  it('demotes its heading to h2 inside an inset WeftProvider (the gallery preview stage)', () => {
    const m = mount(LandingCases, { case: 'hero-inset' })
    const hero = m.el.querySelector('.weft-hero')!
    expect(hero.querySelector('h1')).toBeNull()
    expect(hero.querySelector('h2.weft-hero__title')?.textContent).toBe('Inset hero')
    m.unmount()
  })

  it('keeps orientation="vertical" by default even when the default slot is filled', () => {
    const m = mount(LandingCases, { case: 'hero-slot-vertical' })
    const hero = m.el.querySelector('section.weft-hero') as HTMLElement
    ro.report(hero, 1200)
    expect(hero.dataset.orientation).toBe('vertical')
    expect(hero.querySelector('.weft-hero__media .visual')?.textContent).toBe('Preview frame')
    m.unmount()
  })

  it('keeps data-orientation as the prop while updating data-width with useWidth', async () => {
    const m = mount(LandingCases, { case: 'hero-horizontal' })
    const hero = m.el.querySelector('section.weft-hero') as HTMLElement
    expect(hero.querySelector('h3.weft-hero__title')?.textContent).toBe('Side by side')
    expect(hero.querySelector('.weft-hero__background')?.getAttribute('aria-hidden')).toBe('true')
    expect(hero.querySelector('.custom-headline')?.textContent).toBe('New')
    expect(hero.querySelector('.weft-hero__caption .cap')?.textContent).toBe('v0.1.0')
    expect(hero.dataset.align).toBe('start')
    expect(hero.dataset.orientation).toBe('horizontal')

    ro.report(hero, 1100)
    await nextTick()
    expect(hero.dataset).toMatchObject({ width: 'wide', orientation: 'horizontal' })

    ro.report(hero, 800)
    await nextTick()
    expect(hero.dataset).toMatchObject({ width: 'regular', orientation: 'horizontal' })

    ro.report(hero, 500)
    await nextTick()
    expect(hero.dataset).toMatchObject({ width: 'narrow', orientation: 'horizontal' })
    m.unmount()
  })

  it('renders only one h1 when all six landing blocks sit on the same page', () => {
    const m = mount(LandingCases, { case: 'all-blocks-page' })
    const h1s = m.el.querySelectorAll('h1')
    expect(h1s).toHaveLength(1)
    expect(h1s[0]!.textContent).toBe('Main page heading')
    m.unmount()
  })
})

describe('Cta', () => {
  let ro: ReturnType<typeof installResizeObserver>
  beforeEach(() => {
    ro = installResizeObserver()
  })
  afterEach(() => ro.restore())

  it('renders an h2 in a card by default with actions', () => {
    const m = mount(LandingCases, { case: 'cta-default' })
    const cta = m.el.querySelector('section.weft-cta') as HTMLElement
    expect(cta.dataset).toMatchObject({ variant: 'card', tone: 'primary', orientation: 'vertical', align: 'center' })
    expect(cta.querySelector('h2.weft-cta__title')?.textContent).toBe('Ship one binary')
    const btns = cta.querySelectorAll<HTMLElement>('.weft-cta__actions .weft-button')
    expect(btns).toHaveLength(2)
    expect(btns[0]!.dataset).toMatchObject({ tone: 'primary', variant: 'solid', size: 'lg' })
    expect(btns[1]!.dataset).toMatchObject({ tone: 'neutral', variant: 'outline', size: 'lg' })
    btns[0]!.click()
    expect(m.exposed.clicks).toEqual(['create'])
    m.unmount()
  })

  it('uses neutral buttons on a solid variant for contrast and keeps data-orientation as the prop', async () => {
    const m = mount(LandingCases, { case: 'cta-solid-horizontal' })
    const cta = m.el.querySelector('section.weft-cta') as HTMLElement
    expect(cta.querySelector('h3.weft-cta__title')?.textContent).toBe('Solid band')
    const btns = cta.querySelectorAll<HTMLElement>('.weft-button')
    expect(btns[0]!.dataset).toMatchObject({ tone: 'neutral', variant: 'solid' })
    expect(btns[1]!.dataset).toMatchObject({ tone: 'neutral', variant: 'outline' })

    ro.report(cta, 800)
    await nextTick()
    expect(cta.dataset).toMatchObject({ width: 'regular', orientation: 'horizontal', align: 'start' })

    ro.report(cta, 500)
    await nextTick()
    expect(cta.dataset).toMatchObject({ width: 'narrow', orientation: 'horizontal' })
    m.unmount()
  })

  it('keeps actions in the copy column when an aside slot is provided on a horizontal Cta', async () => {
    const m = mount(LandingCases, { case: 'cta-aside' })
    const cta = m.el.querySelector('section.weft-cta') as HTMLElement
    ro.report(cta, 1100)
    await nextTick()
    expect(cta.querySelector('.weft-cta__copy .weft-cta__actions')).not.toBeNull()
    expect(cta.querySelector('.weft-cta__aside .loom')?.textContent).toBe('Loom')
    m.unmount()
  })
})

describe('FeatureGrid', () => {
  let ro: ReturnType<typeof installResizeObserver>
  beforeEach(() => {
    ro = installResizeObserver()
  })
  afterEach(() => ro.restore())

  it('renders an h2 section heading, h3 item headings, icon discs, standalone links, named slots and responsive Grid columns', async () => {
    const m = mount(LandingCases, { case: 'features-default' })
    const fg = m.el.querySelector('section.weft-feature-grid') as HTMLElement
    expect(fg.dataset).toMatchObject({ variant: 'card', tone: 'primary', align: 'start' })
    expect(fg.querySelector('h2.weft-feature-grid__title')?.textContent).toBe('Built together')
    expect(fg.querySelector('.weft-feature-grid__actions .hdr-act')?.textContent).toBe('All')

    const items = fg.querySelectorAll<HTMLElement>('.weft-feature-grid__item')
    expect(items).toHaveLength(3)
    expect(items[0]!.querySelector('h3.weft-feature-grid__item-title')?.textContent).toBe('Fast')
    expect(items[0]!.querySelector('.weft-feature-grid__icon svg.weft-icon')).not.toBeNull()
    const standaloneLink = items[0]!.querySelector<HTMLAnchorElement>('a.weft-feature-grid__item-link')!
    expect(standaloneLink.getAttribute('href')).toBe('/fast')
    expect(standaloneLink.dataset.standalone).toBe('true')
    expect(standaloneLink.textContent).toBe('Learn more')

    // Second item has `to` without `linkLabel`: the title itself links there as a standalone Link.
    const titleLink = items[1]!.querySelector<HTMLAnchorElement>('h3 a.weft-link')!
    expect(titleLink.getAttribute('href')).toBe('/typed')
    expect(titleLink.dataset.standalone).toBe('true')
    expect(titleLink.textContent).toBe('Typed')

    // Third item uses its named `slot: 'special'`.
    expect(items[2]!.querySelector('.special-slot')?.textContent).toBe('Custom!')

    const grid = fg.querySelector<HTMLElement>('.weft-grid')!
    expect(grid.style.getPropertyValue('--weft-grid-cols-narrow')).toBe('1')
    expect(grid.style.getPropertyValue('--weft-grid-cols-regular')).toBe('2')
    expect(grid.style.getPropertyValue('--weft-grid-cols-wide')).toBe('3')
    ro.report(grid, 500)
    await nextTick()
    expect(grid.style.getPropertyValue('--weft-grid-cols')).toBe('1')
    ro.report(grid, 800)
    await nextTick()
    expect(grid.style.getPropertyValue('--weft-grid-cols')).toBe('2')
    ro.report(grid, 1100)
    await nextTick()
    expect(grid.style.getPropertyValue('--weft-grid-cols')).toBe('3')
    m.unmount()
  })

  it('supports plain variant, center alignment and deeper heading levels', () => {
    const m = mount(LandingCases, { case: 'features-plain' })
    const fg = m.el.querySelector('section.weft-feature-grid') as HTMLElement
    expect(fg.dataset).toMatchObject({ variant: 'plain', align: 'center' })
    expect(fg.querySelector('h3.weft-feature-grid__title')?.textContent).toBe('Plain features')
    expect(fg.querySelector('h4.weft-feature-grid__item-title')?.textContent).toBe('Fast')
    m.unmount()
  })
})

describe('Testimonials', () => {
  it('renders semantic figure, blockquote and figcaption with decorative Avatar initials or image and optional Link', () => {
    const m = mount(LandingCases, { case: 'testimonials-default' })
    const sec = m.el.querySelector('section.weft-testimonials') as HTMLElement
    expect(sec.querySelector('h2.weft-testimonials__title')?.textContent).toBe('Loved by engineers')
    const figures = sec.querySelectorAll<HTMLElement>('figure.weft-testimonials__item')
    expect(figures).toHaveLength(2)

    expect(figures[0]!.querySelector('blockquote p')?.textContent).toBe('One binary changed our deploys.')
    const avatar0 = figures[0]!.querySelector<HTMLElement>('figcaption .weft-avatar')!
    expect(avatar0.querySelector('.weft-avatar__initials')?.textContent).toBe('AQ')
    // Decorative beside the visible author name: no role="img" or aria-label repeating the name.
    expect(avatar0.getAttribute('role')).toBeNull()
    expect(avatar0.getAttribute('aria-label')).toBeNull()
    expect(figures[0]!.querySelector('figcaption a.weft-link')?.getAttribute('href')).toBe('/avery')
    expect(figures[0]!.querySelector('figcaption .weft-testimonials__role')?.textContent).toBe('CTO')

    const img1 = figures[1]!.querySelector<HTMLImageElement>('figcaption img.weft-avatar__image')!
    expect(img1.getAttribute('src')).toBe('https://example.com/morgan.png')
    expect(img1.getAttribute('alt')).toBe('')
    m.unmount()
  })
})

describe('Pricing', () => {
  let ro: ReturnType<typeof installResizeObserver>
  beforeEach(() => {
    ro = installResizeObserver()
  })
  afterEach(() => ro.restore())

  it('renders plans at level + 1, switches billing cycles via ToggleGroup, marks the pressed cycle badge, keeps the active cycle on repeat click and adapts columns', async () => {
    const m = mount(LandingCases, { case: 'pricing-cycles' })
    const sec = m.el.querySelector('section.weft-pricing') as HTMLElement
    expect(sec.querySelector('h2.weft-pricing__title')?.textContent).toBe('Pick a plan')
    expect(sec.querySelector('.weft-pricing__footer .price-foot')?.textContent).toBe('All prices in USD')

    const toggleGroup = sec.querySelector<HTMLElement>('.weft-toggle-group')!
    expect(toggleGroup.getAttribute('aria-label')).toBe('Billing period')
    const cycleBtns = toggleGroup.querySelectorAll<HTMLButtonElement>('.weft-toggle-group__item')
    expect(cycleBtns).toHaveLength(2)
    expect(cycleBtns[0]!.getAttribute('aria-pressed')).toBe('true')
    expect(cycleBtns[1]!.getAttribute('aria-pressed')).toBe('false')
    const badge = cycleBtns[1]!.querySelector<HTMLElement>('.weft-badge')!
    expect(badge.textContent).toBe('Save 20%')
    expect(badge.dataset.pressed).toBeUndefined()

    const cards = sec.querySelectorAll<HTMLElement>('.weft-pricing__plan')
    expect(cards).toHaveLength(2)
    expect(cards[0]!.querySelector('h3.weft-pricing__plan-title')?.textContent).toBe('Starter')
    expect(cards[0]!.querySelector('.weft-pricing__price')?.textContent).toBe('$0')
    expect(cards[0]!.querySelector<HTMLElement>('.weft-pricing__button')?.dataset.variant).toBe('outline')

    expect(cards[1]!.dataset.highlight).toBe('true')
    expect(cards[1]!.querySelector('.weft-pricing__plan-badge')?.textContent).toBe('Popular')
    expect(cards[1]!.querySelector('.weft-pricing__price')?.textContent).toBe('$29')
    expect(cards[1]!.querySelector('.weft-pricing__note')).toBeNull()
    expect(cards[1]!.querySelector<HTMLElement>('.weft-pricing__button')?.dataset.variant).toBe('solid')

    // Clicking the already-active cycle keeps it selected (never deselects to null).
    cycleBtns[0]!.click()
    await nextTick()
    await nextTick()
    expect(m.exposed.cycle).toBe('monthly')
    expect(cycleBtns[0]!.getAttribute('aria-pressed')).toBe('true')

    // Switching to Yearly updates per-cycle price and billingNote and sets data-pressed on the cycle badge.
    cycleBtns[1]!.click()
    await nextTick()
    expect(m.exposed.cycle).toBe('yearly')
    expect(badge.dataset.pressed).toBe('true')
    expect(cards[0]!.querySelector('.weft-pricing__price')?.textContent).toBe('$0')
    expect(cards[1]!.querySelector('.weft-pricing__price')?.textContent).toBe('$24')
    expect(cards[1]!.querySelector('.weft-pricing__note')?.textContent).toBe('Billed $288 yearly')

    // Keyboard roving focus on ToggleGroup (ArrowRight wraps from index 1 to 0).
    cycleBtns[1]!.focus()
    cycleBtns[1]!.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }))
    expect(document.activeElement).toBe(cycleBtns[0])

    // Responsive Grid columns for 2 plans: 1 narrow, 2 regular, 2 wide.
    const grid = sec.querySelector<HTMLElement>('.weft-grid')!
    expect(grid.style.getPropertyValue('--weft-grid-cols-narrow')).toBe('1')
    expect(grid.style.getPropertyValue('--weft-grid-cols-regular')).toBe('2')
    expect(grid.style.getPropertyValue('--weft-grid-cols-wide')).toBe('2')
    ro.report(grid, 500)
    await nextTick()
    expect(grid.style.getPropertyValue('--weft-grid-cols')).toBe('1')
    ro.report(grid, 1100)
    await nextTick()
    expect(grid.style.getPropertyValue('--weft-grid-cols')).toBe('2')
    m.unmount()
  })

  it('leaves the price area empty and warns once in development (and never in production) when a per-cycle map lacks the selected cycle or cycles is empty', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const m = mount(LandingCases, { case: 'pricing-missing' })
    const card = m.el.querySelector<HTMLElement>('.weft-pricing__plan')!
    // Initial cycle is 'monthly', which AnnualOnly's map `{ yearly: '$99' }` lacks.
    expect(card.querySelector('.weft-pricing__price-row')).toBeNull()
    expect(warn).toHaveBeenCalledTimes(1)
    expect(warn.mock.calls[0]?.[0]).toBe(
      'Pricing: plan "AnnualOnly" has no price for cycle "monthly"; add a "monthly" entry to its price map in your page, or give it one plain string.',
    )

    // Switching to yearly shows '$99 /yr'; switching back to monthly does not warn a second time.
    const cycleBtns = m.el.querySelectorAll<HTMLButtonElement>('.weft-toggle-group__item')
    cycleBtns[1]!.click()
    await nextTick()
    expect(card.querySelector('.weft-pricing__price')?.textContent).toBe('$99')
    expect(card.querySelector('.weft-pricing__period')?.textContent).toBe('/yr')

    cycleBtns[0]!.click()
    await nextTick()
    expect(card.querySelector('.weft-pricing__price-row')).toBeNull()
    expect(warn).toHaveBeenCalledTimes(1)

    // A price map with no cycles at all warns once naming the plan.
    const mNoCycles = mount(LandingCases, { case: 'pricing-no-cycles-map' })
    expect(mNoCycles.el.querySelector('.weft-pricing__price-row')).toBeNull()
    expect(warn).toHaveBeenCalledTimes(2)
    expect(warn.mock.calls[1]?.[0]).toBe(
      'Pricing: plan "MapWithoutCycles" gives price as a per-cycle map, but cycles is empty; pass cycles to Pricing in your page, or give price one plain string.',
    )
    mNoCycles.unmount()

    // In production (`import.meta.env.DEV === false`), missing a cycle never warns.
    vi.stubEnv('DEV', false)
    try {
      const mProd = mount(LandingCases, { case: 'pricing-missing-prod' })
      expect(mProd.el.querySelector('.weft-pricing__price-row')).toBeNull()
      expect(warn).toHaveBeenCalledTimes(2)
      mProd.unmount()
    } finally {
      vi.unstubAllEnvs()
    }
    warn.mockRestore()
    m.unmount()
  })
})

describe('Faq', () => {
  let ro: ReturnType<typeof installResizeObserver>
  beforeEach(() => {
    ro = installResizeObserver()
  })
  afterEach(() => ro.restore())

  it('renders an h2 section heading, h3 Accordion triggers, until-found panels, v-model and named item slots', async () => {
    const m = mount(LandingCases, { case: 'faq-default' })
    const faq = m.el.querySelector('section.weft-faq') as HTMLElement
    expect(faq.dataset).toMatchObject({ orientation: 'vertical', align: 'start' })
    expect(faq.querySelector('h2.weft-faq__title')?.textContent).toBe('Frequently asked')
    expect(faq.querySelector('.weft-faq__aside .faq-contact')?.textContent).toBe('Email us')

    const headers = faq.querySelectorAll('h3.weft-accordion__header')
    expect(headers).toHaveLength(2)
    const triggers = faq.querySelectorAll<HTMLButtonElement>('.weft-accordion__trigger')
    const panels = faq.querySelectorAll<HTMLElement>('.weft-accordion__content')
    expect(panels[0]!.getAttribute('hidden')).toBe('until-found')
    expect(panels[1]!.querySelector('.db-slot')?.textContent).toBe('SQLite today.')

    triggers[0]!.click()
    await nextTick()
    expect(m.exposed.openFaq).toBe('js')
    expect(triggers[0]!.getAttribute('aria-expanded')).toBe('true')
    expect(panels[0]!.hasAttribute('hidden')).toBe(false)

    // Keyboard ArrowDown moves between question triggers.
    triggers[0]!.focus()
    triggers[0]!.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    expect(document.activeElement).toBe(triggers[1])
    m.unmount()
  })

  it('adapts horizontal orientation with @container and useWidth and forwards level + 1 to Accordion', async () => {
    const m = mount(LandingCases, { case: 'faq-horizontal' })
    const faq = m.el.querySelector('section.weft-faq') as HTMLElement
    expect(faq.querySelector('h3.weft-faq__title')?.textContent).toBe('Split FAQ')
    expect(faq.querySelectorAll('h4.weft-accordion__header')).toHaveLength(2)
    expect(faq.querySelector('.weft-accordion__trigger')?.getAttribute('aria-expanded')).toBe('true')

    ro.report(faq, 1100)
    await nextTick()
    expect(faq.dataset).toMatchObject({ orientation: 'horizontal', width: 'wide' })

    ro.report(faq, 700)
    await nextTick()
    expect(faq.dataset).toMatchObject({ orientation: 'horizontal', width: 'regular' })
    m.unmount()
  })
})

describe('Landing CSS rules (no container-type collapse, overflow-wrap and contrast)', () => {
  let unload: () => void
  beforeEach(() => {
    unload = loadStyles()
  })
  afterEach(() => unload())

  it('sets overflow-wrap: anywhere without container-type: inline-size on the landing blocks and Grid', () => {
    const m = mount(LandingCases, { case: 'all-blocks-page' })
    for (const sel of ['.weft-hero', '.weft-feature-grid', '.weft-pricing', '.weft-testimonials', '.weft-faq', '.weft-cta']) {
      const el = m.el.querySelector<HTMLElement>(sel)!
      const style = getComputedStyle(el)
      expect(style.containerType, sel).not.toBe('inline-size')
      expect(style.overflowWrap, sel).toBe('anywhere')
    }
    const grid = m.el.querySelector<HTMLElement>('.weft-grid')!
    expect(getComputedStyle(grid).containerType).not.toBe('inline-size')
    expect(getComputedStyle(grid).display).toBe('grid')
    m.unmount()
  })

  it('gives Cta solid outline buttons, Cta soft/solid descriptions and Pricing pressed cycle badges high-contrast foreground colours', () => {
    const mSolid = mount(LandingCases, { case: 'cta-solid-horizontal' })
    const outlineBtn = mSolid.el.querySelectorAll<HTMLElement>('.weft-button')[1]!
    // Draws in --weft-tone-fg rather than neutral text on a solid fill, with wrap enabled for long labels.
    expect(getComputedStyle(outlineBtn).color).toBe(getComputedStyle(mSolid.el.querySelector('.weft-cta__box')!).color)
    expect(getComputedStyle(outlineBtn).whiteSpace).toBe('normal')
    expect(getComputedStyle(mSolid.el.querySelector('.weft-cta__description')!).opacity).not.toBe('0.88')
    mSolid.unmount()

    const mSoft = mount(LandingCases, { case: 'cta-soft' })
    const softDesc = mSoft.el.querySelector<HTMLElement>('.weft-cta__description')!
    const softTitle = mSoft.el.querySelector<HTMLElement>('.weft-cta__title')!
    // Uses --weft-color-text on the soft tint so it meets 4.5:1 across all themes.
    expect(getComputedStyle(softDesc).color).toBe(getComputedStyle(softTitle).color)
    mSoft.unmount()
  })
})
