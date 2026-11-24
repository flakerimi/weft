# The hero's weave

`weave-source.png` is the owner's picture (1024 × 1024): woven saffron and navy threads on gold pegs, on a dark navy ground. The landing page draws it into the night band's ground, behind the two frames (`.landing-hero__ground::after` in `site-src/styles/landing.css`), as a CSS `image-set()` of AVIF then WebP. The page build copies each file under a name made from its bytes, cached for a year.

Its own ground (about `#0F172F`) is the band's `#0F1530` within a shade, so the band's colour was left as it is.

## Where it is shown, and the two widths

- From 901 px up it is a square as tall as the hero, at most 1024 px (988 px at 1440 and 2560 px; 920 px between 901 and 1279 px). Every screen gets `weave-1024`: a 1× screen needs about 1000 px, and 1024 px is the whole source, so a 2× screen gets the most there is.
- At 900 px and under it covers a 708 px tall band under the phone (`cover`): 708 px square at 320 to 708 px, as wide as the screen up to 900 px. A 1× screen gets `weave-640`, a 2× or 3× screen `weave-1024`.

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

`loom-source.png` is the owner's second picture (1254 × 1254): the whole loom from above, a navy pegged board with navy warp and saffron weft, on a dark navy ground. It is shown on the landing page alone, in its closing section (`#start`), beside the install command, on a dark tile (the theme's raised surface in the dark scheme), fading into the tile where they meet; the tile's corners clip it. It is an `<img>` with an empty `alt`, `width` and `height` 480, `loading="lazy"` and `decoding="async"`, in a `<picture>` that offers AVIF first. A `.vue` page has no way yet to name a file the build hashes, so the files are in `site-src/public/`, served as they are and revalidated by their ETag rather than cached for a year.

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
