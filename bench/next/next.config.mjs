// The Next.js side of the benchmark, with Next's defaults (the App Router).
// BENCH_STATIC=1 makes the static export, in out-static/; without it the
// build is the one `next start` serves, in .next/, where the landing is
// prerendered too. The root is bench/, so the page can import content.json
// and the shared stylesheet from beside the other sites.
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const bench = join(dirname(fileURLToPath(import.meta.url)), '..')
const exported = process.env.BENCH_STATIC === '1'

/** @type {import('next').NextConfig} */
export default {
  ...(exported ? { output: 'export', distDir: 'out-static' } : {}),
  turbopack: { root: bench },
  outputFileTracingRoot: bench,
}
