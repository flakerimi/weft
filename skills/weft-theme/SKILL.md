---
name: weft-theme
description: Use when changing how a Weft site or app looks - picking the neutral, lisa or weft theme with --theme, following or forcing light and dark with data-weft-scheme, overriding --weft-* tokens for a brand, self-hosting fonts, mapping Tailwind to the tokens, or adding a new theme to @weft/themes with its contrast checks.
---

# Theming Weft

A theme is one stylesheet of `--weft-*` custom properties, with a light set and a dark set. Weft's components read nothing else. A brand is a few token overrides, never a fork of a component. The page build inlines the theme first on every page, before Weft's base styles and the components' CSS.

## Pick a theme

| Theme | What it is |
| --- | --- |
| `neutral` | The default: greys, a blue primary, Inter |
| `lisa` | Generated from the LisaOS token contract: olive, Rubik |
| `weft` | weft.base.al's: indigo on a cool ground, saffron, Bricolage Grotesque |

```sh
weft-build-pages --root . --out dist --theme weft
```

Put the flag in the site's `package.json` build script. Any other name fails the build. A theme names its fonts but does not ship them.

## Light and dark

Pages follow `prefers-color-scheme`. To force a scheme, set `data-weft-scheme="light"` or `"dark"` on any element; everything inside it follows, and dialogs, sheets and menus opened from inside take the scheme with them:

```html
<html data-weft-scheme="dark">
```

## Override tokens

Put a plain `:root` rule in a stylesheet your pages import. The theme's selectors add no specificity, so this wins in light and in dark:

```css
:root {
  --weft-color-primary: #0B6E4F;
  --weft-radius-control: 4px;
}
```

To change one scheme only, repeat the value on `[data-weft-scheme="dark"]` and inside `@media (prefers-color-scheme: dark)` on `:root:not([data-weft-scheme="light"])`, as the theme does.

The tokens:

| Tokens | What they set |
| --- | --- |
| `--weft-color-bg`, `-surface`, `-surface-raised`, `-border` | the page, cards, raised panels, lines |
| `--weft-color-text`, `-text-muted` | text |
| `--weft-color-primary`, `-danger`, `-warning`, `-success`, `-info`, `-focus` | tones and the focus ring |
| `--weft-color-<tone>-fg`, `-text`, `-hover`, `-active` | derived per tone, worked out at theme build |
| `--weft-space-1` to `-8`, `--weft-radius-control`, `-card`, `-panel` | spacing and corners |
| `--weft-control-sm`, `-md`, `-lg` | drawn control heights |
| `--weft-font-sans`, `-mono`, `--weft-weight-body`, `-heading`, `-title` | type |
| `--weft-duration-*`, `--weft-ease-*` | motion |
| `--weft-hit-min`, `--weft-text-base`, `--weft-page-max` | set by input and surface, not by you |

When you override a tone, its derived `-fg`, `-text`, `-hover` and `-active` colours are not worked out again. Set them too, and check each pair reaches 4.5:1. A theme built in `@weft/themes` is checked for you; an override is not.

## Fonts

Self-host every font, with no font CDN. Point `@font-face` at a package's file. The build then copies it under a hashed name, caches it for a year and preloads its Latin face:

```css
@font-face {
  font-family: 'Bricolage Grotesque Variable';
  font-display: swap;
  font-weight: 200 800;
  src: url('@fontsource-variable/bricolage-grotesque/files/bricolage-grotesque-latin-opsz-normal.woff2') format('woff2-variations');
}
```

To preload any other face, pass `--preload-font <file.woff2>`. A `url()` that names no file fails the build.

## Tailwind

Weft has no Tailwind inside. If your own components use Tailwind, import `@weft/themes/tailwind.css` after the theme. Tailwind's `bg-surface`, `text-primary` and the rest then read the same tokens.

## Add a theme to Weft

1. Write `packages/themes/src/<name>.ts`, exporting a `Theme`. It has:
   - `light` and `dark` sets of every colour key, each as `#RRGGBB`;
   - `font`, `radius`, `control`, `space`, `motion` and `ease`.
   Copy `weft.ts` as a start.
2. Export it from `src/index.ts`, write it in `scripts/build.ts`, and add `"./<name>.css"` to the package's `exports`.
3. Add it to `THEMES` in `packages/vite/src/build.ts`, to the usage line in `packages/vite/bin/weft-build-pages.mjs`, and to the docs' Themes and CLI pages.
4. Test it in `packages/themes/test/<name>.test.ts`, as `weft.test.ts` does: every colour key in both schemes, and focus at 3:1 on the page.
5. Run `pnpm --filter @weft/themes build`. It refuses a theme whose text, `-fg` or hover and active fills cannot reach 4.5:1, and names the colour to change.

Never edit `dist/*.css`: it is generated. Change the `lisa` theme only through Lisa's token contract: `pnpm --filter @weft/themes sync:lisa <path to tokens.json>` copies it into `src/lisa/tokens.json`.

## Do not

- Write colours, sizes or fonts into a component or page stylesheet. Use the tokens, each `var()` with a visible fallback.
- Style a component's insides through its class names to rebrand it. Change the tokens.
- Shrink a control below `--weft-hit-min` on touch.
- Add motion that ignores `prefers-reduced-motion`. Weft's own transitions stop under it.
- Pass `--theme` a file or a name of your own. It takes only the three names; a new theme goes into `@weft/themes`.

## Not yet

`weft create`, which will set up Tailwind for a new app, is planned but not built. See https://weft.base.al/docs/coming-next.
