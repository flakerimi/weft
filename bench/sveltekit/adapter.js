// Which adapter a build uses. BENCH_STATIC=1 writes the static site with
// adapter-static, in build-static/; without it adapter-node writes the Node
// server, in build/, which serves the same prerendered page.
import node from '@sveltejs/adapter-node'
import staticSite from '@sveltejs/adapter-static'

export default process.env.BENCH_STATIC === '1'
  ? staticSite({ pages: 'build-static', assets: 'build-static' })
  : node({ out: 'build' })
