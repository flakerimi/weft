// The benchmark's one image, for both sites.
//
// bench/image/hero.jpg is the source both sites start from. It is made here
// once, if it is missing: a seeded, photo-like landscape (fractal noise and
// film grain, 2400 x 1350), so it compresses like a photograph and not like a
// flat graphic, and anyone can make the same file again.
//
// Astro sizes it with its own <Image>. Weft has no <Image> yet, so this script
// makes the WebP the Weft site serves (bench/weft/public/hero.webp) the way
// Astro's sharp service does for <Image width={1600}>: rotate by EXIF, resize
// to 1600 px wide without enlarging, WebP at sharp's default quality.
import { existsSync, mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import sharp from 'sharp'

const bench = join(dirname(fileURLToPath(import.meta.url)), '..')
const source = join(bench, 'image', 'hero.jpg')
const weftImage = join(bench, 'weft', 'public', 'hero.webp')
const WIDTH = 1600

// mulberry32: a small seeded generator, so the source is the same every time.
function random(seed) {
  return () => {
    seed = (seed + 0x6d2b79f5) | 0
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function makeNoise(rand, size) {
  const grid = new Float32Array(size * size).map(() => rand())
  const smooth = (t) => t * t * (3 - 2 * t)
  return (x, y) => {
    const xi = Math.floor(x), yi = Math.floor(y)
    const xf = smooth(x - xi), yf = smooth(y - yi)
    const at = (i, j) => grid[(((j % size) + size) % size) * size + (((i % size) + size) % size)]
    const top = at(xi, yi) * (1 - xf) + at(xi + 1, yi) * xf
    const bottom = at(xi, yi + 1) * (1 - xf) + at(xi + 1, yi + 1) * xf
    return top * (1 - yf) + bottom * yf
  }
}

async function makeSource() {
  const w = 2400, h = 1350
  const rand = random(20261120)
  const noise = makeNoise(rand, 256)
  const fbm = (x, y) => {
    let v = 0, amp = 0.5, f = 1
    for (let o = 0; o < 6; o++, amp /= 2, f *= 2) v += amp * noise(x * f, y * f)
    return v
  }
  const px = Buffer.alloc(w * h * 3)
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const u = x / w, v = y / h
      // Sky to ground, a ridge line of hills, clouds in the sky.
      const ridge = 0.45 + 0.18 * (fbm(u * 3, 7.3) - 0.5) * 2
      const n = fbm(u * 8, v * 8)
      let r, g, b
      if (v < ridge) {
        const c = fbm(u * 4 + 11, v * 10)
        const cloud = Math.max(0, c - 0.5) * 2.2
        r = 70 + 90 * v + 160 * cloud
        g = 120 + 70 * v + 140 * cloud
        b = 200 + 30 * v + 50 * cloud
      } else {
        const d = (v - ridge) / (1 - ridge)
        r = 40 + 70 * n + 50 * d
        g = 70 + 90 * n - 20 * d
        b = 30 + 40 * n
      }
      const grain = (rand() - 0.5) * 18
      const i = (y * w + x) * 3
      px[i] = Math.max(0, Math.min(255, r + grain))
      px[i + 1] = Math.max(0, Math.min(255, g + grain))
      px[i + 2] = Math.max(0, Math.min(255, b + grain))
    }
  }
  mkdirSync(dirname(source), { recursive: true })
  await sharp(px, { raw: { width: w, height: h, channels: 3 } }).jpeg({ quality: 90 }).toFile(source)
  process.stdout.write(`images: made the source ${source}\n`)
}

if (!existsSync(source)) await makeSource()
mkdirSync(dirname(weftImage), { recursive: true })
const out = await sharp(source).rotate().resize({ width: WIDTH, withoutEnlargement: true }).webp().toFile(weftImage)
process.stdout.write(`images: ${weftImage} is ${out.width} x ${out.height}, ${out.size} B\n`)
