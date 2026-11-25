// Copy a built site into a folder and store brotli and gzip beside every text
// file, for `static -precompressed`: the benchmark's way of giving any
// framework's static output the serving terms Weft's build gives its own.
// Brotli is quality 11 and gzip level 9, as the Weft page build stores them.
// Files already in <out> are written over; nothing is removed from it.
//
// Usage: node precompress.mjs <site folder> <out folder>
import { cpSync, existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { extname, join } from 'node:path'
import zlib from 'node:zlib'

const [site, out] = process.argv.slice(2)
if (!site || !out) {
  process.stderr.write('usage: node precompress.mjs <site folder> <out folder>\n')
  process.exit(2)
}
if (!existsSync(site)) {
  process.stderr.write(`precompress: ${site} is missing; build the site first, as bench/run.sh does.\n`)
  process.exit(1)
}

const TEXT = new Set(['.html', '.css', '.js', '.mjs', '.json', '.svg', '.txt', '.xml', '.webmanifest', '.map'])

cpSync(site, out, { recursive: true, force: true })
let files = 0
let raw = 0
let br = 0
for (const entry of readdirSync(out, { recursive: true, withFileTypes: true })) {
  if (!entry.isFile() || !TEXT.has(extname(entry.name))) continue
  const path = join(entry.parentPath, entry.name)
  const bytes = readFileSync(path)
  const brotli = zlib.brotliCompressSync(bytes, {
    params: {
      [zlib.constants.BROTLI_PARAM_QUALITY]: 11,
      [zlib.constants.BROTLI_PARAM_MODE]: zlib.constants.BROTLI_MODE_TEXT,
      [zlib.constants.BROTLI_PARAM_SIZE_HINT]: bytes.length,
    },
  })
  writeFileSync(`${path}.br`, brotli)
  writeFileSync(`${path}.gz`, zlib.gzipSync(bytes, { level: 9 }))
  files++
  raw += bytes.length
  br += brotli.length
}
process.stdout.write(`precompress: ${out}: ${files} text files, ${raw} B, ${br} B as brotli\n`)
