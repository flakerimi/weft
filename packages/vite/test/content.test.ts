import { createHash } from 'node:crypto'
import { mkdirSync, mkdtempSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { afterAll, describe, expect, it } from 'vitest'
import { highlightCode, loadContent, type ContentConfig } from '../src/content.ts'

const temps: string[] = []
afterAll(() => {
  for (const t of temps) rmSync(t, { recursive: true, force: true })
})

/** A site root in a temp folder holding these files, by path from the root. */
function siteWith(files: Record<string, string>): string {
  const root = mkdtempSync(join(tmpdir(), 'weft-content-'))
  temps.push(root)
  for (const [path, text] of Object.entries(files)) {
    mkdirSync(dirname(join(root, path)), { recursive: true })
    writeFileSync(join(root, path), text)
  }
  return root
}

const docs: ContentConfig = { docs: {} }
/** A page's frontmatter and body, as a file holds them. */
const page = (title: string, body = '', extra = '') =>
  `---\ntitle: ${title}\ndescription: About ${title.toLowerCase()}.\n${extra}---\n${body}`

/** The error loading `files` with `config` fails with. */
async function failure(files: Record<string, string>, config: unknown = docs): Promise<string> {
  try {
    await loadContent(siteWith(files), config)
  } catch (e) {
    return (e as Error).message
  }
  throw new Error('loadContent did not fail')
}

describe('frontmatter', () => {
  it('reads title, description and order, and the fields the collection schema adds', async () => {
    const root = siteWith({
      'content/docs/intro.md': page('Intro', 'Hi.\n', 'order: 2\nsection: Start\ntags: [a, b]\ndraft: false\n'),
    })
    const content = await loadContent(root, { docs: { schema: { section: 'string', tags: 'string[]?', draft: 'boolean?' } } })
    expect(content.page('docs', 'intro').frontmatter).toEqual({
      title: 'Intro',
      description: 'About intro.',
      order: 2,
      section: 'Start',
      tags: ['a', 'b'],
      draft: false,
    })
  })

  it('fails on a page with no title, naming the file and line', async () => {
    expect(await failure({ 'content/docs/intro.md': '---\ndescription: Hi.\n---\nBody.\n' })).toBe(
      'content/docs/intro.md:1: the frontmatter has no title, which every page of the docs collection needs.',
    )
    expect(await failure({ 'content/docs/intro.md': '# No frontmatter\n' })).toBe(
      'content/docs/intro.md:1: the frontmatter has no title, which every page of the docs collection needs.',
    )
  })

  it('fails on a page with no description', async () => {
    expect(await failure({ 'content/docs/intro.md': '---\ntitle: Intro\n---\n' })).toBe(
      'content/docs/intro.md:1: the frontmatter has no description, which every page of the docs collection needs.',
    )
  })

  it('fails on a field of the wrong type, at the line of its value', async () => {
    expect(await failure({ 'content/docs/intro.md': page('Intro', '', 'order: first\n') })).toBe(
      'content/docs/intro.md:4: order is "first", but the docs collection\'s schema makes it a number.',
    )
    expect(await failure({ 'content/docs/intro.md': '---\ntitle: 12\ndescription: Hi.\n---\n' })).toBe(
      'content/docs/intro.md:2: title is 12, but the docs collection\'s schema makes it a string.',
    )
    expect(
      await failure(
        { 'content/docs/intro.md': page('Intro', '', 'tags: [a, 1]\n') },
        { docs: { schema: { tags: 'string[]' } } },
      ),
    ).toBe('content/docs/intro.md:4: tags is ["a",1], but the docs collection\'s schema makes it a list of strings.')
  })

  it('fails on a required schema field that is missing', async () => {
    expect(await failure({ 'content/docs/intro.md': page('Intro') }, { docs: { schema: { section: 'string' } } })).toBe(
      'content/docs/intro.md:1: the frontmatter has no section, which every page of the docs collection needs.',
    )
  })

  it('fails on a field the schema does not have', async () => {
    expect(await failure({ 'content/docs/intro.md': page('Intro', '', 'ordr: 1\n') })).toBe(
      'content/docs/intro.md:4: ordr is not in the docs collection\'s schema; fix the name, or add it to the schema in content.config.ts.',
    )
  })

  it('fails on frontmatter that is not valid YAML, at its line', async () => {
    expect(await failure({ 'content/docs/intro.md': page('Intro', '', 'title: Again\n') })).toBe(
      'content/docs/intro.md:4: the frontmatter is not valid YAML (Map keys must be unique).',
    )
    expect(await failure({ 'content/docs/intro.md': '---\n- a\n- b\n---\n' })).toBe(
      'content/docs/intro.md:2: the frontmatter is not a set of fields (name: value).',
    )
  })

  it('reads a file that starts with a byte order mark', async () => {
    const content = await loadContent(siteWith({ 'content/docs/intro.md': '\uFEFF' + page('Intro', 'Hi.\n') }), docs)
    expect(content.page('docs', 'intro').frontmatter.title).toBe('Intro')
  })

  it('fails on frontmatter that is never closed', async () => {
    expect(await failure({ 'content/docs/intro.md': '---\ntitle: Intro\n\n# Body\n' })).toBe(
      'content/docs/intro.md:1: the frontmatter opened here is never closed by a --- line.',
    )
  })
})

describe('headings', () => {
  const body = [
    '# Getting started',
    '',
    '## Install Weft!',
    '',
    '### The `weft` CLI',
    '',
    '#### Deep',
    '',
    '## Install Weft',
    '',
    '## Über uns',
    '',
    '## Install Weft',
    '',
  ].join('\n')

  it('get ids by GitHub\'s slug rules, unique on the page', async () => {
    const content = await loadContent(siteWith({ 'content/docs/start.md': page('Start', body) }), docs)
    const { html } = content.page('docs', 'start')
    expect(html).toContain('<h1 id="getting-started">Getting started</h1>')
    expect(html).toContain('<h2 id="install-weft">Install Weft!</h2>')
    expect(html).toContain('<h3 id="the-weft-cli">The <code>weft</code> CLI</h3>')
    expect(html).toContain('<h4 id="deep">Deep</h4>')
    expect(html).toContain('<h2 id="install-weft-1">Install Weft</h2>')
    expect(html).toContain('<h2 id="über-uns">Über uns</h2>')
    expect(html).toContain('<h2 id="install-weft-2">Install Weft</h2>')
  })

  it('make the table of contents from h2 and h3', async () => {
    const content = await loadContent(siteWith({ 'content/docs/start.md': page('Start', body) }), docs)
    expect(content.page('docs', 'start').toc).toEqual([
      { depth: 2, id: 'install-weft', text: 'Install Weft!' },
      { depth: 3, id: 'the-weft-cli', text: 'The weft CLI' },
      { depth: 2, id: 'install-weft-1', text: 'Install Weft' },
      { depth: 2, id: 'über-uns', text: 'Über uns' },
      { depth: 2, id: 'install-weft-2', text: 'Install Weft' },
    ])
  })
})

describe('links', () => {
  const files = {
    'content/docs/index.md': page('Docs', 'See [islands](guides/islands.md) and [serving](./guides/serving.md#etags).\n'),
    'content/docs/guides/islands.md': page(
      'Islands',
      'Back to [the docs](../index.md), [down](#when-they-load), [Go](https://go.dev/x.md), [mail](mailto:a@b.c), [abs](/docs/guides/serving#etags).\n\n## When they load\n',
    ),
    'content/docs/guides/serving.md': page('Serving', '## ETags\n'),
  }

  it('rewrites relative .md links to the routes of their pages, keeping the anchor', async () => {
    const content = await loadContent(siteWith(files), docs)
    expect(content.page('docs', '').html).toContain(
      '<a href="/docs/guides/islands">islands</a> and <a href="/docs/guides/serving#etags">serving</a>',
    )
    const { html } = content.page('docs', 'guides/islands')
    expect(html).toContain('<a href="/docs">the docs</a>')
    expect(html).toContain('<a href="#when-they-load">down</a>')
    // External links are left as they are, and not fetched.
    expect(html).toContain('<a href="https://go.dev/x.md">Go</a>')
    expect(html).toContain('<a href="mailto:a@b.c">mail</a>')
    expect(html).toContain('<a href="/docs/guides/serving#etags">abs</a>')
  })

  it('serves a collection at the base its config gives', async () => {
    const content = await loadContent(siteWith(files), { docs: { base: '/manual' } })
    expect(content.page('docs', '').path).toBe('/manual')
    expect(content.page('docs', '').html).toContain('<a href="/manual/guides/islands">islands</a>')
  })

  it('fails on a link to a missing doc, naming the file and line', async () => {
    expect(
      await failure({ ...files, 'content/docs/guides/serving.md': page('Serving', '## ETags\n\nFirst line,\nthen [gone](../missing.md).\n') }),
    ).toBe('content/docs/guides/serving.md:8: the link to ../missing.md leads to no page; content/docs/missing.md does not exist.')
  })

  it('fails on a link to a missing heading on another page', async () => {
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', '\n[x](guides/serving.md#etag)\n') })).toBe(
      'content/docs/index.md:6: the link to guides/serving.md#etag leads to no heading; content/docs/guides/serving.md has no heading with the id etag.',
    )
  })

  it('fails on a link to a missing heading on the same page', async () => {
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', '## Here\n\n[x](#there)\n') })).toBe(
      'content/docs/index.md:7: the link to #there leads to no heading; this page has no heading with the id there.',
    )
  })

  it('checks a link from the site root into a collection', async () => {
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', '[x](/docs/guides/gone)\n') })).toBe(
      'content/docs/index.md:5: the link to /docs/guides/gone leads to no page; the docs collection has no page guides/gone.',
    )
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', '[x](/docs/guides/serving#nope)\n') })).toBe(
      'content/docs/index.md:5: the link to /docs/guides/serving#nope leads to no heading; content/docs/guides/serving.md has no heading with the id nope.',
    )
  })

  it('finds the line of a link in a table', async () => {
    const table = '| a | b |\n|---|---|\n| c | d |\n| e | [x](gone.md) |\n'
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', table) })).toBe(
      'content/docs/index.md:8: the link to gone.md leads to no page; content/docs/gone.md does not exist.',
    )
  })

  it('fails on a relative link that is not to a .md file, or leaves content/', async () => {
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', '[x](guides/)\n') })).toBe(
      'content/docs/index.md:5: the link to guides/ is relative but not to a .md file; link to the page\'s .md file, or give a path from the site root.',
    )
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', '[x](../../README.md)\n') })).toBe(
      'content/docs/index.md:5: the link to ../../README.md leads outside content/; link to the .md file of a page in a collection.',
    )
  })

  it('keeps the query and the anchor of a relative .md link, and checks the anchor', async () => {
    const withQuery = { ...files, 'content/docs/index.md': page('Docs', '[x](guides/serving.md?tab=go#etags)\n') }
    const content = await loadContent(siteWith(withQuery), docs)
    expect(content.page('docs', '').html).toContain('<a href="/docs/guides/serving?tab=go#etags">x</a>')
    expect(await failure({ ...files, 'content/docs/index.md': page('Docs', '[x](guides/serving.md?tab=go#nope)\n') })).toBe(
      'content/docs/index.md:5: the link to guides/serving.md?tab=go#nope leads to no heading; content/docs/guides/serving.md has no heading with the id nope.',
    )
  })
})

