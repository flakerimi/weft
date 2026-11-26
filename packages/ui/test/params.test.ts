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
  // Not UTF-8: the escapes that do not decode stay as they are, and the rest decodes.
  open('/products/:id', '/products/%FF')
  expect(useParams()).toEqual({ id: '%FF' })
  open('/products/:id', '/products/caf%C3%A9%FF%20x')
  expect(useParams()).toEqual({ id: 'café%FF x' })
  open('/products/:id', '/products/%C3%A9%C3')
  expect(useParams()).toEqual({ id: 'é%C3' })
  // A static segment is compared as the server compares it, decoded.
  open('/café/:id', '/caf%C3%A9/1')
  expect(useParams()).toEqual({ id: '1' })
})

it('splits the address as the server does: decoded first, then cleaned', () => {
  // The server decodes %2F before it matches, so this is the edit page of product "a".
  open('/products/:id/edit', '/products/a%2Fedit')
  expect(useParams()).toEqual({ id: 'a' })
  open('/products/:id', '/products/a%2Fedit')
  expect(useParams()).toEqual({})
  // An encoded slash splits the path where the server splits it even when another escape is not UTF-8.
  open('/products/:id/edit', '/products/%FF%2Fedit')
  expect(useParams()).toEqual({ id: '%FF' })
  open('/products/:id', '/products/%FF%2Fedit')
  expect(useParams()).toEqual({})
  // A trailing slash and a doubled one, as path.Clean drops them.
  open('/products/:id', '/products/42/')
  expect(useParams()).toEqual({ id: '42' })
  open('/products/:id', '/products//42')
  expect(useParams()).toEqual({ id: '42' })
  // A decoded .. takes a segment back, as path.Clean takes it.
  open('/products/:id', '/products/x/%2E%2E/42')
  expect(useParams()).toEqual({ id: '42' })
  open('/docs/:...rest', '/docs//a///b/')
  expect(useParams()).toEqual({ rest: 'a/b' })
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

it('in development, says once that a page built ahead has no params in the browser', async () => {
  vi.resetModules() // a copy that has not said it yet
  const fresh = (await import('../src/head/useParams.ts')).useParams
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  open(undefined, '/about')
  expect(fresh()).toEqual({})
  expect(fresh()).toEqual({})
  expect(warn).toHaveBeenCalledTimes(1)
  expect(String(warn.mock.calls[0]![0])).toContain('runtime = true')
})
