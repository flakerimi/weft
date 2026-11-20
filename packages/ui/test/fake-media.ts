/** A controllable stand-in for window.matchMedia. */
export function fakeMedia(initial: Record<string, boolean>) {
  const state = { ...initial }
  const lists = new Map<string, { matches: boolean; listeners: Set<(e: { matches: boolean }) => void> }>()
  const mq = (query: string) => {
    let entry = lists.get(query)
    if (!entry) {
      entry = { matches: state[query] ?? false, listeners: new Set() }
      lists.set(query, entry)
    }
    const e = entry
    return {
      get matches() { return e.matches },
      media: query,
      addEventListener: (_: 'change', fn: (ev: { matches: boolean }) => void) => e.listeners.add(fn),
      removeEventListener: (_: 'change', fn: (ev: { matches: boolean }) => void) => e.listeners.delete(fn),
    } as unknown as MediaQueryList
  }
  return {
    mq,
    set(query: string, matches: boolean) {
      state[query] = matches
      const e = lists.get(query)
      if (!e) return
      e.matches = matches
      e.listeners.forEach((fn) => fn({ matches }))
    },
    listenerCount: (query: string) => lists.get(query)?.listeners.size ?? 0,
  }
}