describe('images', () => {
  const png = Buffer.from('89504e470d0a1a0a0000000d49484452', 'hex')
  const hashOf = (bytes: Buffer) => createHash('sha256').update(bytes).digest('hex').slice(0, 8)

  it('serve an image beside the page under a content-addressed name', async () => {
    const root = siteWith({
      'content/docs/index.md': page('Docs', '![Shot](./shot.png) and ![Again](guides/../shot.png \'t\')\n'),
      'content/docs/guides/islands.md': page('Islands', '![Shot](../shot.png)\n'),
    })
    writeFileSync(join(root, 'content/docs/shot.png'), png)
    const content = await loadContent(root, docs)
    const src = `/assets/content/shot-${hashOf(png)}.png`
    expect(content.page('docs', '').html).toContain(`<img src="${src}" alt="Shot"> and <img src="${src}" alt="Again" title="t">`)
    expect(content.page('docs', 'guides/islands').html).toContain(`<img src="${src}" alt="Shot">`)
    // One file, however many pages show it.
    expect(content.assets()).toEqual([{ path: src, file: join(root, 'content/docs/shot.png') }])
  })

  it('fail on an image that is not there, naming the file and line', async () => {
    expect(await failure({ 'content/docs/index.md': page('Docs', '\n![A](./a.png)\n') })).toBe(
      'content/docs/index.md:6: the image ./a.png leads to no file; content/docs/a.png does not exist.',
    )
    expect(await failure({ 'content/docs/index.md': page('Docs', '![A](../../a.png)\n') })).toBe(
      'content/docs/index.md:5: the image ../../a.png leads outside content/; put it beside the page, or in public/ with a path from the site root.',
    )
  })

  it('fail on a relative image that is not an image file', async () => {
    expect(await failure({ 'content/docs/index.md': page('Docs', '![A](./notes.md)\n') })).toBe(
      'content/docs/index.md:5: the image ./notes.md is not a .png, .jpg, .jpeg, .gif, .webp, .avif or .svg file.',
    )
  })
})

