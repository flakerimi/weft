import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import FileUploadCases from '../fixtures/forms/FileUploadCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from './helpers.ts'
import { $, $$, active, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
let urls: string[]
let revoked: string[]
beforeEach(() => {
  urls = []
  revoked = []
  vi.spyOn(URL, 'createObjectURL').mockImplementation(() => {
    const url = `blob:test/${urls.length}`
    urls.push(url)
    return url
  })
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation((url: string) => void revoked.push(url))
})
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
  vi.restoreAllMocks()
})
type Exposed = {
  value: File[] | File | null
  input: string
  rejected: { file: File; reason: string }[]
  uploaded: [string, unknown][]
  failed: string[]
  changes: number
  submitted: string[]
}
async function cases(props: Record<string, unknown> = {}) {
  const given = Object.fromEntries(Object.entries(props).map(([k, v]) => [k, typeof v === 'function' ? () => v : v]))
  const m = mount(FileUploadCases, given)
  unmount = m.unmount
  await settle()
  return { e: m.exposed as unknown as Exposed, unmount: m.unmount }
}
const png = (name = 'photo.png', size = 1536) => new File([new Uint8Array(size)], name, { type: 'image/png' })
const pdf = (name = 'report.pdf', size = 2_500_000) => new File([new Uint8Array(size)], name, { type: 'application/pdf' })
const txt = (name = 'notes.txt') => new File(['hello'], name, { type: 'text/plain' })
const zone = () => $('.weft-file-upload__zone')
const browse = () => $<HTMLButtonElement>('.weft-file-upload__browse')
const native = () => $<HTMLInputElement>('.weft-file-upload input[type="file"]')
const items = () => $$('.weft-file-upload__file')
const names = () => items().map((li) => li.querySelector('.weft-file-upload__name')!.textContent!.trim())
const live = () => $('.weft-file-upload [aria-live]').textContent!.trim()

function transfer(files: File[]) {
  const dt = new DataTransfer()
  for (const f of files) dt.items.add(f)
  return dt
}
/** A drag of `files` over `el`, then the drop, as a browser sends them. */
async function drop(el: Element, files: File[]) {
  const dt = transfer(files)
  const send = (type: string) => {
    const event = new Event(type, { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'dataTransfer', { value: dt })
    el.dispatchEvent(event)
    return event
  }
  send('dragenter')
  const over = send('dragover')
  await settle()
  const dragging = zone()?.dataset.dragging
  const dropped = send('drop')
  await settle()
  return { over, dropped, dragging }
}
async function paste(el: Element, files: File[]) {
  const event = new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: transfer(files) })
  el.dispatchEvent(event)
  await settle()
  return event
}
/** Files chosen in the platform's dialog. */
async function pick(files: File[]) {
  native().files = transfer(files).files
  native().dispatchEvent(new Event('change', { bubbles: true }))
  await settle()
}

