import { describe, expect, it } from 'vitest'
import { effectScope, nextTick, reactive, watch } from 'vue'
import { useControllable } from '../../src/headless/useControllable.ts'

function setup<T>(initial: T | undefined, fallback: T | (() => T)) {
  const props = reactive({ modelValue: initial }) as { modelValue: T | undefined }
  const emitted: [string, T][] = []
  const emit = (event: 'update:modelValue', value: T) => void emitted.push([event, value])
  const scope = effectScope()
  const model = scope.run(() => useControllable(props, 'modelValue', emit, fallback))!
  return { props, emitted, model, scope }
}

describe('useControllable', () => {
  it('without a bound value keeps its own, starting from the default, and reports changes', () => {
    const { model, emitted, scope } = setup<string>(undefined, 'a')
    expect(model.value).toBe('a')
    model.value = 'b'
    expect(model.value).toBe('b')
    expect(emitted).toEqual([['update:modelValue', 'b']])
    scope.stop()
  })

  it('takes a default from a function, called once', () => {
    let calls = 0
    const { model, scope } = setup<string[]>(undefined, () => (calls++, []))
    model.value = [...model.value, 'x']
    expect(model.value).toEqual(['x'])
    expect(calls).toBe(1)
    scope.stop()
  })

  it('with a bound value shows it and asks the parent to change it', () => {
    const { props, model, emitted, scope } = setup<string>('p', 'a')
    expect(model.value).toBe('p')
    model.value = 'q'
    expect(emitted).toEqual([['update:modelValue', 'q']])
    expect(model.value).toBe('p') // until the parent agrees
    props.modelValue = 'q'
    expect(model.value).toBe('q')
    scope.stop()
  })

  it('does not report setting the value it already has', () => {
    const { model, emitted, scope } = setup<string>(undefined, 'a')
    model.value = 'a'
    expect(emitted).toEqual([])
    scope.stop()
  })

  it('is reactive for watchers either way', async () => {
    const { props, model, scope } = setup<number>(undefined, 1)
    const seen: number[] = []
    scope.run(() => watch(model, (v) => seen.push(v)))
    model.value = 2
    await nextTick()
    props.modelValue = 5
    await nextTick()
    expect(seen).toEqual([2, 5])
    scope.stop()
  })

  it('keeps its own value when the parent stops binding one', () => {
    const { props, model, scope } = setup<string>('p', 'a')
    props.modelValue = undefined
    expect(model.value).toBe('p')
    scope.stop()
  })
})
