import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import LinkCases from './fixtures/basics/LinkCases.vue'
import { navigations, press } from './keyboard.ts'
import { mount } from './mount.ts'
import { loadStyles, ruleFor } from './styles.ts'

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
  history.replaceState(null, '', '/')
})
/** Mount a Link reading `Docs`; handlers go in as getters, as a root component reads function props. */
async function link(props: Record<string, unknown>) {
  const m = mount(LinkCases, props)
  mounted.push(m)
  await nextTick()
  return m.el.querySelector('a') as HTMLAnchorElement
}

describe('Link', () => {
  it('is an anchor to its address, with its content', async () => {
    const a = await link({ to: '/docs' })
    expect(a.className).toBe('weft-link')
    expect(a.getAttribute('href')).toBe('/docs')
    expect(a.textContent).toBe('Docs')
    expect(a.hasAttribute('rel')).toBe(false)
    expect(a.hasAttribute('aria-current')).toBe(false)
  })

  it('takes href as well as to', async () => {
    expect((await link({ href: '/pricing' })).getAttribute('href')).toBe('/pricing')
  })

  describe('external', () => {
    it.each(['https://example.com/', 'http://example.com/a', '//cdn.example.com/x'])('%s gets rel="noopener"', async (to) => {
      expect((await link({ to })).getAttribute('rel')).toBe('noopener')
    })

    it('keeps a rel it is given, adding noopener once', async () => {
      expect((await link({ to: 'https://example.com', rel: 'nofollow' })).getAttribute('rel')).toBe('nofollow noopener')
      expect((await link({ to: 'https://example.com', rel: 'noopener' })).getAttribute('rel')).toBe('noopener')
    })

    it('is external when told, and not when told otherwise', async () => {
      expect((await link({ to: '/out', external: true })).getAttribute('rel')).toBe('noopener')
      expect((await link({ to: 'https://example.com', external: false })).hasAttribute('rel')).toBe(false)
    })

    it('leaves mail and phone links, and the target, as given', async () => {
      expect((await link({ to: 'mailto:hi@example.com' })).hasAttribute('rel')).toBe(false)
      const a = await link({ to: 'https://example.com', target: '_blank' })
      expect(a.getAttribute('target')).toBe('_blank')
      expect(a.getAttribute('rel')).toBe('noopener')
    })
  })

  describe('the current page', () => {
    it('gets aria-current="page" when its address is the page the browser is on', async () => {
      history.replaceState(null, '', '/docs/')
      expect((await link({ to: '/docs' })).getAttribute('aria-current')).toBe('page')
      expect((await link({ to: '/docs/?tab=api' })).getAttribute('aria-current')).toBe('page')
      expect((await link({ to: '/pricing' })).hasAttribute('aria-current')).toBe(false)
      expect((await link({ to: 'https://example.com/docs' })).hasAttribute('aria-current')).toBe(false)
    })

    it('is marked from the browser only after mounting, so the server and hydration agree', async () => {
      history.replaceState(null, '', '/docs')
      const m = mount(LinkCases, { to: '/docs' })
      mounted.push(m)
      // The first render is the one the server makes: it cannot know the page.
      const a = m.el.querySelector('a')!
      expect(a.hasAttribute('aria-current')).toBe(false)
      await nextTick()
      expect(a.getAttribute('aria-current')).toBe('page')
    })

    it('takes active over what the browser says', async () => {
      history.replaceState(null, '', '/docs')
      expect((await link({ to: '/docs', active: false })).hasAttribute('aria-current')).toBe(false)
      expect((await link({ to: '/elsewhere', active: true })).getAttribute('aria-current')).toBe('page')
    })
  })

  it('emits click', async () => {
    const onPress = vi.fn()
    const a = await link({ to: '/docs', onPress: () => onPress })
    a.click()
    expect(onPress).toHaveBeenCalledTimes(1)
    expect(onPress.mock.calls[0]![0]).toBeInstanceOf(MouseEvent)
  })

  describe('disabled', () => {
    it('has no address, so it leaves the tab order, and is announced as a disabled link', async () => {
      const a = await link({ to: '/docs', disabled: true })
      expect(a.hasAttribute('href')).toBe(false)
      expect(a.getAttribute('role')).toBe('link')
      expect(a.getAttribute('aria-disabled')).toBe('true')
      expect(a.hasAttribute('rel')).toBe(false)
    })

    it('drops its target, which would have nowhere to go', async () => {
      expect((await link({ to: 'https://example.com', target: '_blank', disabled: true })).hasAttribute('target')).toBe(false)
    })

    it('does nothing when pressed', async () => {
      const onPress = vi.fn()
      const a = await link({ to: '/docs', disabled: true, onPress: () => onPress })
      const press = new MouseEvent('click', { bubbles: true, cancelable: true })
      a.dispatchEvent(press)
      expect(press.defaultPrevented).toBe(true)
      expect(onPress).not.toHaveBeenCalled()
    })
  })

  describe('keyboard, focus and touch', () => {
    let unload: () => void
    beforeEach(() => { unload = loadStyles() })
    afterEach(() => unload())

    it('is a native link: in the tab order, with a visible focus ring', async () => {
      const a = await link({ to: '/docs' })
      expect(a.tabIndex).toBe(0)
      a.focus()
      expect(document.activeElement).toBe(a)
      expect(getComputedStyle(a).outlineStyle).toBe('solid')
    })

    it('is followed with Enter, and not with Space', async () => {
      const a = await link({ to: '/docs' })
      const followed = navigations(a)
      expect(press(a, ' ')).toBe(false)
      expect(followed.urls).toEqual([])
      expect(press(a, 'Enter')).toBe(true)
      expect(followed.urls).toEqual([new URL('/docs', location.href).href])
      followed.restore()
    })

    it('is neither focused nor followed by keyboard when disabled', async () => {
      const a = await link({ to: '/docs', disabled: true })
      const followed = navigations(a)
      // No href: browsers do not let it take focus, so no key reaches it.
      expect(a.hasAttribute('href')).toBe(false)
      expect(a.hasAttribute('tabindex')).toBe(false)
      expect(press(a, 'Enter')).toBe(false)
      expect(followed.urls).toEqual([])
      followed.restore()
    })

    it('in running text, has no hit area of its own, which would cover the lines around it', async () => {
      const a = await link({ to: '/docs' })
      expect(a.hasAttribute('data-standalone')).toBe(false)
      expect(ruleFor('.weft-link::before')).toBeUndefined()
      // It stays an inline element, flowing with the text around it.
      expect(ruleFor('.weft-link')!.style.getPropertyValue('display')).toBe('')
      expect(ruleFor('.weft-link')!.style.getPropertyValue('position')).toBe('')
    })

    it('draws in the primary text colour the theme checked for AA on the page', () => {
      expect(ruleFor('.weft-link')!.style.getPropertyValue('color')).toBe('var(--weft-color-primary-text, LinkText)')
    })

    it('standalone, is an inline box with a hit area of at least --weft-hit-min', async () => {
      const a = await link({ to: '/docs', standalone: true })
      expect(a.dataset.standalone).toBe('true')
      expect(getComputedStyle(a).display).toBe('inline-flex')
      const hit = ruleFor('.weft-link[data-standalone]::before')!
      expect(hit.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(hit.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(ruleFor('.weft-link[data-standalone]')!.style.getPropertyValue('position')).toBe('relative')
    })
  })
})
