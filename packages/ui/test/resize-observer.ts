type Callback = (entries: { contentRect: { width: number }; target: Element }[]) => void

/** A ResizeObserver stand-in whose sizes tests report by hand. */
export function installResizeObserver() {
  const live = new Set<FakeObserver>()
  class FakeObserver {
    targets = new Set<Element>()
    constructor(public cb: Callback) { live.add(this) }
    observe(el: Element) { this.targets.add(el) }
    unobserve(el: Element) { this.targets.delete(el) }
    disconnect() { this.targets.clear(); live.delete(this) }
  }
  const original = globalThis.ResizeObserver
  globalThis.ResizeObserver = FakeObserver as unknown as typeof ResizeObserver
  return {
    report(el: Element, width: number) {
      live.forEach((o) => o.targets.has(el) && o.cb([{ contentRect: { width }, target: el }]))
    },
    liveCount: () => live.size,
    restore() { globalThis.ResizeObserver = original },
  }
}
