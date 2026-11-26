import { afterEach, expect, it, vi } from 'vitest'
import { useParams } from '../src/index.ts'

/** A page the build wrote with `route` on <html> (none for undefined), opened at `path`. */
function open(route: string | undefined, path: string) {
  if (route === undefined) delete document.documentElement.dataset.weftRoute
  else document.documentElement.dataset.weftRoute = route
  history.replaceState(null, '', path)
}

afterEach(() => {
  vi.unstubAllGlobals()
  open(undefined, '/')
})

it('useParams reads the params of the address against the pattern the build wrote', () => {
  open('/products/:id', '/products/42')
  expect(useParams()).toEqual({ id: '42' })
  open('/:section/:id', '/shop/7/')
  expect(useParams()).toEqual({ section: 'shop', id: '7' })
})

it('decodes each value, and leaves one that does not decode as it is', () => {
  open('/products/:id', '/products/caf%C3%A9%20au%20lait')
  expect(useParams()).toEqual({ id: 'café au lait' })
  open('/products/:id', '/products/100%25')
  expect(useParams()).toEqual({ id: '100%' })
  open('/products/:id', '/products/%E0%A4%A')
  expect(useParams()).toEqual({ id: '%E0%A4%A' })
  // A static segment is compared as the server compares it, decoded.
  open('/café/:id', '/caf%C3%A9/1')
  expect(useParams()).toEqual({ id: '1' })
})

it('gives a catch-all the rest of the path without its leading slash, or an empty string', () => {
  open('/docs/:...rest', '/docs/guides/islands%20here')
  expect(useParams()).toEqual({ rest: 'guides/islands here' })
  open('/docs/:...rest', '/docs')
  expect(useParams()).toEqual({ rest: '' })
  open('/:...rest', '/')
  expect(useParams()).toEqual({ rest: '' })
})

it('is empty on a page that is not a runtime page, or an address its pattern does not match', () => {
  open(undefined, '/products/42')
  expect(useParams()).toEqual({})
  for (const path of ['/products', '/products/1/extra', '/Products/1', '/product/1']) {
    open('/products/:id', path)
    expect(useParams(), path).toEqual({})
  }
})

it('is empty on the server, where there is no document', () => {
  open('/products/:id', '/products/42')
  vi.stubGlobal('document', undefined)
  expect(useParams()).toEqual({})
})
