---
title: The adaptive model
description: How Weft's components adapt to touch or pointer, landing or app, and the width they have.
order: 33
---

On this page you learn the three things every Weft component adapts to, how it learns them, and how to set them for part of a page. You write one component; it does not have a mobile version.

## Three axes

| Axis | Values | Where it comes from |
| --- | --- | --- |
| **Input** | `pointer`, `touch` | The browser: `(pointer: coarse)` means touch. It follows changes live. |
| **Surface** | `landing`, `app` | You declare it, for the page or part of it. |
| **Width** | `narrow`, `regular`, `wide` | The component's own container: under 640 px is narrow, under 1024 px regular. |

Input is read from what the browser reports, never from a device name. Width comes from the container, not the window, so a table in a side panel lays out narrow on a big screen.

## What changes

Each component's docs say what it does on each axis. Some examples:

- **DatePicker.** On pointer it is a text field with a calendar that floats beside it. On touch the field is one button that opens the calendar in a bottom sheet, with dates 44 px a side.
- **FileUpload.** On pointer it is a drop zone with a Browse button. On touch it is one button that opens the system's chooser: camera, photos or files.
- **Tooltip.** On touch it renders nothing, since nothing hovers; the trigger keeps its name for screen readers.
- **Dialog.** Under 640 px it fills the screen.
- **Table.** In a narrow container each row becomes a card of label and value pairs, once it is hydrated.

Surface changes density, not behaviour: on `landing` text is 18 px and the page is at most 1200 px wide; on `app` text is 14 px and the page fills the width. On touch every control's hit area is at least 44 px; on pointer, 24 px.

## Set the surface

Wrap a page, or part of one, in `WeftProvider`:

```vue
<script setup vapor lang="ts">
import { Page, Section, WeftProvider } from '@weft/ui'
</script>

<template>
  <WeftProvider surface="landing">
    <Page title="Pricing">
      <Section title="Plans">Three plans, one price each.</Section>
    </Page>
  </WeftProvider>
</template>
```

`WeftProvider` takes `surface`, `input` and `locale`. Set `input` only to force one, for a kiosk, a preview or a test. It renders with `display: contents`, so it does not change the layout. Providers nest: an inner one overrides the axes it sets and inherits the rest.

## Before the browser takes over

The server cannot know the input or the width. So:

- the theme's CSS follows `(pointer: coarse)` from the first paint, and a page that never loads a script still has 44 px targets on a phone;
- components read the input as `pointer` until they have mounted, then switch, so a phone hydrates the markup the server drew;
- components that style from `[data-width]` in CSS register one small inline pre-paint script (`<script data-weft-prepaint>`, [`Grid`'s page](/components/grid) has its exact text) that sets `data-width` from each element's own width before the first paint, and on resize, without hydration;
- components whose layout follows the viewport pick it with CSS media queries until they mount.

Which components adapt to the width on a page that is never hydrated, and how:

| Component | Before hydration | How |
| --- | --- | --- |
| `Grid`, `Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq`, `Cta` | Yes, from their own box | The pre-paint script sets `data-width`; CSS lays them out from it. |
| `Stepper` (horizontal), `Timeline` (horizontal) | Yes, from their own box | The same: narrow, CSS draws them vertical. A vertical one does not depend on its width and registers no script. |
| `Tabs` (vertical) | Yes, from their own box | The same: narrow, CSS puts the tab list above the panels. Until it is hydrated the list keeps `aria-orientation="vertical"`, which assistive technology is told too; the tabs and the panel are the same, and switching tabs needs hydration anyway. A horizontal one registers no script. |
| `Header`, `AppShell`, `NavigationMenu`, `Footer` | Yes, from the viewport | They span the page: a viewport `@media` query lays them out until they mount, choosing between the forms the server drew (`Header`'s menu button or its links, say). `Footer` measures its own box once mounted, and before that too on a page that carries the pre-paint script. |
| `Pagination` | Yes, from the viewport | It draws the pages of both forms, each entry of one form only marked `data-only`, and a viewport `@media` query shows the one that fits until it mounts. |
| `Dialog` | Not needed | It draws only once open, which takes a script; open, it fills the screen under 640 px. |
| `Table` | No: only once hydrated | Its cards are other markup, not other CSS. Drawing both forms on the server would double the HTML and the work of hydrating it (each cell's slot drawn twice, selection checkboxes and sort buttons twice), so it stays a table, which scrolls sideways inside its own box, until it is hydrated in an island. |

With scripts off, the components that measure their own box fall back to their `wide` layout (or the `width` their `WeftProvider` set); those that follow the viewport still adapt, since a media query needs no script. Give `useWidth` `{ prepaint: false }` only in a component that measures the viewport, whose layout does not depend on its width (a vertical `Stepper` or `Timeline`, a horizontal `Tabs`), or that switches its markup in JavaScript.

## Read the axes yourself

`useWeft()` returns the input, the surface and the locale as refs:

```vue
<script setup vapor lang="ts">
import { useWeft } from '@weft/ui'

const weft = useWeft()
</script>

<template>
  <p>{{ weft.input.value === 'touch' ? 'Tap' : 'Click' }} a day to book it.</p>
</template>
```

`useWidth(element)` returns the width class of an element, measured as it resizes:

```vue
<script setup vapor lang="ts">
import { useTemplateRef } from 'vue'
import { useWidth } from '@weft/ui'

const box = useTemplateRef<HTMLElement>('box')
const width = useWidth(box)
</script>

<template>
  <div ref="box" :data-width="width">This box is {{ width }}.</div>
</template>
```

Both are reactive in the browser, so use them in islands. On the server, input is `pointer` and width is `wide`.
