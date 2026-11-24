# The site's icon

Three files in `site-src/public/`, which the page build names in every page's `<head>` by itself (see "Icons" in `site-src/content/docs/guides/pages-and-islands.md`):

| File | What it is | Bytes |
| --- | --- | --- |
| `favicon.svg` | The mark, drawn by hand: one `rect` and two `path`s | 553 |
| `favicon.ico` | 32 × 32 and 16 × 16, rendered from the SVG, each stored as PNG data | 1,082 |
| `apple-touch-icon.png` | 180 × 180, full-bleed and opaque (iOS rounds the corners itself) | 4,156 |

## The mark

A plain weave seen straight on, a flat and simplified echo of the loom in `docs/design/hero-2026-11-23/loom-source.png`: not a copy of it, and nothing of the owner's wordmark.

- A 64-unit square, corner radius 14 (22 % of the side), filled with the site's dark ground `#0F1530`.
- Three vertical threads in indigo and three horizontal threads in saffron `#E2A11B`, 8 units wide with 4-unit gaps, running from 8 to 56 with rounded ends. Each horizontal thread goes over, under, over, the next under, over, under. Where a thread goes under, it stops 2 units either side of the thread over it, so a sliver of ground shows on both sides; no two shapes overlap, so nothing draws a seam at any size.
- The indigo is `#4C64CF`, the site's `#2B3D8F` lightened at the same hue. `#2B3D8F` on the ground is 1.85 : 1 and all but vanished at 16 and 32 px; `#4C64CF` is 3.44 : 1 and reads at both. Saffron on the ground is 7.99 : 1.
- The widths and gaps are whole pixels at 16 and 32 px (2 and 1 px at 16), so the 16 px image is a clean saffron-and-blue grid with the over and under still visible; three by three was kept, as it is not mud at 16 px.

## Remaking the files

With rsvg-convert 2.62.3 and ffmpeg 8.1, from the repository's root, these make the same bytes:

```sh
p=site-src/public
tmp=$(mktemp -d)
rsvg-convert -w 32 -h 32 $p/favicon.svg -o $tmp/32.png
rsvg-convert -w 16 -h 16 $p/favicon.svg -o $tmp/16.png
ffmpeg -loglevel error -y -i $tmp/32.png -i $tmp/16.png -map 0 -map 1 -c png $p/favicon.ico
rsvg-convert -w 180 -h 180 -b '#0F1530' $p/favicon.svg -o $tmp/180.png
ffmpeg -loglevel error -y -i $tmp/180.png -vf format=rgb24 $p/apple-touch-icon.png
```

ffmpeg's ICO muxer writes one image per stream, as PNG data with `-c png`. The touch icon is drawn on the ground colour (`-b`), which fills the rounded square's corners, and `format=rgb24` drops the alpha channel, so it is opaque.

`site-src/favicon_test.go` checks the files: the SVG is well-formed, under 2 KB, and has only `svg`, `rect` and `path`, with no picture, script, filter or link; the ICO's header lists a 32 and a 16 px image, each PNG data of that size; the touch icon is 180 × 180 and opaque.
