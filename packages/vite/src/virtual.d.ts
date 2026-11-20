declare module 'virtual:weft/islands' {
  /** Island name (the component's SFC `__name`) → a loader for its chunk; served by `weft()`. */
  const islands: Record<string, () => Promise<import('vue').Component>>
  export default islands
}
