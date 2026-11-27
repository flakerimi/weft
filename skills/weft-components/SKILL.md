---
name: weft-components
description: Use before writing or changing the markup of a Weft page, island or app - choosing which @weft/ui component fits, composing layout from Page, Section, Stack and Grid, looking components up with weft catalog (or one entry's JSON on weft.base.al), and keeping to the adaptive rules for touch, pointer, surface and width.
---

# Choosing and composing Weft components

Weft has 84 Vapor components in `@weft/ui`. Each one adapts to touch or pointer, to a landing page or an app, and to the width of its own container. Pick the right one and compose it; never rebuild it. The catalogue says what each component is for, when not to use it, and what it does on each axis.

## Read the catalogue first

Never read the whole catalogue: it is hundreds of kilobytes of JSON. Ask `weft` for the pieces you need:

```sh
weft catalog                        # every component, one line each, by group
weft catalog DatePicker Select      # each one: use, avoid, adapts, props, events, slots, notes, an example
weft catalog DatePicker --examples  # with every example
weft catalog Button --json          # the entry as JSON
weft catalog --find date            # the components whose name or text has the word
```

Names match without regard to case or hyphens (`date-picker`), and an unknown name gets the closest ones. In an app it reads `.weft/catalog/components.json`, the app's version; elsewhere, the catalogue `weft` carries. Start from an example rather than inventing a usage.

Without `weft`, fetch one entry, such as `https://weft.base.al/components/date-picker.json` for `DatePicker`; `https://weft.base.al/catalog.json` has all of them. In the Weft repository the catalogue is `catalog/components.json`, generated: never edit it. Examples live in `packages/ui/examples/<Name>/`, and docs in `packages/ui/src/<group>/<Name>.docs.md`.

## Choose by `use` and `avoid`

Always read `avoid`: it names the component to use instead. The choices most often got wrong:

| You need | Use | Not |
| --- | --- | --- |
| The outer layout of a page | `Page`, once, with `Section`s inside | a `div` with margins |
| Things one after another | `Stack`, with `direction` and `gap` | wrappers with their own CSS |
| Tiles that reflow with the room | `Grid`, with `cols` such as `{ narrow: 1, regular: 2, wide: 3 }` | media queries |
| An action | `Button`; `Button` with `to` for a link that looks like one | `Link` |
| Navigation in text | `Link` | `Button` |
| A choice from a short known list | `RadioGroup` for 2-6 options, `Select` for longer lists | `Combobox` |
| A choice people search or type | `Combobox`; `InputTags` for free values | `Select` |
| On or off at once / sent with a form | `Switch` / `Checkbox` | the other one |
| A task that interrupts the page | `Dialog`; `ConfirmDialog` before something hard to undo | `Sheet` |
| A panel at an edge | `Sheet`; `Drawer` for a touch panel a finger drags away | `Dialog` |
| A message in place / site-wide / passing | `Alert` / `Banner` / `useToast().add` with one `Toaster` | the others |
| Views of one thing / sections opened one at a time | `Tabs` / `Accordion` | the other one |
| Records to sort, choose and page | `Table`; `DataTable` for a CRUD resource table with search, filters and dialogs | a grid of `Card`s |
| Nothing to show / loading failed | `EmptyState` / `ErrorState` | an `Alert` |
| An app frame | `AppShell` | `Header` and `Footer`, which are for sites |
| A landing section | `Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq`, `Cta` | a `Grid` of `Card`s |

The main value is always `v-model`, and names mean one thing everywhere:

- `size`: `sm`, `md` or `lg`;
- `tone`: `neutral`, `primary`, `danger`, `warning` or `success`;
- `variant`: `solid`, `soft`, `outline` or `ghost`, where a component has variants;
- `loading`, `disabled`, and `icon` as a Lucide name (`icon="trash"`);
- the slots `header`, `footer`, `actions`, `empty`, `leading` and `trailing`.

## Compose

A page the build renders, and every island and component in it, may use Weft's components without importing them: `@weft/vite` adds `import { … } from '@weft/ui'` for each tag the template uses. It leaves your own names alone: a name the file imports or declares, the component's own file name, lowercase tags and anything inside `v-pre`. Import anything else (`useWeft`, `useToast`, types) yourself. Writing the component imports by hand works the same:

```vue
<script setup vapor lang="ts">
import { reactive } from 'vue'
import { Button, Card, Form, FormField, Input, Page, Section, Stack, Switch, type FieldSchema } from '@weft/ui'

const state = reactive({ name: '', email: '', digest: true })
const schema: FieldSchema<keyof typeof state> = { name: { required: true }, email: { required: true, email: true } }
// Form calls it only when every field passes.
const save = (values: typeof state) => fetch('/api/settings', { method: 'PUT', body: JSON.stringify(values) })
</script>

<template>
  <Page title="Settings">
    <Section title="Profile">
      <Card>
        <Form :state="state" :schema="schema" @submit="save">
          <Stack :gap="4">
            <FormField label="Name" name="name"><Input v-model="state.name" /></FormField>
            <FormField label="Email" name="email"><Input v-model="state.email" type="email" /></FormField>
            <Switch v-model="state.digest" label="Weekly digest" />
            <Button type="submit" label="Save" />
          </Stack>
        </Form>
      </Card>
    </Section>
  </Page>
</template>
```

On a page the build renders, a form works only inside an `Island` (see `weft-site`).

## The adaptive rules

- **Never write a mobile version.** Components adapt themselves: on touch `DatePicker`, `Select` and menus open bottom sheets and `Tooltip` renders nothing; `Dialog` fills the screen under 640 px; a narrow `Table` turns each row into a card. `adapts` says what each one does.
- **Detect capability, never devices.** Read the input from `useWeft().input` (`pointer` or `touch`) and the width class of an element from `useWidth(ref)` (`narrow` under 640 px, `regular` under 1024 px, `wide`). Never sniff the user agent or test `window.innerWidth`.
- **Width is the container's, not the window's.** A component in a side panel lays out narrow on a big screen; do not fight it with viewport media queries.
- **Declare the surface.** Wrap the layout in `<WeftProvider surface="landing">` for marketing pages (18 px text, at most 1200 px wide) or `surface="app"` for tools. Surface changes density, never behaviour. Set `input` only for a kiosk, a preview or a test.
- **Touch targets are at least `--weft-hit-min`, 44 px.** Do not shrink controls below it with CSS.
- **Nothing a touch user needs goes only in a tooltip.** An icon-only `Button` still needs a name: an `aria-label`, or the `Tooltip`'s `text`.

## Do not

- Write a raw `<button>`, `<input>`, `<select>` or `<dialog>` where a Weft component exists, or hand-roll a menu, a popover or a date field.
- Import another UI library, or anything that needs the virtual DOM. Weft is Vapor only: `<script setup vapor lang="ts">`, with no Options API, no `getCurrentInstance`, no `v-memo`, no render functions and no `vaporInteropPlugin`.
- Lay pages out with `class` and custom CSS. `class` is accepted, but use `Stack`, `Grid`, `Page` and `Section`, and the `--weft-*` tokens (see `weft-theme`).

## Not yet

Planned, not built, so do not look for them (phases at https://weft.base.al/docs/coming-next):

- The charts and `Image`. Use `StatCard` today;
- `weft mcp`, which will serve the catalogue to agents over MCP; run `weft catalog` instead;
- `weft check` and `@weft/eslint-plugin`, which will flag raw HTML controls; check your markup against the rules above.
