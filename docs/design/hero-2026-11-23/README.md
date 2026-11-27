# The hero's weave

`weave-source.png` is the owner's picture (1024 × 1024): woven saffron and navy threads on gold pegs, on a dark navy ground. The landing page draws it into the night band's ground, behind the two frames (`.landing-hero__ground::after` in `site-src/styles/landing.css`), as a CSS `image-set()` of AVIF then WebP. The page build copies each file under a name made from its bytes, cached for a year.

Its own ground (about `#0F172F`) is the band's `#0F1530` within a shade, so the band's colour was left as it is.

## Where it is shown, and the two widths

- From 1280 px up, where the copy and the frames sit side by side, it is a 1024 px square at the hero's bottom end, its pegged edge rising clear above the two frames, which are drawn at 80 % and set low in their column. Every screen gets `weave-1024`: a 1× screen needs 1024 px, and 1024 px is the whole source, so a 2× screen gets the most there is.
- From 901 to 1279 px (the copy above the frames) it is a square as tall as the hero, at most 920 px, at the bottom end: `weave-1024`.
- At 900 px and under it covers a band under the phone and the gap above it (`cover`): 708 px tall from 641 to 900 px, and 700 px from 640 px down, where the gap between the copy and the phone is 64 px, so the phone follows the install command closely: the weave starts where the copy ends, fades in over that gap, its pegs rising above the phone, and is whole behind and beside it, within a phone's first screen. A 1× screen gets `weave-640`, a 2× or 3× screen `weave-1024`.

Each served file is at most 60 KB (`landingImageLimit` in `site-src/landing_test.go`).

| File | Pixels | Bytes |
| --- | --- | --- |
| `site-src/images/weave-1024.avif` | 1024 × 1024 | 55,287 |
| `site-src/images/weave-1024.webp` | 1024 × 1024 | 56,442 |
| `site-src/images/weave-640.avif` | 640 × 640 | 36,077 |
| `site-src/images/weave-640.webp` | 640 × 640 | 40,210 |

## Remaking the files

With ffmpeg 8.1, avifenc 1.3.0 (libaom 3.13.1) and cwebp 1.6.0, from the repository's root, these make the same bytes:

```sh
d=docs/design/hero-2026-11-23
tmp=$(mktemp -d)
ffmpeg -loglevel error -y -i $d/weave-source.png -vf "format=rgb24" $tmp/weave-1024.png
ffmpeg -loglevel error -y -i $d/weave-source.png -vf "scale=640:640:flags=lanczos,format=rgb24" $tmp/weave-640.png
avifenc -q 35 -s 2 -y 420 -j all $tmp/weave-1024.png site-src/images/weave-1024.avif
cwebp -quiet -q 22 -m 6 -sharp_yuv -af $tmp/weave-1024.png -o site-src/images/weave-1024.webp
avifenc -q 40 -s 2 -y 420 -j all $tmp/weave-640.png site-src/images/weave-640.avif
cwebp -quiet -q 40 -m 6 -sharp_yuv -af $tmp/weave-640.png -o site-src/images/weave-640.webp
```

The source has an alpha channel that is opaque everywhere; `format=rgb24` drops it. The 1024 px WebP is held to 60 KB at quality 22, softer than the AVIF; it is the fallback for a browser without AVIF.

# The loom at the close

`loom-source.png` is the owner's second picture (1254 × 1254): the whole loom from above, a navy pegged board with navy warp and saffron weft, on a dark navy ground. It is shown on the landing page alone, in its closing section (`#start`), beside the install command, on a dark tile (the theme's raised surface in the dark scheme). Its own ground is greyer than the tile, so a radial mask fades it into the tile all round the board. It is an `<img>` with an empty `alt`, `width` and `height` 480, `loading="lazy"` and `decoding="async"`, in a `<picture>` that offers AVIF first. A `.vue` page has no way yet to name a file the build hashes, so the files are in `site-src/public/`, served as they are and revalidated by their ETag rather than cached for a year.

It is 480 px wide beside the copy (45 % of the tile, at most 480 px) and the tile's width when stacked under 640 px (288 to 608 px): 1× screens get the 480 px files, 2× and 3× screens the 960 px ones.

| File | Pixels | Bytes |
| --- | --- | --- |
| `site-src/public/loom-480.avif` | 480 × 480 | 12,616 |
| `site-src/public/loom-960.avif` | 960 × 960 | 36,778 |
| `site-src/public/loom-480.webp` | 480 × 480 | 16,954 |
| `site-src/public/loom-960.webp` | 960 × 960 | 47,666 |

```sh
d=docs/design/hero-2026-11-23
tmp=$(mktemp -d)
ffmpeg -loglevel error -y -i $d/loom-source.png -vf "scale=960:960:flags=lanczos,format=rgb24" $tmp/loom-960.png
ffmpeg -loglevel error -y -i $d/loom-source.png -vf "scale=480:480:flags=lanczos,format=rgb24" $tmp/loom-480.png
avifenc -q 50 -s 2 -y 420 -j all $tmp/loom-960.png site-src/public/loom-960.avif
avifenc -q 50 -s 2 -y 420 -j all $tmp/loom-480.png site-src/public/loom-480.avif
cwebp -quiet -q 60 -m 6 -sharp_yuv -af $tmp/loom-960.png -o site-src/public/loom-960.webp
cwebp -quiet -q 60 -m 6 -sharp_yuv -af $tmp/loom-480.png -o site-src/public/loom-480.webp
```
