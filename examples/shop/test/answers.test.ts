// The shop's pure rules, run with Node's own test runner: pnpm test.
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { ApiError } from '../web/.weft-types/api.ts'
import { formProblem, pageFailed, productId, unreachable } from '../web/answers.ts'

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
})

test("formProblem: a bug in the page's own code is logged, and is not a server out of reach", (t) => {
  const logged = t.mock.method(console, 'error', () => {})
  const bug = new TypeError("Cannot read properties of undefined (reading 'id')")
  assert.deepEqual(formProblem(bug), { fields: {}, message: pageFailed })
  assert.equal(logged.mock.callCount(), 1)
  assert.equal(logged.mock.calls[0]!.arguments.at(-1), bug)
  // An ApiError is the server's answer, or none, and is not logged.
  formProblem(new ApiError(0, 'network', 'Failed to fetch'))
  formProblem(new ApiError(500, 'internal', 'Something went wrong on the server; its log has the details.'))
  assert.equal(logged.mock.callCount(), 1)
})

test('productId takes only an id as the server writes it', () => {
  assert.equal(productId('42'), 42)
  assert.equal(productId('9007199254740991'), 9007199254740991)
  for (const raw of ['', ' ', '0', '007', '-1', '+4', '4.0', '4e0', '0x2a', '1_000', 'NaN', 'abc', '9007199254740992', '%20', undefined]) {
    assert.equal(productId(raw), null, JSON.stringify(raw))
  }
})