describe('code blocks', () => {
  const code = '```ts\nconst a: number = 1 // one\n```\n'

  it('are highlighted at build time with CSS variables, and carry no script', async () => {
    const content = await loadContent(siteWith({ 'content/docs/code.md': page('Code', code) }), docs)
    const { html, parts } = content.page('docs', 'code')
    expect(html).toContain('<pre class="shiki weft"')
    expect(html).toContain('style="color:var(--weft-code-token-keyword)">const</span>')
    expect(html).toContain('var(--weft-code-token-comment)')
    expect(html).not.toMatch(/<script|#[0-9a-f]{6}/i)
    // Without copy, the page is one part.
    expect(parts).toEqual([{ html }])
  })

  it('highlight a block with no language as plain text', async () => {
    const content = await loadContent(siteWith({ 'content/docs/code.md': page('Code', '```\n<b>&\n```\n\n    indented\n') }), docs)
    const { html } = content.page('docs', 'code')
    expect(html).toContain('<span>&#x3C;b>&#x26;</span>')
    expect(html).toContain('<span>indented</span>')
  })

  it('split a page at each block marked copy, which carries its source for a copy island', async () => {
    const body = `Intro.\n\n\`\`\`sh copy\nweft serve\n\`\`\`\n\nAfter.\n`
    const content = await loadContent(siteWith({ 'content/docs/code.md': page('Code', body) }), docs)
    const { html, parts } = content.page('docs', 'code')
    expect(parts).toHaveLength(3)
    expect(parts[0]).toEqual({ html: '<p>Intro.</p>\n' })
    expect(parts[1]).toMatchObject({ code: 'weft serve\n', lang: 'sh' })
    expect(parts[1]!.html).toMatch(/^<pre class="shiki weft"[^]*>weft<\/span>[^]*> serve<\/span>[^]*<\/pre>\n$/)
    expect(parts[2]).toEqual({ html: '<p>After.</p>\n' })
    expect(html).toBe(parts.map((p) => p.html).join(''))
  })

  it('read the language in any case', async () => {
    const body = '```JSON\n{ "a": 1 }\n```\n\n```Ts copy\nconst a = 1\n```\n'
    const content = await loadContent(siteWith({ 'content/docs/code.md': page('Code', body) }), docs)
    const { html, parts } = content.page('docs', 'code')
    expect(html).toContain('var(--weft-code-token-keyword)">const</span>')
    expect(html).toMatch(/var\(--weft-code-token-[\w-]+\)">"a"<\/span>/)
    expect(parts[1]).toMatchObject({ lang: 'ts' })
  })

  it('fail on ansi, whose terminal colours Weft does not draw', async () => {
    expect(await failure({ 'content/docs/code.md': page('Code', '```ansi\nx\n```\n') })).toBe(
      'content/docs/code.md:5: the code block\'s language ansi draws terminal colours, which Weft\'s code colours do not cover; use text.',
    )
  })

  it('fail on a language Shiki does not know', async () => {
    expect(await failure({ 'content/docs/code.md': page('Code', '\n```klingon\nx\n```\n') })).toBe(
      'content/docs/code.md:6: the code block\'s language klingon is not one Shiki knows; use one of its language ids, or text.',
    )
  })

  it('fail on a word after the language other than copy', async () => {
    expect(await failure({ 'content/docs/code.md': page('Code', '```ts cpoy\nx\n```\n') })).toBe(
      'content/docs/code.md:5: the code block has "cpoy" after its language; the one word Weft reads there is copy.',
    )
  })

  it('fail on a copy block inside a list or a quote', async () => {
    expect(await failure({ 'content/docs/code.md': page('Code', '- Step:\n\n  ```sh copy\n  x\n  ```\n') })).toBe(
      'content/docs/code.md:7: a copy code block must be at the top level of the page, not inside a list or a quote.',
    )
  })
})

describe('collections', () => {
  const files = {
    'content/docs/index.md': page('Docs', '', 'order: 1\n'),
    'content/docs/guides/islands.md': page('Islands', '', 'order: 3\n'),
    'content/docs/guides/serving.md': page('Serving', '', 'order: 2\n'),
    'content/docs/zzz.md': page('Appendix'),
    'content/docs/aaa.md': page('Also unordered'),
  }

  it('order pages by order, then by slug, for previous and next', async () => {
    const content = await loadContent(siteWith(files), docs)
    expect(content.list('docs').map((p) => p.slug)).toEqual(['', 'guides/serving', 'guides/islands', 'aaa', 'zzz'])
    expect(content.list('docs')[1]).toEqual({
      slug: 'guides/serving',
      path: '/docs/guides/serving',
      title: 'Serving',
      description: 'About serving.',
      order: 2,
    })
    const serving = content.page('docs', 'guides/serving')
    expect(serving.prev?.path).toBe('/docs')
    expect(serving.next?.path).toBe('/docs/guides/islands')
    expect(content.page('docs', '').prev).toBeNull()
    expect(content.page('docs', 'zzz').next).toBeNull()
  })

  it('give the params for getPaths', async () => {
    const content = await loadContent(siteWith(files), docs)
    expect(content.paths('docs')).toEqual([
      { slug: '' },
      { slug: 'guides/serving' },
      { slug: 'guides/islands' },
      { slug: 'aaa' },
      { slug: 'zzz' },
    ])
    expect(content.paths('docs', 'path')[1]).toEqual({ path: 'guides/serving' })
  })

  it('refuse a page or collection that is not there, in a sentence', async () => {
    const content = await loadContent(siteWith(files), docs)
    expect(() => content.page('docs', 'gone')).toThrow(
      new Error('useContent: the docs collection has no page gone (content/docs/gone.md).'),
    )
    expect(() => content.page('blog', 'x')).toThrow(
      new Error('useContent: content.config.ts declares no collection blog; add it there, with its folder content/blog.'),
    )
  })

  it('refuse two files that make one page', async () => {
    expect(await failure({ 'content/docs/guides.md': page('A'), 'content/docs/guides/index.md': page('B') })).toBe(
      'content/docs/guides.md and content/docs/guides/index.md are both the page guides of the docs collection; remove one.',
    )
  })

  it('refuse a file name that is not a URL segment, and a file that is not Markdown', async () => {
    expect(await failure({ 'content/docs/My Guide.md': page('A') })).toBe(
      'content/docs/My Guide.md has a name that is not a plain URL segment; use lowercase letters, digits, - and _.',
    )
    expect(await failure({ 'content/docs/intro.md': page('A'), 'content/docs/shot.txt': 'x' })).toBe(
      'content/docs/shot.txt is neither a .md file nor an image; put other files in public/.',
    )
  })

  it('refuse a symlink in a collection', async () => {
    const root = siteWith({ 'content/docs/intro.md': page('A'), 'elsewhere.md': page('B') })
    symlinkSync(join(root, 'elsewhere.md'), join(root, 'content/docs/linked.md'))
    await expect(loadContent(root, docs)).rejects.toThrow(
      new Error('content/docs/linked.md is a symlink, which Weft does not follow; put the file itself there.'),
    )
  })

  it('refuse a folder of content/ that content.config.ts does not declare', async () => {
    expect(await failure({ 'content/docs/intro.md': page('A'), 'content/blog/post.md': page('B') })).toBe(
      'content/blog is not a collection content.config.ts declares; declare it there, or move it out of content/.',
    )
  })

  it('refuse a declared collection with no folder', async () => {
    expect(await failure({ 'content/docs/intro.md': page('A') }, { docs: {}, blog: {} })).toBe(
      'content.config.ts declares the collection blog, but content/blog does not exist.',
    )
  })

  it('refuse a config that is not a set of collections, or a schema it cannot read', async () => {
    await expect(loadContent(siteWith({ 'content/docs/intro.md': page('A') }), undefined)).rejects.toThrow(
      new Error(
        'content/ has collections but the site has no content.config.ts; add one whose default export declares them, like { docs: {} }.',
      ),
    )
    expect(await failure({ 'content/docs/intro.md': page('A') }, [])).toBe(
      'content.config.ts: the default export is not a set of collections, like { docs: {} }.',
    )
    expect(await failure({ 'content/docs/intro.md': page('A') }, { docs: { schema: { title: 'string' } } })).toBe(
      'content.config.ts: the docs collection\'s schema names title, which every collection has; leave it out.',
    )
    expect(await failure({ 'content/docs/intro.md': page('A') }, { docs: { schema: { when: 'date' } } })).toBe(
      'content.config.ts: the docs collection\'s schema gives when the type "date"; use string, number, boolean or string[], with ? when optional.',
    )
    expect(await failure({ 'content/docs/intro.md': page('A') }, { docs: { base: 'docs/' } })).toBe(
      'content.config.ts: the docs collection\'s base is "docs/"; give a path from the site root with no trailing /, like /docs.',
    )
    expect(await failure({ 'content/docs/intro.md': page('A') }, { docs: { sheme: {} } })).toBe(
      'content.config.ts: the docs collection has sheme, which a collection does not take; it takes base, dir, index, titleField and schema.',
    )
  })
})

describe('a collection in a folder of its own', () => {
  // Agent skills, as Claude Code reads them: skills/<name>/SKILL.md, named by `name`.
  const skills: ContentConfig = { docs: {}, skills: { dir: '../skills', index: 'SKILL.md', titleField: 'name' } }
  const skill = (name: string, body = '') => `---\nname: ${name}\ndescription: Use when ${name} is the job.\n---\n${body}`
  /** The site's folder, beside a skills folder, in one temp folder. */
  const withSkills = (files: Record<string, string>) => join(siteWith(files), 'site')

  it('reads pages from the folder dir names, each folder\'s page the file index names', async () => {
    const root = withSkills({
      'site/content/docs/index.md': page('Docs', 'Read [the site skill](/skills/weft-site).\n'),
      'skills/weft-site/SKILL.md': skill('weft-site', '# Build a site\n\nSee [Go](../weft-go/SKILL.md#serve).\n'),
      'skills/weft-go/SKILL.md': skill('weft-go', '## Serve\n'),
    })
    const content = await loadContent(root, skills)
    expect(content.list('skills').map((p) => [p.slug, p.path, p.title])).toEqual([
      ['weft-go', '/skills/weft-go', 'weft-go'],
      ['weft-site', '/skills/weft-site', 'weft-site'],
    ])
    const site = content.page('skills', 'weft-site')
    expect(site.file).toBe('../skills/weft-site/SKILL.md')
    // The title is the field titleField names; the frontmatter keeps the file's fields, and title as the page's title.
    expect(site.frontmatter).toEqual({ name: 'weft-site', description: 'Use when weft-site is the job.', title: 'weft-site' })
    expect(site.html).toContain('<a href="/skills/weft-go#serve">Go</a>')
    expect(content.page('docs', '').html).toContain('<a href="/skills/weft-site">the site skill</a>')
  })

  it('gives each page its file\'s text as it is on disk', async () => {
    const text = '\uFEFF' + skill('weft-go', 'Line one,\r\nline two.\r\n')
    const content = await loadContent(withSkills({ 'site/content/docs/index.md': page('Docs'), 'skills/weft-go/SKILL.md': text }), skills)
    expect(content.page('skills', 'weft-go').source).toBe(text)
    expect(content.page('docs', '').source).toBe(page('Docs'))
  })

  it('checks the title field as title is checked, and refuses title beside it', async () => {
    const files = { 'site/content/docs/index.md': page('Docs') }
    const fail = async (more: Record<string, string>) => {
      try {
        await loadContent(withSkills({ ...files, ...more }), skills)
      } catch (e) {
        return (e as Error).message
      }
      throw new Error('loadContent did not fail')
    }
    expect(await fail({ 'skills/weft-go/SKILL.md': '---\ndescription: Hi.\n---\n' })).toBe(
      '../skills/weft-go/SKILL.md:1: the frontmatter has no name, which every page of the skills collection needs.',
    )
    expect(await fail({ 'skills/weft-go/SKILL.md': skill('weft-go').replace('description', 'title: Go\ndescription') })).toBe(
      '../skills/weft-go/SKILL.md:3: the skills collection takes a page\'s title from name, so a page has no title field; remove it.',
    )
    // Only the file index names is exempt from the rule for names; another is still a URL segment.
    expect(await fail({ 'skills/weft-go/SKILL.md': skill('weft-go'), 'skills/weft-go/Notes.md': skill('notes') })).toBe(
      '../skills/weft-go/Notes.md has a name that is not a plain URL segment; use lowercase letters, digits, - and _.',
    )
    expect(await fail({ 'skills/weft-go/SKILL.md': skill('weft-go', '[x](../../site/content/docs/index.md) and [y](../../README.md)\n') })).toBe(
      '../skills/weft-go/SKILL.md:5: the link to ../../README.md leads outside content/ and ../skills/; link to the .md file of a page in a collection.',
    )
  })

  it('says where a missing page or folder would be', async () => {
    const content = await loadContent(withSkills({ 'site/content/docs/index.md': page('Docs'), 'skills/weft-go/SKILL.md': skill('weft-go') }), skills)
    expect(() => content.page('skills', 'gone')).toThrow(new Error('useContent: the skills collection has no page gone (../skills/gone/SKILL.md).'))
    expect(await failure({ 'content/docs/index.md': page('Docs') }, skills)).toBe(
      'content.config.ts declares the collection skills, but ../skills does not exist.',
    )
  })

  it('refuses a dir, index or titleField it cannot use, in one sentence', async () => {
    const one = { 'content/docs/intro.md': page('A') }
    const why = (skills: unknown) => failure(one, { docs: {}, skills })
    expect(await why({ dir: '/abs/skills' })).toBe(
      'content.config.ts: the skills collection\'s dir is "/abs/skills"; give a folder from the site root, like ../skills, with / between names.',
    )
    expect(await why({ dir: '..' })).toBe('content.config.ts: the skills collection\'s dir ".." holds the site itself; give a folder beside it or under it.')
    expect(await why({ dir: 'content/other' })).toBe(
      'content.config.ts: the skills collection\'s dir is under content/, where each folder is the collection of its name; leave dir out, or give a folder outside content/.',
    )
    expect(await why({ dir: 'content/docs/skills' })).toBe(
      'content.config.ts: the skills collection\'s dir is under content/, where each folder is the collection of its name; leave dir out, or give a folder outside content/.',
    )
    expect(await failure(one, { docs: { dir: '../notes' }, skills: { dir: '../notes/skills' } })).toBe(
      'content.config.ts: the docs and skills collections\' folders, ../notes and ../notes/skills, overlap; give each a folder of its own.',
    )
    expect(await why({ index: 'skills/SKILL.md' })).toBe(
      'content.config.ts: the skills collection\'s index is "skills/SKILL.md"; give the name of a .md file, like SKILL.md.',
    )
    expect(await why({ titleField: 'order' })).toBe(
      'content.config.ts: the skills collection\'s titleField is "order"; give the name of a frontmatter field other than description and order, like name.',
    )
    expect(await why({ titleField: 'name', schema: { title: 'string' } })).toBe(
      'content.config.ts: the skills collection\'s schema names title, which is the page\'s title; leave it out.',
    )
    expect(await why({ titleField: 'name', schema: { name: 'string' } })).toBe(
      'content.config.ts: the skills collection\'s schema names name, which every collection has; leave it out.',
    )
  })
})

describe('highlightCode', () => {
  it('highlights source a page reads itself, as code blocks are, by language id or alias', async () => {
    const html = await highlightCode('<script setup lang="ts">\nconst a = 1\n</script>\n', 'vue')
    expect(html).toMatch(/^<pre class="shiki weft"/)
    expect(html).toContain('style="color:var(--weft-code-token-keyword)">const</span>')
    expect(html).not.toMatch(/#[0-9a-f]{6}/i)
    // The last newline is not an empty line.
    expect(html).not.toMatch(/<span class="line"><\/span><\/code>/)
    expect(await highlightCode('a & b', 'TXT')).toContain('<span>a &#x26; b</span>')
    expect(await highlightCode('{"a": 1}', 'JSON')).toContain('var(--weft-code-token-')
  })

  it('refuses a language Shiki does not know, in one sentence', async () => {
    await expect(highlightCode('x', 'klingon')).rejects.toThrow(
      'highlightCode: the language klingon is not one Shiki knows; use one of its language ids, or text.',
    )
  })
})
