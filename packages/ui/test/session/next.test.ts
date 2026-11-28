// nextPath, moved from the template's web/answers.ts with its cases: where to
// go after signing in, which must be a path on this site and nothing else.
import { describe, expect, it } from 'vitest'
import { nextPath } from '../../src/session/next.ts'

const origin = 'http://shop.test:8080'

describe('nextPath', () => {
  it('follows only a path on this site', () => {
    const home = '/'
    const refused: (string | null | undefined)[] = [
      // Dot segments that collapse into //host once parsed.
      '/.//evil.example',
      '/a/..//evil.example',
      '/%2e//evil.example',
      '/%2E%2E//evil.example',
      '/.\\/evil.example',
      '/./\\evil.example',
      '/.\\\\evil.example',
      // Another site, written plainly.
      '//evil.example',
      '/\\evil.example',
      'https://evil.example/',
      'http://shop.test:8080.evil.example/',
      'javascript:alert(1)',
      'evil.example',
      // Nothing, and control characters the parser would drop.
      '',
      null,
      undefined,
      '/\t/evil.example',
      '/\n/evil.example',
      '\n//evil.example',
      '/prod\tucts',
      '/prod\nucts',
      ' /products',
    ]
    for (const next of refused) expect(nextPath(next, origin), JSON.stringify(next)).toBe(home)

    for (const next of ['/', '/products', '/products/3/edit?x=1#y', '/products?q=a%20b', '/login?next=%2F%2Fevil.example']) {
      expect(nextPath(next, origin), next).toBe(next)
    }
  })

  it('gives only paths that stay on the site when followed', () => {
    const tries = ['/.//x', '/a/../b', '/%2e/x', '/x?y=//z', '/x#//y', '/a/./b/../../c', '/%252e//x']
    for (const next of tries) {
      const path = nextPath(next, origin)
      expect(path.startsWith('/') && !path.startsWith('//'), `${next} gave ${path}`).toBe(true)
      expect(new URL(path, origin).origin, `${next} gave ${path}`).toBe(origin)
    }
  })
})