describe('FileUpload', () => {
  describe('on pointer', () => {
    it('is a drop zone with a Browse button named by its field, over a hidden file input', async () => {
      await cases({ multiple: true, accept: 'image/*', description: 'PNG or JPG' })
      expect(zone().textContent).toContain('Drop files here')
      const b = browse()
      expect(b.tagName).toBe('BUTTON')
      expect(b.textContent!.trim()).toBe('Browse')
      expect($('label').getAttribute('for')).toBe(b.id)
      const name = b.getAttribute('aria-labelledby')!.split(' ').map((id) => (id === b.id ? b.textContent!.trim() : document.getElementById(id)!.textContent!.trim()))
      expect(name).toEqual(['Attachments', 'Browse'])
      expect(b.getAttribute('aria-describedby')!.split(' ')).toEqual([$('.weft-form-field__description').id, $('.weft-file-upload__description').id])
      const n = native()
      expect(n.tabIndex).toBe(-1)
      expect(n.getAttribute('aria-hidden')).toBe('true')
      expect(n.name).toBe('files')
      expect(n.accept).toBe('image/*')
      expect(n.multiple).toBe(true)
      expect(items()).toHaveLength(0)
    })

    it('opens the file dialog from its button and from a press on the zone, once each', async () => {
      await cases()
      const click = vi.spyOn(native(), 'click').mockImplementation(() => {})
      browse().click()
      expect(click).toHaveBeenCalledTimes(1)
      zone().click()
      expect(click).toHaveBeenCalledTimes(2)
    })

    it('takes files chosen in the dialog, listing each with its size in the locale', async () => {
      const { e } = await cases({ multiple: true, locale: 'de-DE' })
      await pick([png(), pdf()])
      expect((e.value as File[]).map((f) => f.name)).toEqual(['photo.png', 'report.pdf'])
      expect(names()).toEqual(['photo.png', 'report.pdf'])
      expect(items().map((li) => li.querySelector('.weft-file-upload__size')!.textContent!.trim())).toEqual(['1,5 kB', '2,5 MB'])
      expect(e.changes).toBe(1)
      expect(live()).toBe('photo.png, report.pdf added')
    })

    it('takes dropped files, marking the zone while a drag is over it', async () => {
      const { e } = await cases({ multiple: true })
      const { over, dropped, dragging } = await drop(zone(), [png(), pdf()])
      expect(dragging).toBe('true')
      expect(over.defaultPrevented).toBe(true)
      expect(dropped.defaultPrevented).toBe(true)
      expect(zone().dataset.dragging).toBeUndefined()
      expect((e.value as File[]).map((f) => f.name)).toEqual(['photo.png', 'report.pdf'])
      // The native input holds them too, so the form submits them.
      expect(native().files!.length).toBe(2)
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(e.submitted).toEqual(['photo.png', 'report.pdf'])
    })

    it('takes pasted files, and leaves a paste with none alone', async () => {
      const { e } = await cases({ multiple: true })
      browse().focus()
      const event = await paste(active()!, [png('screenshot.png')])
      expect(event.defaultPrevented).toBe(true)
      expect((e.value as File[]).map((f) => f.name)).toEqual(['screenshot.png'])
      const text = new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: new DataTransfer() })
      browse().dispatchEvent(text)
      expect(text.defaultPrevented).toBe(false)
    })

    it('refuses files of another type, too large, or past the limit, saying why', async () => {
      const { e } = await cases({ multiple: true, accept: 'image/*,.pdf', maxSize: 2_000_000, maxFiles: 2 })
      await drop(zone(), [txt(), png(), pdf('big.pdf', 3_000_000), png('b.png'), png('c.png')])
      expect((e.value as File[]).map((f) => f.name)).toEqual(['photo.png', 'b.png'])
      expect(e.rejected.map((r) => [r.file.name, r.reason])).toEqual([
        ['notes.txt', 'type'],
        ['big.pdf', 'size'],
        ['c.png', 'count'],
      ])
      expect(live()).toContain('notes.txt, big.pdf, c.png not added')
    })

    it('keeps one file without multiple, a new one replacing it', async () => {
      const { e } = await cases()
      expect(zone().textContent).toContain('Drop a file here')
      await drop(zone(), [png('a.png'), png('b.png')])
      expect((e.value as File).name).toBe('a.png')
      await pick([pdf()])
      expect((e.value as File).name).toBe('report.pdf')
      expect(names()).toEqual(['report.pdf'])
      expect(revoked).toEqual(['blob:test/0'])
    })

    it('previews images through object URLs, revoked when removed or gone, never reading a file', async () => {
      const reader = vi.spyOn(globalThis, 'FileReader')
      const { e, unmount: gone } = await cases({ multiple: true })
      await pick([png('a.png'), pdf(), png('b.png')])
      expect(urls).toEqual(['blob:test/0', 'blob:test/1'])
      const thumbs = items().map((li) => li.querySelector('img')?.getAttribute('src') ?? null)
      expect(thumbs).toEqual(['blob:test/0', null, 'blob:test/1'])
      expect(items()[0]!.querySelector('img')!.getAttribute('alt')).toBe('')
      $<HTMLButtonElement>('[aria-label="Remove a.png"]').click()
      await settle()
      expect(revoked).toEqual(['blob:test/0'])
      expect((e.value as File[]).map((f) => f.name)).toEqual(['report.pdf', 'b.png'])
      gone()
      unmount = undefined
      expect(revoked).toEqual(['blob:test/0', 'blob:test/1'])
      expect(reader).not.toHaveBeenCalled()
    })

    it('removes a file with its button, focus moving to the next, then to Browse', async () => {
      const { e } = await cases({ multiple: true })
      await pick([png('a.png'), png('b.png')])
      const first = $<HTMLButtonElement>('[aria-label="Remove a.png"]')
      first.focus()
      first.click()
      await settle()
      expect(active()).toBe($('[aria-label="Remove b.png"]'))
      expect(live()).toBe('a.png removed')
      ;(active() as HTMLButtonElement).click()
      await settle()
      expect(e.value).toEqual([])
      expect(active()).toBe(browse())
      expect(native().files!.length).toBe(0)
    })

    it('ignores everything while disabled', async () => {
      const { e } = await cases({ multiple: true, disabled: true })
      expect(browse().disabled).toBe(true)
      expect(native().disabled).toBe(true)
      // A file dropped is still kept from the browser, which would open it in place of the page.
      const { over, dropped } = await drop(zone(), [png()])
      expect(over.defaultPrevented).toBe(true)
      expect(dropped.defaultPrevented).toBe(true)
      expect(zone().dataset.dragging).toBeUndefined()
      await paste(zone(), [png()])
      expect(e.value).toEqual([])
    })

    it('keeps the browser from opening a file dropped anywhere on it, its list too', async () => {
      const { e } = await cases({ multiple: true })
      await pick([png('a.png')])
      const { over, dropped } = await drop($('.weft-file-upload__files'), [png('b.png')])
      expect(over.defaultPrevented).toBe(true)
      expect(dropped.defaultPrevented).toBe(true)
      expect((e.value as File[]).map((f) => f.name)).toEqual(['a.png'])
    })

    it('keeps the native input to the files it holds after the dialog: a refused choice submits nothing', async () => {
      const { e } = await cases({ accept: 'image/*' })
      await pick([txt()])
      expect(e.value).toBeNull()
      expect(native().files!.length).toBe(0)
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(e.submitted.filter(Boolean)).toEqual([])
    })

    it('keeps the native input to every file listed after a second choice in the dialog', async () => {
      const { e } = await cases({ multiple: true, accept: 'image/*' })
      await pick([png('a.png')])
      await pick([txt(), png('b.png')])
      expect([...native().files!].map((f) => f.name)).toEqual(['a.png', 'b.png'])
      await pick([txt()])
      expect([...native().files!].map((f) => f.name)).toEqual(['a.png', 'b.png'])
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(e.submitted).toEqual(['a.png', 'b.png'])
    })

    it('starts again from each choice in the dialog with reset', async () => {
      const { e } = await cases({ multiple: true, reset: true })
      await pick([png('a.png'), png('b.png')])
      await pick([png('c.png')])
      expect((e.value as File[]).map((f) => f.name)).toEqual(['c.png'])
      expect(revoked).toEqual(['blob:test/0', 'blob:test/1'])
      // A drop still adds.
      await drop(zone(), [png('d.png')])
      expect((e.value as File[]).map((f) => f.name)).toEqual(['c.png', 'd.png'])
    })

    it('lays files out as a list or a grid of tiles, below the zone or inside it', async () => {
      const first = await cases({ multiple: true, layout: 'grid', position: 'inside' })
      expect(zone().textContent).toContain('Drop files here')
      await pick([png('a.png')])
      const list = $('.weft-file-upload__files')
      expect(list.dataset.layout).toBe('grid')
      expect(zone().contains(list)).toBe(true)
      // With files inside, the zone's prompt gives way to them; Browse stays.
      expect($('.weft-file-upload__label')).toBeNull()
      expect(browse()).not.toBeNull()
      const click = vi.spyOn(native(), 'click').mockImplementation(() => {})
      items()[0]!.click()
      expect(click).not.toHaveBeenCalled()
      first.unmount()
      unmount = undefined
      await cases({ multiple: true })
      await pick([png('a.png')])
      expect($('.weft-file-upload__files').dataset.layout).toBe('list')
      expect(zone().contains($('.weft-file-upload__files'))).toBe(false)
    })

    it('marks the zone while dragging and rings its button on focus', async () => {
      const unstyle = loadStyles()
      try {
        await cases()
        expect(ruleFor('.weft-file-upload__zone[data-dragging]')).toBeDefined()
        expect(ruleFor(':focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      } finally {
        unstyle()
      }
    })
  })

  describe('uploading', () => {
    type Call = { file: File; signal: AbortSignal; onProgress: (f: number) => void; resolve: (v: unknown) => void; reject: (e: unknown) => void }
    function server() {
      const calls: Call[] = []
      const upload = (file: File, ctx: { signal: AbortSignal; onProgress: (f: number) => void }) =>
        new Promise((resolve, reject) => calls.push({ file, ...ctx, resolve, reject }))
      return { calls, upload }
    }
    const bar = (name: string) => $(`[role="progressbar"][aria-label="Uploading ${name}"]`)

    it('uploads each file added, showing its progress, then that it is done', async () => {
      const s = server()
      const { e } = await cases({ multiple: true, upload: s.upload })
      await pick([png('a.png'), pdf()])
      expect(s.calls.map((c) => c.file.name)).toEqual(['a.png', 'report.pdf'])
      expect(bar('a.png').hasAttribute('aria-valuenow')).toBe(false)
      s.calls[0]!.onProgress(0.4)
      await settle()
      expect(bar('a.png').getAttribute('aria-valuenow')).toBe('40')
      expect(items()[0]!.dataset.status).toBe('uploading')
      s.calls[0]!.resolve({ id: 7 })
      await settle()
      expect(bar('a.png')).toBeNull()
      expect(items()[0]!.dataset.status).toBe('done')
      expect(e.uploaded).toEqual([['a.png', { id: 7 }]])
      expect(live()).toBe('a.png uploaded')
    })

    it('shows a failure with a retry that uploads again', async () => {
      const s = server()
      const { e } = await cases({ multiple: true, upload: s.upload })
      await pick([png('a.png')])
      s.calls[0]!.reject(new Error('503'))
      await settle()
      expect(items()[0]!.dataset.status).toBe('error')
      expect(items()[0]!.textContent).toContain('Upload failed')
      expect(e.failed).toEqual(['a.png'])
      $<HTMLButtonElement>('[aria-label="Retry a.png"]').click()
      await settle()
      expect(s.calls).toHaveLength(2)
      expect(items()[0]!.dataset.status).toBe('uploading')
    })

    it('cancels an upload, aborting its signal, and removes the file', async () => {
      const s = server()
      const { e } = await cases({ multiple: true, upload: s.upload })
      await pick([png('a.png')])
      $<HTMLButtonElement>('[aria-label="Cancel upload of a.png"]').click()
      await settle()
      expect(s.calls[0]!.signal.aborted).toBe(true)
      expect(e.value).toEqual([])
      // A late answer changes nothing.
      s.calls[0]!.resolve('late')
      await settle()
      expect(e.uploaded).toEqual([])
    })

    it('only collects files without an upload function', async () => {
      await cases({ multiple: true })
      await pick([png('a.png')])
      expect($('[role="progressbar"]')).toBeNull()
      expect(items()[0]!.dataset.status).toBe('idle')
    })
  })

  describe('on touch', () => {
    it('is one button offering the camera, photos or files through accept and capture, the same list below', async () => {
      const unstyle = loadStyles()
      try {
        const { e } = await cases({ input: 'touch', multiple: true, accept: 'image/*', capture: 'environment' })
        expect(zone()).toBeNull()
        const b = browse()
        expect(b.textContent!.trim()).toBe('Choose files')
        expect(px(getComputedStyle(b).blockSize)).toBe(44)
        expect(native().accept).toBe('image/*')
        expect(native().getAttribute('capture')).toBe('environment')
        const click = vi.spyOn(native(), 'click').mockImplementation(() => {})
        b.click()
        expect(click).toHaveBeenCalledTimes(1)
        await pick([png('a.png')])
        expect(names()).toEqual(['a.png'])
        expect((e.value as File[]).length).toBe(1)
        // The remove button is a CloseButton, whose hit area is 44 px on touch.
        expect($('[aria-label="Remove a.png"]').classList.contains('weft-close-button')).toBe(true)
      } finally {
        unstyle()
      }
    })

    it('keeps its files when the input switches', async () => {
      const m = mount(FileUploadCases, { multiple: true })
      unmount = m.unmount
      await settle()
      await pick([png('a.png')])
      ;(m.exposed as { input: string }).input = 'touch'
      await settle()
      expect(zone()).toBeNull()
      expect(names()).toEqual(['a.png'])
      expect(urls).toHaveLength(1)
    })
  })
})
