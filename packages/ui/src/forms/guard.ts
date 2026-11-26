/**
 * The pre-paint script a Form puts in the page's head (usePrePaint), which
 * cancels a Weft form's submit until its island is live, so a form drawn by
 * the server never sends itself as a plain POST. In development it also says,
 * once, that a form in no island does nothing: nothing ever makes it live.
 */
export function formGuard(dev: boolean): string {
  return dev ? devGuard : guard
}

const guard =
  'window.addEventListener("submit",function(e){var t=e.target;if(t&&t.classList&&t.classList.contains("weft-form"))e.preventDefault()},true)'

const devGuard =
  'window.addEventListener("submit",function(e){var t=e.target;if(t&&t.classList&&t.classList.contains("weft-form")){e.preventDefault();' +
  'if(!t.closest("weft-island")&&!window.__weftFormAlone){window.__weftFormAlone=1;' +
  'console.warn("Form: this form is in no island, so its submit does nothing; put it in a component and place that on the page with <Island>.")}}},true)'
