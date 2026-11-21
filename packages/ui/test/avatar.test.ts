import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { registerIcon } from '../src/basics/icons.ts'
import Avatar from '../src/basics/Avatar.vue'
import AvatarCases from './fixtures/basics/AvatarCases.vue'
import { mount } from './mount.ts'

registerIcon('bot', '<rect width="16" height="12" x="4" y="8"></rect>')

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
  vi.restoreAllMocks()
})
function avatar(props: Record<string, unknown>) {
  const m = mount(Avatar, props)
  mounted.push(m)
  return m.el.firstElementChild as HTMLElement
}
function cases(props: Record<string, unknown>) {
  const m = mount(AvatarCases, props)
  mounted.push(m)
  return m.el.firstElementChild as HTMLElement
}
/** happy-dom loads no images; say whether one has finished, as a browser's `complete` would. */
const imagesComplete = (complete: boolean) =>
  vi.spyOn(HTMLImageElement.prototype, 'complete', 'get').mockReturnValue(complete)

describe('Avatar', () => {
  it('shows the image, named by alt, at size md', () => {
    imagesComplete(false)
    const a = avatar({ src: '/ada.png', alt: 'Ada Lovelace' })
    expect(a.className).toBe('weft-avatar')
    expect(a.dataset.size).toBe('md')
    const img = a.querySelector('img')!
    expect(img.getAttribute('src')).toBe('/ada.png')
    expect(img.getAttribute('alt')).toBe('Ada Lovelace')
    expect(a.hasAttribute('role')).toBe(false)
    expect(a.querySelector('.weft-avatar__initials')).toBeNull()
  })

  it('falls back to initials from alt when the image fails, named for assistive technology', async () => {
    imagesComplete(false)
    const a = avatar({ src: '/missing.png', alt: 'Ada King Lovelace' })
    a.querySelector('img')!.dispatchEvent(new Event('error'))
    await nextTick()
    expect(a.querySelector('img')).toBeNull()
    const initials = a.querySelector('.weft-avatar__initials')!
    expect(initials.textContent).toBe('AL')
    expect(initials.getAttribute('aria-hidden')).toBe('true')
    expect(a.getAttribute('role')).toBe('img')
    expect(a.getAttribute('aria-label')).toBe('Ada King Lovelace')
  })

  it('falls back when the server-rendered image had already failed before it hydrated', async () => {
    // A broken image the browser has finished with: complete, with no width.
    imagesComplete(true)
    const a = avatar({ src: '/missing.png', alt: 'Ada Lovelace' })
    await nextTick()
    expect(a.querySelector('img')).toBeNull()
    expect(a.querySelector('.weft-avatar__initials')?.textContent).toBe('AL')
  })

  it('shows initials without a src, and text in place of initials', () => {
    expect(avatar({ alt: 'grace hopper' }).querySelector('.weft-avatar__initials')?.textContent).toBe('GH')
    expect(avatar({ alt: 'Cher' }).querySelector('.weft-avatar__initials')?.textContent).toBe('C')
    expect(avatar({ alt: 'Ada Lovelace', text: 'Ad' }).querySelector('.weft-avatar__initials')?.textContent).toBe('Ad')
  })

  it('falls back to a silhouette with neither image nor name, decorative', () => {
    const a = avatar({})
    // Drawn by Avatar itself, so a build that never shows it ships no icon for it.
    const svg = a.querySelector('svg.weft-avatar__silhouette')!
    expect(svg.getAttribute('aria-hidden')).toBe('true')
    expect(svg.innerHTML).not.toBe('')
    expect(a.hasAttribute('role')).toBe(false)
  })

  it('falls back to the icon it is given in place of the silhouette', () => {
    const a = avatar({ icon: 'bot' })
    expect(a.querySelector('svg.weft-icon')!.innerHTML).toContain('rect')
    expect(a.querySelector('.weft-avatar__silhouette')).toBeNull()
  })

  it('tries a new src again after a failure', async () => {
    imagesComplete(false)
    const src = ref('/missing.png')
    const a = avatar({ src: () => src.value, alt: 'Ada Lovelace' })
    a.querySelector('img')!.dispatchEvent(new Event('error'))
    await nextTick()
    expect(a.querySelector('img')).toBeNull()
    src.value = '/ada.png'
    await nextTick()
    expect(a.querySelector('img')?.getAttribute('src')).toBe('/ada.png')
  })

  it('takes a size', () => {
    expect(avatar({ alt: 'A', size: 'lg' }).dataset.size).toBe('lg')
  })
})

describe('AvatarGroup', () => {
  const people = ['Ada Lovelace', 'Grace Hopper', 'Alan Turing', 'Edsger Dijkstra', 'Barbara Liskov']
  const shown = (g: HTMLElement) =>
    Array.from(g.querySelectorAll('.weft-avatar:not(.weft-avatar-group__more)')).map((a) => a.getAttribute('aria-label'))

  it('is a group of every avatar without max', () => {
    const g = cases({ case: 'group', people })
    expect(g.className).toBe('weft-avatar-group')
    expect(g.getAttribute('role')).toBe('group')
    expect(g.getAttribute('aria-label')).toBe('Team')
    expect(shown(g)).toEqual(people)
    expect(g.querySelector('.weft-avatar-group__more')).toBeNull()
  })

  it('shows the first max avatars and counts the rest', () => {
    const g = cases({ case: 'group', people, max: 3 })
    expect(shown(g)).toEqual(people.slice(0, 3))
    const more = g.querySelector('.weft-avatar-group__more') as HTMLElement
    // Drawn as +2, said as "2 more".
    expect(more.querySelector('[aria-hidden="true"]')?.textContent).toBe('+2')
    expect(more.querySelector('.weft-visually-hidden')?.textContent).toBe('2 more')
    expect(more.dataset.size).toBe('md')
  })

  it('recounts as avatars come and go', async () => {
    const list = ref(people.slice(0, 2))
    const g = cases({ case: 'group', people: () => list.value, max: 3 })
    expect(g.querySelector('.weft-avatar-group__more')).toBeNull()
    list.value = people
    await nextTick()
    expect(g.querySelector('.weft-avatar-group__more [aria-hidden="true"]')?.textContent).toBe('+2')
    list.value = people.slice(0, 4)
    await nextTick()
    expect(g.querySelector('.weft-avatar-group__more [aria-hidden="true"]')?.textContent).toBe('+1')
    expect(g.querySelector('.weft-avatar-group__more .weft-visually-hidden')?.textContent).toBe('1 more')
  })

  it('sizes every avatar in it', () => {
    const g = cases({ case: 'group-sized' })
    expect(g.dataset.size).toBe('lg')
    expect((g.querySelector('.weft-avatar') as HTMLElement).dataset.size).toBe('lg')
  })
})
