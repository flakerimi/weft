# Showcase themes and a theme menu on weft.base.al

Written 2026-11-27. The owner asked for "showcased themes, so we dont remove
but just change, this way we show how themes works too", as "a drop down, a
theme like synthwave, monokai". It follows a trial of Quicksand in place of
Bricolage Grotesque (branch `quicksand`, not merged): rather than change the
site's font, the site lets a visitor pick a theme and see every page in it.

## What it is

- A **Theme** dropdown in the site header. Picking a theme switches the whole
  site, every page, at run time, and the choice is kept for the next page and
  the next visit. Weft stays the default, as built.
- **Three new themes** in `@weft/themes`, made the way the existing three are,
  so they are real themes an app can build with (`--theme synthwave`), not
  site decorations:
  - `synthwave`: neon on a deep violet ground (light: the same hues on a pale
    lilac ground), Space Grotesk for text.
  - `editor`: a warm charcoal code-editor look with candy accents (pink,
    lime, amber, violet), in the spirit of Monokai, monospace everywhere
    (JetBrains Mono for text too). Light: the same accents on a warm paper
    ground.
  - `sandstorm`: the owner's: warm desert sand, dune ochre and a dusty
    terracotta, soft and rounded (larger radii), Quicksand for text. Dark: the
    same hues at dusk, on a deep umber ground.
- The dropdown lists the six: Weft, Neutral, Lisa, Synthwave, Editor, Sandstorm.

The palettes are original. "Monokai" and "Synthwave '84" are existing editor
themes; ours take the style, not their colours or names (`synthwave` is the
genre's name).

## Constraints

- **Contrast:** every new theme passes `@weft/themes`' own checks in light and
  dark (text, muted text on every ground, each tone's text and fill, focus).
  A palette that cannot pass is changed, never the check.
- **Self-hosted fonts only:** every font comes from an `@fontsource-variable`
  package (latest versions: run `npm view`), Latin subset, served by the site;
  the tests that forbid off-site loads keep passing.
- **Page budgets are never raised.** The default theme and its fonts stay what
  every page inlines and preloads. Another theme's CSS and fonts are separate
  files, fetched only when it is picked. Where the menu or its pre-paint code
  does not fit a page's budget, room is made in that page's own text; if the
  `/skills/*` pages cannot take it, they may go without the menu (they are for
  agents) and keep the site's default.
- **No flash:** a kept choice is applied before the first paint, with a
  pre-paint snippet (`usePrePaint`), as the width measure and stored
  preferences are.
- Vapor only, `--weft-*` tokens with fallbacks, Weft components (a `Select` or
  `DropdownMenu` from `@weft/ui`, labelled "Theme"), standard library only in
  Go. Docs describe the code as it is.

## How

1. **The themes:** `packages/themes/src/{synthwave,editor,sandstorm}.ts`, exported
   and built to `dist/<name>.css` like the others; `weft-build-pages --theme`
   and `weft create`'s theme choice accept them wherever the other names are
   accepted. Tests as for `weft.ts` (`weft.test.ts`), and the contrast suite
   over all six.
2. **Scoped theme files for the site:** generalise the gallery's `scopeTheme`
   (`site-src/gallery/plugin.ts`) to take any selector, and build one file per
   non-default theme scoped to `:root[data-site-theme="<name>"]`, light and
   dark following `data-weft-scheme` and the system as the theme CSS does,
   with its `@font-face` rules. The files and their fonts are hashed and
   cached like the site's other assets.
3. **The menu:** an island in `SiteHeader.vue`. On change it sets
   `document.documentElement.dataset.siteTheme`, adds the theme's stylesheet
   once, and keeps the choice in `localStorage` (`weft:site-theme`); "Weft"
   removes both. Without JavaScript the pages are Weft and the menu does
   nothing.
4. **Before paint:** a pre-paint snippet reads the kept choice and applies the
   attribute and the stylesheet link before the page is painted. The CSP docs
   already say how to collect a pre-paint hash.
5. **Docs:** `site-src/content/docs/guides/themes.md` lists the six themes and
   says they can be tried from the header's Theme menu; the `weft-theme`
   skill's table; `CHANGELOG.md` under "Unreleased".

## Not in this change

- A run-time theme switch for apps in `@weft/ui`: apps still pick their theme
  at build time. The site's switch is the site's own code.
- The Quicksand-only trial on `quicksand`: superseded by `sandstorm`; the branch is
  kept, not merged.
