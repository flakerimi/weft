---
title: Themes
description: Pick a theme, follow light and dark, and change colours, type and spacing with tokens.
order: 28
---

On this page you pick a theme for your site, switch between light and dark, and change the look with tokens. A theme is one stylesheet of CSS custom properties. Weft's components read nothing else.

## The themes

`@weft/themes` has three:

| Theme | What it is |
| --- | --- |
| `neutral` | The default: greys and a blue primary, Inter for text |
| `lisa` | Generated from the LisaOS token contract: olive, Rubik |
| `weft` | This site's: indigo on a cool ground, saffron, Bricolage Grotesque |

Pick one with the page build's `--theme` flag:

```sh
weft-build-pages --root . --out dist --theme weft
```

The build inlines the theme first on every page, before Weft's base styles and the components' CSS. A theme names its fonts; it does not ship them. Load them yourself with `@font-face`, as this site does. A file your CSS names with `url()`, a package's font included, is copied into the build under a name made from its bytes. A page preloads a `.woff2` face of each family its CSS uses when the face has no `unicode-range` or its range covers Basic Latin; preload others with `--preload-font`. A `url()` that names no file, or a file outside the site and the packages it uses, fails the build.

## The tokens

Every value a component draws with is a `--weft-*` custom property:

| Tokens | What they set |
| --- | --- |
| `--weft-color-bg`, `-surface`, `-surface-raised`, `-border` | The page, cards and raised panels, and lines |
| `--weft-color-text`, `-text-muted` | Text |
| `--weft-color-primary`, `-danger`, `-warning`, `-success`, `-info` | The four tones, info, and `-focus` for focus rings |
| `--weft-color-primary-soft` | A pale ground of the primary, under selected table rows and tree items |
| `--weft-space-1`, `-2`, `-3`, `-4`, `-6`, `-8` (no 5 or 7), `--weft-radius-control`, `-card`, `-panel` | Spacing and corners |
| `--weft-control-sm`, `-md`, `-lg` | The drawn height of controls |
| `--weft-shadow-control`, `-card`, `-popup`, `-dialog` | Shadows: the small raised parts of controls (a switch's or a slider's thumb, a selected pill tab); cards and other resting surfaces (`none` in every theme); menus, popovers, select lists and toasts (a tooltip draws none); dialogs, sheets and the command palette |
| `--weft-border-width`, `--weft-border-width-strong`, `--weft-focus-width` | Line widths: every hairline (1px), the emphasised lines such as a tab's indicator (2px), and the focus ring (2px). `0px` removes the hairlines |
| `--weft-font-sans`, `-mono`, `--weft-weight-body`, `-heading`, `-title` | Type |
| `--weft-duration-*`, `--weft-ease-*` | Motion |
| `--weft-hit-min`, `--weft-text-base`, `--weft-page-max` | Set by input and surface; see [the adaptive model](adaptive.md) |

For each tone the theme also has `-fg` (text on the solid fill), `-text` (the tone as text on the page), and `-hover` and `-active` fills. They are worked out when the theme is built, so that every pair reaches WCAG AA contrast, 4.5:1. A theme whose colours cannot reach it fails to build.

## Light and dark

A theme has a light and a dark set. The page follows the system's setting through `prefers-color-scheme`. To force one, set `data-weft-scheme` on an element; everything inside follows:

```html
<html data-weft-scheme="dark">
```

Dialogs, sheets and menus that open outside that element take its scheme with them.

## Change the tokens

Set tokens in a stylesheet your pages import. It comes after the theme, and the theme's scheme selectors add no specificity, so a rule on `:root, [data-weft-scheme]` wins in light and in dark:

```css
:root, [data-weft-scheme] {
  --weft-color-primary: #0B6E4F;
  --weft-radius-control: 4px;
}
```

The rule names `[data-weft-scheme]` as well as `:root` because an element with its own `data-weft-scheme` sets all of the theme's tokens again on itself, for everything inside it. A `:root` rule alone would not reach inside that element. Dialogs, sheets and menus opened from inside it carry its `data-weft-scheme` with them, so the same rule reaches them too.

To change a token in one scheme only, set it on `[data-weft-scheme="dark"]` and in a `prefers-color-scheme: dark` media query on `:root:not([data-weft-scheme="light"])`, as the theme does.

When you change a colour this way, the derived `-fg`, `-text`, `-hover` and `-active` colours are not worked out again. Set them too, and check their contrast.

## Tailwind

Weft has no Tailwind inside. If your own components use Tailwind, import `@weft/themes/tailwind.css` after the theme. It maps Tailwind's colour, radius, shadow and font names (`bg-surface`, `text-primary`, `shadow-popup`) to the same tokens.
