import { afterEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useScrollLock } from '../../src/headless/useScrollLock.ts'
import { reset } from './dom.ts'

afterEach(reset)

const overflow = () => document.documentElement.style.overflow

describe('useScrollLock', () => {
  it('locks page scroll while active and puts the page back as it was', async () => {
    document.documentElement.style.overflow = 'clip'
    const on = ref(false)
    const scope = effectScope()
    scope.run(() => useScrollLock(on))
    expect(overflow()).toBe('clip')
    on.value = true
    await nextTick()
    expect(overflow()).toBe('hidden')
    on.value = false
    await nextTick()
    expect(overflow()).toBe('clip')
    scope.stop()
  })

  it('is counted: the page scrolls again only when the last lock is released', async () => {
    const a = ref(true)
    const b = ref(true)
    const c = ref(true)
    const scopes = [a, b, c].map((r) => {
      const s = effectScope()
      s.run(() => useScrollLock(r))
      return s
    })
    expect(overflow()).toBe('hidden')
    a.value = false
    await nextTick()
    expect(overflow()).toBe('hidden')
    scopes[1]!.stop() // b goes away while still active
    expect(overflow()).toBe('hidden')
    c.value = false
    await nextTick()
    expect(overflow()).toBe('')
    scopes.forEach((s) => s.stop())
  })

  it('releases its lock when its scope ends while active', () => {
    const scope = effectScope()
    scope.run(() => useScrollLock(ref(true)))
    expect(overflow()).toBe('hidden')
    scope.stop()
    expect(overflow()).toBe('')
  })

  it('counts one active instance once, however often it is set', async () => {
    const a = ref(true)
    const b = ref(true)
    const sa = effectScope()
    const sb = effectScope()
    sa.run(() => useScrollLock(a))
    sb.run(() => useScrollLock(b))
    a.value = false
    await nextTick()
    a.value = true
    await nextTick()
    a.value = false
    await nextTick()
    b.value = false
    await nextTick()
    expect(overflow()).toBe('')
    sa.stop()
    sb.stop()
  })
})
