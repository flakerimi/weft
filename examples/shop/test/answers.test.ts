// The shop's pure rules, run with Node's own test runner: pnpm test.
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { ApiError, type User } from '../web/.weft-types/api.ts'
import { formProblem, nextPath, productId, sessionFrom, unreachable } from '../web/answers.ts'

const origin = 'http://shop.test:8080'

test('nextPath follows only a path on this site', () => {
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
  for (const next of refused) assert.equal(nextPath(next, origin), home, JSON.stringify(next))

  for (const next of ['/', '/products', '/products/3/edit?x=1#y', '/products?q=a%20b', '/login?next=%2F%2Fevil.example']) {
    assert.equal(nextPath(next, origin), next, next)
  }
})

test('every path nextPath gives stays on the site when followed', () => {
  const tries = ['/.//x', '/a/../b', '/%2e/x', '/x?y=//z', '/x#//y', '/a/./b/../../c', '/%252e//x']
  for (const next of tries) {
    const path = nextPath(next, origin)
    assert.ok(path.startsWith('/') && !path.startsWith('//'), `${next} gave ${path}`)
    assert.equal(new URL(path, origin).origin, origin, `${next} gave ${path}`)
  }
})

test('formProblem: field errors, a message, and a server that cannot be reached', () => {
  assert.deepEqual(formProblem(new ApiError(422, 'invalid', '1 field has a problem.', { name: 'This is already taken.' })), {
    fields: { name: 'This is already taken.' },
    message: '',
  })
  assert.deepEqual(formProblem(new ApiError(403, 'forbidden', 'This setup link is not valid.')), {
    fields: {},
    message: 'This setup link is not valid.',
  })
  // fetch's own words ("Failed to fetch", "Load failed") are not for a person.
  assert.deepEqual(formProblem(new ApiError(0, 'network', 'Failed to fetch', {}, new TypeError('Failed to fetch'))), {
    fields: {},
    message: unreachable,
  })
  assert.deepEqual(formProblem(new Error('boom')), { fields: {}, message: unreachable })
})

const owner: User = { id: 1, created_at: '', updated_at: '', email: 'owner@example.com', role: 'owner' }

test('sessionFrom: a user, no session, or a server that cannot be reached', async () => {
  assert.deepEqual(await sessionFrom(async () => owner), { user: owner, problem: '' })
  assert.deepEqual(
    await sessionFrom(async () => {
      throw new ApiError(401, 'unauthorized', 'Sign in to do that.')
    }),
    { user: null, problem: '' },
  )
  assert.deepEqual(
    await sessionFrom(async () => {
      throw new ApiError(0, 'network', 'Load failed')
    }),
    { user: null, problem: unreachable },
  )
  assert.deepEqual(
    await sessionFrom(async () => {
      throw new ApiError(503, 'unavailable', 'The database is busy; try again in a moment.')
    }),
    { user: null, problem: 'The database is busy; try again in a moment.' },
  )
})

test('productId takes only an id as the server writes it', () => {
  assert.equal(productId('42'), 42)
  assert.equal(productId('9007199254740991'), 9007199254740991)
  for (const raw of ['', ' ', '0', '007', '-1', '+4', '4.0', '4e0', '0x2a', '1_000', 'NaN', 'abc', '9007199254740992', '%20', undefined]) {
    assert.equal(productId(raw), null, JSON.stringify(raw))
  }
})
