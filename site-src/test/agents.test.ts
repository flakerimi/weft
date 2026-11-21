// The skills in skills/ (the repository's root) and the files for agents
// the site writes from them, the docs and the catalogue: llms.txt and
// llms-full.txt, as plain functions of what the pages read.
import { readdirSync, readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { loadContent } from '@weft/vite'
import config from '../content.config.ts'
import { installCommand, llmsFull, llmsTxt, repoFile, SITE } from '../agents/llms.ts'
import { slugOf, type Catalog } from '../gallery/model.ts'

const site = new URL('..', import.meta.url).pathname
const skillsDir = new URL('../../skills/', import.meta.url)
const folders = readdirSync(skillsDir, { withFileTypes: true }).filter((d) => d.isDirectory()).map((d) => d.name).sort()
const catalog = JSON.parse(readFileSync(new URL('../../catalog/components.json', import.meta.url), 'utf8')) as Catalog
const content = await loadContent(site, config)
const docs = content.list('docs').map((d) => content.page('docs', d.slug))
const skills = content.list('skills').map((s) => content.page('skills', s.slug))
const input = { docs, skills, components: catalog.components }

describe('the skills', () => {
  it('are the four Weft skills, each a folder with a SKILL.md', () => {
    expect(folders).toEqual(['weft-components', 'weft-go', 'weft-site', 'weft-theme'])
    expect(skills.map((s) => s.slug)).toEqual(folders)
    for (const s of skills) expect(s.file).toBe(`../skills/${s.slug}/SKILL.md`)
  })

  // As Claude Code and the Agent Skills format read them: frontmatter on the
  // first line, a name that is the folder's, lowercase words joined by -, at
  // most 64 characters, and a description of at most 1024 with no tags.
  it('have frontmatter a skill may have: its name and when to use it, nothing else', () => {
    for (const s of skills) {
      const text = readFileSync(new URL(`${s.slug}/SKILL.md`, skillsDir), 'utf8')
      expect(text.startsWith('---\n'), s.file).toBe(true)
      expect(Object.keys(s.frontmatter).sort(), s.file).toEqual(['description', 'name', 'title'])
      const { description } = s.frontmatter
      const name = String(s.frontmatter.name)
      expect(name).toBe(s.slug)
      expect(name).toMatch(/^[a-z0-9]+(?:-[a-z0-9]+)*$/)
      expect(name.length).toBeLessThanOrEqual(64)
      expect(description).toMatch(/^Use (?:when|before) /)
      expect(description.length, s.file).toBeLessThanOrEqual(1024)
      expect(description, s.file).not.toMatch(/[<>]/)
      // A skill's page has one h1, the skill's own heading at the top of its body.
      expect(s.html.match(/<h1 /g), s.file).toHaveLength(1)
      expect(s.html.startsWith('<h1 '), s.file).toBe(true)
    }
  })

  it('say what is not built yet, pointing at Coming next', () => {
    for (const s of skills) {
      expect(s.source, s.file).toContain('## Not yet')
      expect(s.source, s.file).toContain(`${SITE}/docs/coming-next`)
    }
  })
})

describe('llms.txt', () => {
  const text = llmsTxt(input)
  const section = (name: string) => text.split(`\n## ${name}\n`)[1]!.split('\n## ')[0]!

  it('has a title, a summary, then the Docs, Components, Skills and Optional sections', () => {
    expect(text).toMatch(/^# Weft\n\n> [^\n]+\n\n[^#\n][^\n]+\n\n## Docs\n/)
    expect([...text.matchAll(/^## (.+)$/gm)].map((m) => m[1])).toEqual(['Docs', 'Components', 'Skills', 'Optional'])
    expect(text).toContain(`${catalog.components.length} Vapor components`)
    expect(text.endsWith('\n')).toBe(true)
  })

  it('lists every doc with its description', () => {
    const lines = section('Docs').trim().split('\n')
    expect(lines).toEqual(docs.map((d) => `- [${d.frontmatter.title}](${SITE}${d.path}): ${d.frontmatter.description}`))
  })

  it('lists every component with its purpose, its page and its raw entry', () => {
    const body = section('Components')
    expect(body).toContain(`- [catalog.json](${SITE}/catalog.json): `)
    for (const c of catalog.components) {
      const slug = slugOf(c.name)
      expect(body).toContain(`- [${c.name}](${SITE}/components/${slug}): ${c.purpose} ([JSON](${SITE}/components/${slug}.json))\n`)
    }
  })

  it('links every skill by its raw file, with when to use it', () => {
    const lines = section('Skills').trim().split('\n')
    expect(lines).toEqual(skills.map((s) => `- [${s.slug}](${SITE}/skills/${s.slug}.md): ${s.frontmatter.description}`))
  })

  it('leaves the full text and the long references to Optional', () => {
    const body = section('Optional')
    for (const path of ['/llms-full.txt', '/skills', '/components/parity.md', '/components/sizes.md']) expect(body).toContain(`](${SITE}${path})`)
  })
})

describe('llms-full.txt', () => {
  const text = llmsFull(input)

  it('holds every doc and skill as its file has it, after a line naming its page and its file', () => {
    for (const page of [...docs, ...skills]) {
      const marker = `Source: ${SITE}${page.path}\nFile: ${repoFile(page.file)}\n`
      expect(text, page.file).toContain(marker)
      const after = text.slice(text.indexOf(marker) + marker.length)
      expect(after.slice(after.indexOf('\n\n') + 2).startsWith(page.source), page.file).toBe(true)
    }
    expect(repoFile('content/docs/index.md')).toBe('site-src/content/docs/index.md')
    expect(repoFile('../skills/weft-go/SKILL.md')).toBe('skills/weft-go/SKILL.md')
  })

  it('has a blank line before each rule, so Markdown reads no line above one as a heading', () => {
    const lines = text.split('\n')
    const rules = lines.flatMap((line, i) => (/^=+$/.test(line) ? [i] : []))
    expect(rules).toHaveLength(2 * (docs.length + skills.length))
    for (const i of rules) expect(lines[i - 1], `line ${i}: ${lines[i - 1]}`).toBe('')
  })

  it('has the docs first, in their order, then the skills', () => {
    const order = [...text.matchAll(/^Source: (.+)$/gm)].map((m) => m[1])
    expect(order).toEqual([...docs, ...skills].map((p) => SITE + p.path))
  })
})

describe('the install command', () => {
  it('fetches each skill into a folder of its own, for one project or for every project', () => {
    expect(installCommand(['weft-go'], '.claude/skills')).toBe(
      `mkdir -p .claude/skills/weft-go && curl -fsSL ${SITE}/skills/weft-go.md -o .claude/skills/weft-go/SKILL.md`,
    )
    expect(installCommand(['weft-go', 'weft-site'], '~/.claude/skills')).toBe(
      `for s in weft-go weft-site; do mkdir -p ~/.claude/skills/$s && curl -fsSL ${SITE}/skills/$s.md -o ~/.claude/skills/$s/SKILL.md; done`,
    )
  })
})
