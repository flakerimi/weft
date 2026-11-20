// @vitest-environment happy-dom
// Ids inside an island are counted in the island's own scope, named by its
// position on the page, on the server (Island) and in the browser (the island
// runtime) alike, so a hydrated island links to the ids the server wrote.
// The server half is in ids-server.test.ts.
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { ID_SCOPE } from '@weft/ui'
import { ISLAND_ID_SCOPE, startIslands } from '../src/island-runtime.ts'
import Labelled from './fixtures/ids/components/Labelled.vue'

const settle = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('ids in islands', () => {
  it('uses the key useId reads', () => {
    expect(ISLAND_ID_SCOPE).toBe(ID_SCOPE)
  })

  it('match the server after hydration in the browser', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    // Server HTML exactly as renderPage writes it for this page (see ids-server.test.ts).
    const island = (n: number) =>
      `<weft-island data-id="Labelled" data-when="load"><script type="application/json">{}</script>` +
      `<div><button type="button" id="field-i${n}-0">label</button><!----></div></weft-island>`
    document.body.innerHTML = `<main>${island(0)}${island(1)}</main>`
    startIslands({ Labelled: () => Promise.resolve(Labelled as Component) })
    await settle()
    const [first, second] = [...document.querySelectorAll('weft-island')]
    second!.querySelector('button')!.click()
    first!.querySelector('button')!.click()
    await settle()
    expect(first!.querySelector('.late')!.getAttribute('aria-labelledby')).toBe('field-i0-0')
    expect(second!.querySelector('.late')!.getAttribute('aria-labelledby')).toBe('field-i1-0')
    expect(warn).not.toHaveBeenCalled()
  })
})
