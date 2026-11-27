---
group: layout
purpose: Lays its children out in equal columns, as many as its own width allows.
use: For tiles, cards and stat blocks that should reflow from one column to several as the space grows.
avoid: For a single column or row, use Stack; for tabular data, use a table.
adapts:
  input: Nothing changes.
  surface: Nothing changes.
  width: Measured from the grid's own box (useWidth, before paint through its pre-paint observer and on resize), not the window, so a grid in a side panel or static page stays narrow on a large screen; by default 1 column narrow (under 640 px), 2 regular (under 1024 px) and 3 wide.
---
`cols` is a fixed number or one per width class, `{ narrow, regular, wide }`.
A width left out falls back to the nearest smaller one given, else to the
default for it.

On the server, `Grid` (and the landing blocks `Hero`, `FeatureGrid`,
`Pricing`, `Testimonials`, `Faq` and `Cta`, a horizontal `Stepper` or
`Timeline`, and a vertical `Tabs`) registers `useWidth`'s pre-paint
observer (`widthPrePaint`), which the page build wraps in `(function(){…})()`
inside `<script data-weft-prepaint>` in `<head>` (its SHA-256 CSP hash when
alone is `'sha256-iNtcuTghwj4FD4cClz4HdWeY2ACaeNPFjXtP7CUdMP0='`):

    (function(){if(typeof ResizeObserver<"u"){var o=new Set,d=function(t,w){if(w>0)t.setAttribute("data-width",w<640?"narrow":w<1024?"regular":"wide")},r=window.__weftWidthRo=new ResizeObserver(function(e){for(var i=0;i<e.length;i++){var t=e[i].target;if(t.hasAttribute("data-weft-width"))r.unobserve(t);else d(t,b(e[i]))}}),b=function(x){var z=x.borderBoxSize;return z?(z[0]||z).inlineSize:x.target.getBoundingClientRect().width},s=function(){var l=document.querySelectorAll("[data-width]:not([data-weft-width])"),n=[],w=[],i;for(i=0;i<l.length;i++)if(!o.has(l[i])){o.add(l[i]);n.push(l[i]);w.push(l[i].getBoundingClientRect().width)}for(i=0;i<n.length;i++){d(n[i],w[i]);r.observe(n[i],{box:"border-box"})}};s();if(document.readyState==="loading"){var m=new MutationObserver(s);m.observe(document,{childList:true,subtree:true});addEventListener("DOMContentLoaded",function(){m.disconnect();s()},{once:true})}}})()

It observes `[data-width]` elements added while the page parses until
`DOMContentLoaded`; `[data-width]` markup inserted after `DOMContentLoaded`
by code outside Vue keeps its server attribute. A page that renders other
pre-paint components too (`Form`, `AppShell` unless `persist` is `false`, or a
closable `Banner` with an `id`) joins their wrapped snippets into the same
script with `;\n` in registration order (embedding any `persist` key or Banner
`id`, with the width snippet included at most once); a strict
Content-Security-Policy needs one `'sha256-…'` hash per distinct
`<script data-weft-prepaint>` (and for `<script type="speculationrules">`) in
`dist/`, re-collected after each Weft upgrade (see `Form`'s docs for the
command that lists them).
