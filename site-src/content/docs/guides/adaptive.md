---
title: The adaptive model
description: How Weft's components adapt to touch or pointer, landing or app, and the width they have.
order: 29
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
- **Table.** In a narrow container each row becomes a card of label and value pairs.

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
- components whose layout changes with the viewport pick it with CSS media queries until they mount.

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
