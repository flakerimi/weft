// What Link and Button (with `to`) share about an address.

/** An address with a scheme and a host (`https://…`, `//…`): another site, as far as the markup can tell. */
export function isExternal(address: string | undefined): boolean {
  return /^([a-z][a-z\d+.-]*:)?\/\//i.test(address ?? '')
}

/** The rel for a link: the one given, with `noopener` added once for an external link; undefined when empty. */
export function relFor(rel: string | undefined, external: boolean): string | undefined {
  const words = (rel ?? '').split(/\s+/).filter(Boolean)
  if (external && !words.includes('noopener')) words.push('noopener')
  return words.length ? words.join(' ') : undefined
}
