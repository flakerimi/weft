// A file of its own: Button warns about an unnamed icon once per page.
import { expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import TooltipContentIconButton from '../fixtures/basics/TooltipContentIconButton.vue'
import { mount } from '../mount.ts'

it("names only its trigger: an icon-only Button in a Tooltip's content still warns", async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const m = mount(TooltipContentIconButton)
  await nextTick()
  await new Promise((r) => setTimeout(r, 0))
  expect(document.querySelector('.weft-tooltip .weft-button')).not.toBeNull()
  // (Icon warns too: no build registered "plus" here.)
  const fromButton = warn.mock.calls.map((c) => String(c[0])).filter((m) => m.startsWith('Button:'))
  expect(fromButton).toEqual([expect.stringContaining('Button: a button with only an icon needs a name')])
  m.unmount()
  warn.mockRestore()
})
