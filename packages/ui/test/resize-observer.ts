type Callback = (entries: { contentRect: { width: number }; borderBoxSize: { inlineSize: number; blockSize: number }[]; target: Element }[]) => void

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
    /** Report `el` at a border box `width` px wide, its content box `content` px (the same, unless padded). */
    report(el: Element, width: number, content = width) {
      live.forEach((o) => o.targets.has(el) && o.cb([{ contentRect: { width: content }, borderBoxSize: [{ inlineSize: width, blockSize: 0 }], target: el }]))
    },
    liveCount: () => live.size,
    restore() { globalThis.ResizeObserver = original },
  }
}
