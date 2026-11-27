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
`Pricing`, `Testimonials`, `Faq` and `Cta`) registers `useWidth`'s pre-paint
observer (`widthPrePaint`), which the page build wraps in `(function(){…})()`
inside `<script data-weft-prepaint>` in `<head>`:

    (function(){if(typeof ResizeObserver<"u"){var d=function(t,w){if(w>0)t.setAttribute("data-width",w<640?"narrow":w<1024?"regular":"wide")},r=window.__weftWidthRo=new ResizeObserver(function(e){for(var i=0;i<e.length;i++){var t=e[i].target;if(t.hasAttribute("data-weft-width"))r.unobserve(t);else d(t,e[i].contentRect.width)}}),s=function(){var l=document.querySelectorAll("[data-width]:not([data-weft-width])");for(var i=0;i<l.length;i++){d(l[i],l[i].getBoundingClientRect().width);r.observe(l[i])}};s();if(document.readyState==="loading"){var m=new MutationObserver(s);m.observe(document,{childList:true,subtree:true});addEventListener("DOMContentLoaded",function(){m.disconnect();s()},{once:true})}}})()

It observes `[data-width]` elements added while the page parses until
`DOMContentLoaded`; `[data-width]` markup inserted after `DOMContentLoaded`
by code outside Vue keeps its server attribute.
