---
title: Themes
description: Pick a theme, follow light and dark, and change colours, type and spacing with tokens.
order: 23
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

The build inlines the theme first on every page, before Weft's base styles and the components' CSS. A theme names its fonts; it does not ship them. Load them yourself with `@font-face`, as this site does. A file your CSS names with `url()`, a package's font included, is copied into the build under a name made from its bytes, and each page preloads the `.woff2` fonts its CSS names, so declare only the subsets you use.

## The tokens

Every value a component draws with is a `--weft-*` custom property:

| Tokens | What they set |
| --- | --- |
| `--weft-color-bg`, `-surface`, `-surface-raised`, `-border` | The page, cards and raised panels, and lines |
| `--weft-color-text`, `-text-muted` | Text |
| `--weft-color-primary`, `-danger`, `-warning`, `-success`, `-info` | The four tones, info, and `-focus` for focus rings |
| `--weft-space-1` to `-8`, `--weft-radius-control`, `-card`, `-panel` | Spacing and corners |
| `--weft-control-sm`, `-md`, `-lg` | The drawn height of controls |
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

Set tokens in a stylesheet your pages import. It comes after the theme, so it wins:

```css
:root {
  --weft-color-primary: #0B6E4F;
  --weft-radius-control: 4px;
}
```

When you change a colour this way, the derived `-fg`, `-text`, `-hover` and `-active` colours are not worked out again. Set them too, and check their contrast.

## Tailwind

Weft has no Tailwind inside. If your own components use Tailwind, import `@weft/themes/tailwind.css` after the theme. It maps Tailwind's colour names (`bg-surface`, `text-primary`) to the same tokens.
