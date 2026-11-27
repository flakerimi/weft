import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { registerIcon } from '../src/basics/icons.ts'
import Button from '../src/basics/Button.vue'
import ButtonCases from './fixtures/basics/ButtonCases.vue'
import TooltipIconButton from './fixtures/basics/TooltipIconButton.vue'
import { navigations, press } from './keyboard.ts'
import { mount } from './mount.ts'
import { loadStyles, ruleFor } from './styles.ts'
// What a Button with perm loads on demand, taken as fetched, as it is once one on the page has asked.
import '../src/session/load.ts'

registerIcon('plus', '<path d="M5 12h14"></path>')
registerIcon('arrow-right', '<path d="M12 5l7 7"></path>')
registerIcon('loader', '<path d="M12 2v4"></path>')

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
})
/** Mount a Button. A function prop of a root component is read as a getter, so pass handlers as `() => handler`. */
function render(props: Record<string, unknown>) {
  const m = mount(Button, props)
  mounted.push(m)
  return m.el.firstElementChild as HTMLElement
}
function cases(props: Record<string, unknown>) {
  const m = mount(ButtonCases, props)
  mounted.push(m)
  return m.el
}

describe('Button', () => {
  it('is a native button of type button, solid primary md, by default', () => {
    const b = render({ label: 'Save' })
    expect(b.tagName).toBe('BUTTON')
    expect(b.getAttribute('type')).toBe('button')
    expect(b.className).toBe('weft-button')
    expect(b.dataset).toMatchObject({ variant: 'solid', tone: 'primary', size: 'md' })
    expect(b.textContent).toBe('Save')
    expect(b.hasAttribute('disabled')).toBe(false)
    expect(b.hasAttribute('aria-busy')).toBe(false)
  })

  it('takes variant, tone, size, block and type', () => {
    const b = render({ label: 'Delete', variant: 'outline', tone: 'danger', size: 'lg', block: true, type: 'submit' })
    expect(b.dataset).toMatchObject({ variant: 'outline', tone: 'danger', size: 'lg', block: 'true' })
    expect(b.getAttribute('type')).toBe('submit')
  })

  it('draws a leading icon and a trailing icon around its label, hidden from assistive technology', () => {
    const b = render({ label: 'Next', icon: 'plus', trailingIcon: 'arrow-right' })
    const icons = b.querySelectorAll('svg.weft-icon')
    expect(icons).toHaveLength(2)
    expect(icons[0]!.innerHTML).toBe('<path d="M5 12h14"></path>')
    expect(icons[1]!.innerHTML).toBe('<path d="M12 5l7 7"></path>')
    for (const i of icons) expect(i.getAttribute('aria-hidden')).toBe('true')
    const parts = Array.from(b.querySelector('.weft-button__content')!.children)
    expect(parts.map((p) => p.getAttribute('class'))).toEqual(['weft-icon', 'weft-button__label', 'weft-icon'])
    expect(b.hasAttribute('data-square')).toBe(false)
  })

  it('is square with an icon and no label', () => {
    const b = render({ icon: 'plus', 'aria-label': 'Add' })
    expect(b.dataset.square).toBe('true')
    expect(b.getAttribute('aria-label')).toBe('Add')
    expect(b.querySelector('.weft-button__label')).toBeNull()
  })

  it('puts its default slot in place of the label', () => {
    const b = cases({ case: 'slot' }).querySelector('button')!
    expect(b.querySelector('.mine')?.textContent).toBe('Add')
    expect(b.textContent).toBe('Add')
    expect(b.hasAttribute('data-square')).toBe(false)
  })

  it('passes attributes such as aria-label and class to its element', () => {
    const b = render({ label: 'x', class: 'extra', 'data-test': 'go' })
    expect(b.classList.contains('weft-button')).toBe(true)
    expect(b.classList.contains('extra')).toBe(true)
    expect(b.dataset.test).toBe('go')
  })

  it('emits click when pressed', () => {
    const onClick = vi.fn()
    const b = render({ label: 'Go', onClick: () => onClick })
    b.click()
    expect(onClick).toHaveBeenCalledTimes(1)
    expect(onClick.mock.calls[0]![0]).toBeInstanceOf(MouseEvent)
  })

  it('takes leading and trailing slots in place of its icons', () => {
    const b = cases({ case: 'ends' }).querySelector('button')!
    const parts = Array.from(b.querySelector('.weft-button__content')!.children).map((p) => p.getAttribute('class'))
    expect(parts).toEqual(['lead', 'weft-button__label', 'trail'])
    expect(b.querySelector('svg')).toBeNull()
  })

  it('is square when told, whatever its content', () => {
    expect(render({ label: 'A', square: true }).dataset.square).toBe('true')
  })

  it('marks itself active, and as the current page when it is a link', () => {
    const b = render({ label: 'Grid', active: true })
    expect(b.dataset.active).toBe('true')
    expect(b.hasAttribute('aria-current')).toBe(false)
    expect(render({ label: 'Docs', to: '/docs', active: true }).getAttribute('aria-current')).toBe('page')
    expect(render({ label: 'Docs', to: '/docs' }).hasAttribute('aria-current')).toBe(false)
  })

  describe('naming', () => {
    // First: the warning is given once per page, so a later case would pass whatever it did.
    it('does not warn when a Tooltip around it names it', async () => {
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
      const m = mount(TooltipIconButton)
      mounted.push(m)
      await nextTick()
      expect(m.el.querySelector('button')!.getAttribute('aria-label')).toBe('Add')
      expect(warn).not.toHaveBeenCalled()
      warn.mockRestore()
    })

    it('warns once, in development, when it has only an icon and no name', () => {
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
      render({ icon: 'plus', 'aria-label': 'Add' })
      render({ icon: 'plus', label: 'Add' })
      render({ icon: 'plus', 'aria-labelledby': 'x' })
      expect(warn).not.toHaveBeenCalled()
      render({ icon: 'plus' })
      render({ icon: 'plus' })
      expect(warn).toHaveBeenCalledTimes(1)
      expect(String(warn.mock.calls[0]![0])).toBe(
        'Button: a button with only an icon needs a name; give it a label or an aria-label, which assistive technology reads in place of the icon.',
      )
      warn.mockRestore()
    })
  })

  describe('disabled', () => {
    it('is natively disabled, so mouse, keyboard and form submit cannot activate it', () => {
      const onClick = vi.fn()
      const b = render({ label: 'Go', disabled: true, onClick: () => onClick })
      expect((b as HTMLButtonElement).disabled).toBe(true)
      b.click()
      expect(onClick).not.toHaveBeenCalled()
    })
  })

  describe('loading', () => {
    it('is busy and unavailable but keeps focus, its content and so its width', () => {
      const b = render({ label: 'Save', icon: 'plus', loading: true })
      expect(b.getAttribute('aria-busy')).toBe('true')
      expect(b.getAttribute('aria-disabled')).toBe('true')
      // Not natively disabled, which would drop focus from a button pressed by keyboard.
      expect((b as HTMLButtonElement).disabled).toBe(false)
      expect(b.dataset.loading).toBe('true')
      // The content stays in place (hidden by CSS) so the button keeps its width.
      expect(b.querySelector('.weft-button__label')?.textContent).toBe('Save')
      const spinner = b.querySelector('.weft-spinner')!
      expect(spinner.getAttribute('aria-hidden')).toBe('true')
    })

    it('blocks a click, from the mouse or from a keyboard press, and the form submit it would cause', () => {
      const onPress = vi.fn()
      const onSubmit = vi.fn()
      const el = cases({ case: 'form', loading: true, onPress: () => onPress, onSubmit: () => onSubmit })
      const b = el.querySelector('button')!
      // Enter and Space on a native button reach it as a click (detail 0).
      const keyboard = new MouseEvent('click', { bubbles: true, cancelable: true, detail: 0 })
      b.dispatchEvent(keyboard)
      expect(keyboard.defaultPrevented).toBe(true)
      b.click()
      expect(onPress).not.toHaveBeenCalled()
      expect(onSubmit).not.toHaveBeenCalled()
    })

    it('submits its form once it is not loading', () => {
      const onPress = vi.fn()
      const onSubmit = vi.fn()
      const el = cases({ case: 'form', onPress: () => onPress, onSubmit: () => onSubmit })
      el.querySelector('button')!.click()
      expect(onPress).toHaveBeenCalledTimes(1)
      expect(onSubmit).toHaveBeenCalledTimes(1)
    })

    it('shows loadingIcon, turning, in place of the spinner', () => {
      const b = render({ label: 'Save', loading: true, loadingIcon: 'loader' })
      const icon = b.querySelector('.weft-button__spinner svg.weft-icon')!
      expect(icon.innerHTML).toContain('M12 2v4')
      expect(icon.getAttribute('class')).toContain('weft-button__loading-icon')
      expect(b.querySelector('.weft-spinner')).toBeNull()
    })

    it('follows the loading prop live', async () => {
      const loading = ref(false)
      const onClick = vi.fn()
      const b = render({ label: 'Save', loading: () => loading.value, onClick: () => onClick })
      expect(b.querySelector('.weft-spinner')).toBeNull()
      loading.value = true
      await nextTick()
      expect(b.getAttribute('aria-busy')).toBe('true')
      b.click()
      expect(onClick).not.toHaveBeenCalled()
      loading.value = false
      await nextTick()
      expect(b.hasAttribute('aria-busy')).toBe(false)
      expect(b.hasAttribute('aria-disabled')).toBe(false)
      b.click()
      expect(onClick).toHaveBeenCalledTimes(1)
    })
  })

  describe('with to', () => {
    it('is a link to that address, with no type', () => {
      const onClick = vi.fn()
      const a = render({ label: 'Docs', to: '/docs', onClick: () => onClick })
      expect(a.tagName).toBe('A')
      expect(a.getAttribute('href')).toBe('/docs')
      expect(a.hasAttribute('type')).toBe(false)
      expect(a.className).toBe('weft-button')
      const press = new MouseEvent('click', { bubbles: true, cancelable: true })
      a.dispatchEvent(press)
      expect(onClick).toHaveBeenCalledTimes(1)
      expect(press.defaultPrevented).toBe(false)
    })

    it('to another site gets rel="noopener", as a Link does', () => {
      expect(render({ label: 'Out', to: 'https://example.com' }).getAttribute('rel')).toBe('noopener')
      expect(render({ label: 'In', to: '/docs' }).hasAttribute('rel')).toBe(false)
    })

    it('takes rel and target, merging noopener into rel', () => {
      const a = render({ label: 'Out', to: 'https://example.com', rel: 'nofollow', target: '_blank' })
      expect(a.getAttribute('rel')).toBe('nofollow noopener')
      expect(a.getAttribute('target')).toBe('_blank')
      expect(render({ label: 'In', to: '/docs', rel: 'help' }).getAttribute('rel')).toBe('help')
      // A button has no rel or target.
      const b = render({ label: 'Go', rel: 'nofollow', target: '_blank' })
      expect(b.hasAttribute('rel')).toBe(false)
      expect(b.hasAttribute('target')).toBe(false)
    })

    it('is followed with Enter', () => {
      const a = render({ label: 'Docs', to: '/docs' })
      const followed = navigations(a)
      expect(press(a, 'Enter')).toBe(true)
      expect(followed.urls).toEqual([new URL('/docs', location.href).href])
      followed.restore()
    })

    it('when disabled, drops its address and leaves the tab order, announced as a disabled link', () => {
      const onClick = vi.fn()
      const a = render({ label: 'Docs', to: '/docs', disabled: true, onClick: () => onClick })
      expect(a.hasAttribute('href')).toBe(false)
      expect(a.getAttribute('role')).toBe('link')
      expect(a.getAttribute('aria-disabled')).toBe('true')
      const press = new MouseEvent('click', { bubbles: true, cancelable: true })
      a.dispatchEvent(press)
      expect(press.defaultPrevented).toBe(true)
      expect(onClick).not.toHaveBeenCalled()
    })

    it('when loading, keeps focus but does not navigate, by click, middle click or Enter', () => {
      const onClick = vi.fn()
      const a = render({ label: 'Docs', to: '/docs', loading: true, onClick: () => onClick })
      const followed = navigations(a)
      expect(a.getAttribute('href')).toBe('/docs')
      expect(a.getAttribute('aria-busy')).toBe('true')
      const click = new MouseEvent('click', { bubbles: true, cancelable: true })
      a.dispatchEvent(click)
      expect(click.defaultPrevented).toBe(true)
      // A middle click opens a link in a new tab as the default of auxclick.
      const middle = new MouseEvent('auxclick', { bubbles: true, cancelable: true, button: 1 })
      a.dispatchEvent(middle)
      expect(middle.defaultPrevented).toBe(true)
      press(a, 'Enter')
      expect(document.activeElement).toBe(a)
      expect(followed.urls).toEqual([])
      expect(onClick).not.toHaveBeenCalled()
      followed.restore()
    })

    it('lets a middle click through when not loading', () => {
      const a = render({ label: 'Docs', to: '/docs' })
      const middle = new MouseEvent('auxclick', { bubbles: true, cancelable: true, button: 1 })
      a.dispatchEvent(middle)
      expect(middle.defaultPrevented).toBe(false)
    })
  })

  describe('keyboard and focus', () => {
    let unload: () => void
    beforeEach(() => { unload = loadStyles() })
    afterEach(() => unload())

    it('is in the tab order and shows a focus ring when focused', () => {
      const b = render({ label: 'Go' })
      expect(b.tabIndex).toBe(0)
      b.focus()
      expect(document.activeElement).toBe(b)
      expect(getComputedStyle(b).outlineStyle).toBe('solid')
      expect(getComputedStyle(b).outlineWidth).toBe('2px')
    })

    it('is pressed with Enter and with Space', () => {
      const onClick = vi.fn()
      const b = render({ label: 'Go', onClick: () => onClick })
      expect(press(b, 'Enter')).toBe(true)
      expect(press(b, ' ')).toBe(true)
      expect(onClick).toHaveBeenCalledTimes(2)
    })

    it('submits its form with Enter, unless it is loading', () => {
      const onSubmit = vi.fn()
      expect(press(cases({ case: 'form', onSubmit: () => onSubmit }).querySelector('button')!, 'Enter')).toBe(true)
      expect(onSubmit).toHaveBeenCalledTimes(1)
      const busy = cases({ case: 'form', loading: true, onSubmit: () => onSubmit }).querySelector('button')!
      press(busy, 'Enter')
      press(busy, ' ')
      expect(onSubmit).toHaveBeenCalledTimes(1)
      expect(document.activeElement).toBe(busy)
    })

    it('cannot be reached or pressed by keyboard when disabled', () => {
      const onClick = vi.fn()
      const b = render({ label: 'Go', disabled: true, onClick: () => onClick })
      expect(press(b, 'Enter')).toBe(false)
      expect(press(b, ' ')).toBe(false)
      expect(onClick).not.toHaveBeenCalled()
    })

    it('stays in the tab order while loading, so keyboard focus is not lost', () => {
      const b = render({ label: 'Go', loading: true })
      b.focus()
      expect(document.activeElement).toBe(b)
    })
  })

  describe('touch', () => {
    let unload: () => void
    beforeEach(() => { unload = loadStyles() })
    afterEach(() => unload())

    it('grows its hit area to --weft-hit-min with a pseudo-element, centred on the button', () => {
      const hit = ruleFor('.weft-button::before')!
      expect(hit).toBeDefined()
      expect(hit.style.getPropertyValue('content')).toMatch(/^(""|'')$/)
      expect(hit.style.getPropertyValue('position')).toBe('absolute')
      expect(hit.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(hit.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(ruleFor('.weft-button')!.style.getPropertyValue('position')).toBe('relative')
    })

    /** A length in px; happy-dom resolves the var()s in max() but leaves max() itself to the reader. */
    const px = (value: string) => {
      const max = /^max\((.*)\)$/.exec(value)
      return max ? Math.max(...max[1]!.split(',').map((v) => parseFloat(v))) : parseFloat(value)
    }
    const heights = (input: string, size: string) => {
      const el = cases({ case: 'adaptive', input, size })
      const height = (sel: string) => px(getComputedStyle(el.querySelector(sel)!).blockSize)
      return { plain: height('.plain'), square: height('.square'), hit: px(getComputedStyle(el.querySelector('.hit-probe')!).minBlockSize) }
    }

    it('is 44 px tall at md on touch, and its control height on pointer', () => {
      expect(heights('touch', 'md')).toMatchObject({ plain: 44, hit: 44 })
      expect(heights('pointer', 'md')).toMatchObject({ plain: 36, hit: 24 })
    })

    it('keeps small and square buttons at their drawn size, reaching 44 px through the hit area', () => {
      expect(heights('touch', 'sm')).toMatchObject({ plain: 28, square: 28 })
      expect(heights('pointer', 'sm')).toMatchObject({ plain: 28, square: 28 })
      expect(heights('touch', 'md').square).toBe(36)
      expect(heights('pointer', 'md').square).toBe(36)
      expect(heights('touch', 'lg')).toMatchObject({ plain: 44, square: 44 })
    })
  })
})

// perm: the button is drawn only for a user with the permission. Who is signed
// in comes from GET /api/auth/me (the contract's Me JSON, stubbed), asked once
// for this file: the session is module state, so these tests run in order,
// the first while it is still loading, and each later one asks again.
describe('Button perm', () => {
  const me = (permissions: string[], role = 'member') => ({
    id: 7,
    created_at: '2026-11-28T09:00:00Z',
    updated_at: '2026-11-28T09:00:00Z',
    email: 'mia@example.com',
    name: 'Mia',
    role,
    permissions,
  })
  const answer = (status: number, body: unknown) =>
    new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } })
  const fetchMock = vi.fn()
  beforeEach(() => vi.stubGlobal('fetch', fetchMock))
  afterEach(() => vi.unstubAllGlobals())
  const settle = () => new Promise((r) => setTimeout(r, 0))
  /** The button a mounted Button drew, or null when it drew nothing. */
  function drawn(props: Record<string, unknown>) {
    const m = mount(Button, props)
    mounted.push(m)
    return () => m.el.querySelector('button, a')
  }
  async function signIn(body: ReturnType<typeof me> | null) {
    const { useSession } = await import('../src/session/session.ts')
    fetchMock.mockResolvedValueOnce(body ? answer(200, body) : answer(401, { error: { code: 'unauthorized', message: 'Sign in to do that.' } }))
    await useSession().refresh()
    await nextTick()
  }

  it('without perm nothing changes and no request is made', async () => {
    const b = drawn({ label: 'Save' })
    await settle()
    expect(b()?.textContent).toBe('Save')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('Button with perm renders nothing while loading, then appears', async () => {
    let resolve!: (r: Response) => void
    fetchMock.mockReturnValueOnce(new Promise<Response>((r) => (resolve = r)))
    const b = drawn({ label: 'Edit', perm: 'products:edit', class: 'mine', 'data-test': 'edit' })
    await settle()
    expect(b()).toBeNull()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0]![0]).toBe('/api/auth/me')
    // A second button asks nothing more.
    drawn({ label: 'Delete', perm: 'products:delete' })
    resolve(answer(200, me(['products:edit', 'products:read'])))
    await settle()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const el = b() as HTMLElement
    expect(el.textContent).toBe('Edit')
    // Attributes still fall through to the element.
    expect(el.classList.contains('weft-button')).toBe(true)
    expect(el.classList.contains('mine')).toBe(true)
    expect(el.dataset.test).toBe('edit')
  })

  it('Button with perm stays hidden without it', async () => {
    await signIn(me(['products:read']))
    const edit = drawn({ label: 'Edit', perm: 'products:edit' })
    const link = drawn({ label: 'Reports', to: '/reports', perm: 'reports:read' })
    const read = drawn({ label: 'Open', perm: 'products:read' })
    await settle()
    expect(edit()).toBeNull()
    expect(link()).toBeNull()
    expect(read()?.textContent).toBe('Open')
  })

  it('follows the session: the owner, a role change and signing out', async () => {
    const users = drawn({ label: 'Users', perm: 'users:read' })
    await settle()
    expect(users()).toBeNull()
    await signIn(me([], 'owner'))
    expect(users()?.textContent).toBe('Users')
    await signIn(me(['products:read']))
    expect(users()).toBeNull()
    await signIn(me(['users:read'], 'administrator'))
    expect(users()?.textContent).toBe('Users')
    await signIn(null)
    expect(users()).toBeNull()
  })

  it('a perm given later is followed', async () => {
    await signIn(me(['products:read']))
    const perm = ref<string | undefined>(undefined)
    const b = drawn({ label: 'Edit', perm: () => perm.value })
    await settle()
    expect(b()).not.toBeNull()
    perm.value = 'products:edit'
    await settle()
    expect(b()).toBeNull()
    perm.value = 'products:read'
    await settle()
    expect(b()).not.toBeNull()
  })
})
