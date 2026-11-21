import { describe, expect, it } from 'vitest'
import { effectScope, nextTick, reactive, watch } from 'vue'
import { useControllable } from '../../src/headless/useControllable.ts'

function setup<T>(initial: T | undefined, fallback: Exclude<T, undefined> | (() => Exclude<T, undefined>)) {
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

  it('with a bound value shows it and asks the parent to change it', async () => {
    const { props, model, emitted, scope } = setup<string>('p', 'a')
    expect(model.value).toBe('p')
    model.value = 'q'
    expect(emitted).toEqual([['update:modelValue', 'q']])
    props.modelValue = 'q'
    await nextTick()
    expect(model.value).toBe('q')
    scope.stop()
  })

  it('reads a bound write back at once, so writes in one task build on each other', () => {
    const { model, emitted, scope } = setup<number>(1, 0)
    model.value = model.value + 1
    model.value = model.value + 1
    expect(model.value).toBe(3)
    expect(emitted).toEqual([['update:modelValue', 2], ['update:modelValue', 3]])
    scope.stop()
  })

  it('shows the parent\'s value again at the next tick when the parent refuses the write', async () => {
    const { model, emitted, scope } = setup<string>('p', 'a')
    model.value = 'q'
    expect(model.value).toBe('q')
    await nextTick()
    expect(model.value).toBe('p')
    expect(emitted).toEqual([['update:modelValue', 'q']])
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
