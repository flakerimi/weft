// A Banner closed under an id stays closed on later visits: localStorage
// holds `closed` under `weft-banner:<id>`.

/** The storage key of a Banner's id. */
export const bannerKey = (id: string) => `weft-banner:${id}`

/**
 * The pre-paint snippet (usePrePaint) for a closable Banner with `id`: when
 * it was closed on an earlier visit it adds a rule hiding it, before the
 * first paint, so a reload does not flash it until the Banner mounts and
 * leaves the page. Safe inside an inline script: `<` is escaped.
 */
export function bannerSnippet(id: string): string {
  const rule = `.weft-banner[data-id=${JSON.stringify(id)}]{display:none}`
  const code = `try{if(localStorage.getItem(${JSON.stringify(bannerKey(id))})==="closed"){var s=document.createElement("style");s.textContent=${JSON.stringify(rule)};document.head.appendChild(s)}}catch(e){}`
  return code.replace(/</g, '\\u003c')
}
