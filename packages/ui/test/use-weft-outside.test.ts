import { afterEach, expect, it } from 'vitest'
import { createVaporApp, defineVaporComponent } from 'vue'
import { useWeft } from '../src/adaptive/context.ts'
import { setMatchMedia } from '../src/adaptive/detect.ts'
import { fakeMedia } from './fake-media.ts'

afterEach(() => setMatchMedia(undefined))

// A router guard or a Pinia setup store runs inside app.runWithContext with no
// component: there is nothing to hydrate there, so it must read the live input.
it('reads the live input inside app.runWithContext, outside any component', () => {
  setMatchMedia(fakeMedia({ '(pointer: coarse)': true }).mq)
  const app = createVaporApp(defineVaporComponent({ setup: () => [] }))
  const input = app.runWithContext(() => useWeft().input)
  expect(input.value).toBe('touch')
})
