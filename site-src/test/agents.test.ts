// The skills in skills/ (the repository's root) and the files for agents
// the site writes from them, the docs and the catalogue: llms.txt,
// llms-full.txt and index.md, as plain functions of what the pages read.
import { readdirSync, readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { loadContent } from '@weft/vite'
import config from '../content.config.ts'
import { installCommand, llmsFull, llmsTxt, repoFile, SITE } from '../agents/llms.ts'
import { AGENT_PROMPT, commandsNotYet, indexMd, NOT_YET, rulesOf, siteVersion } from '../agents/index-md.ts'
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

  it('has a title, a summary, then the Start here, Docs, Components, Skills and Optional sections', () => {
    expect(text).toMatch(/^# Weft\n\n> [^\n]+\n\n[^#\n][^\n]+\n\n## Start here\n/)
    expect([...text.matchAll(/^## (.+)$/gm)].map((m) => m[1])).toEqual(['Start here', 'Docs', 'Components', 'Skills', 'Optional'])
    expect(text).toContain(`${catalog.components.length} Vapor components`)
    expect(text.endsWith('\n')).toBe(true)
  })

  it('starts with /index.md, the page an agent builds an app from', () => {
    expect(section('Start here').trim()).toMatch(new RegExp(`^- \\[Build an app with Weft\\]\\(${SITE}/index\\.md\\): [^\\n]+$`))
    expect(text.indexOf('](')).toBe(text.indexOf(`](${SITE}/index.md)`))
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

describe('index.md', () => {
  const appAgents = readFileSync(new URL('../../templates/app/AGENTS.md.tmpl', import.meta.url), 'utf8')
  const skill = readFileSync(new URL('../../skills/weft-components/SKILL.md', import.meta.url), 'utf8')
  const comingNext = docs.find((d) => d.path === '/docs/coming-next')!.source
  // The commands cmd/weft has, from its table of commands.
  const commands = [...readFileSync(new URL('../../cmd/weft/main.go', import.meta.url), 'utf8').matchAll(/^\t"([a-z]+)": +\{/gm)].map((m) => m[1]!)
  const text = indexMd({ ...input, version: '1.2.3', appAgents })
  // Built outside a publish, it names no version.
  const unversioned = indexMd({ ...input, appAgents })
  const weftGo = readFileSync(new URL('../../skills/weft-go/SKILL.md', import.meta.url), 'utf8')
  const section = (heading: string) => text.split(`\n## ${heading}\n`)[1]!.split('\n## ')[0]!

  it('is a brief in six steps, with the version and the counts computed', () => {
    expect(text.startsWith('# Build an app with Weft\n\n')).toBe(true)
    expect([...text.matchAll(/^## (.+)$/gm)].map((m) => m[1])).toEqual([
      '1. Check what is installed',
      '2. Create the app',
      '3. Build what was asked',
      '4. Rules',
      '5. Check your work',
      '6. What is not built yet',
      'More',
    ])
    expect(text).toContain('It describes Weft 1.2.3.')
    const groups = new Set(catalog.components.map((c) => c.group)).size
    expect(text).toContain(`${catalog.components.length} components that adapt`)
    expect(text).toContain(`# ${catalog.components.length} components in ${groups} groups, one line each`)
    for (const s of skills) expect(section('More')).toContain(s.slug)
    expect(text.endsWith('\n')).toBe(true)
    expect(text.length, 'an agent pays for every line').toBeLessThan(8 * 1024)
  })

  it('names no version when it is built without one, and checks weft by asking it for weft catalog', () => {
    expect(unversioned).not.toContain('It describes')
    // A version, not an address such as 127.0.0.1.
    expect(unversioned).not.toMatch(/(?<![\d.])\d+\.\d+\.\d+(?!\.?\d)/)
    expect(unversioned).toBe(text.replace(' It describes Weft 1.2.3.', ''))
    for (const page of [text, unversioned]) {
      expect(page).not.toContain('Older than')
      const step = page.split('\n## 1. Check what is installed\n')[1]!.split('\n## ')[0]!
      expect(step).toMatch(/^weft catalog\b/m)
      expect(step).toContain('- If `weft catalog` says `weft: unknown command "catalog"`, this `weft` is older than this page: run `weft update`.')
    }
  })

  it('names only commands weft has, and tells the agent to look components up with weft catalog', () => {
    expect(commands).toContain('catalog')
    const run = [...text.matchAll(/^weft ([a-z]+)/gm)].map((m) => m[1]!)
    expect(run).toEqual(expect.arrayContaining(['version', 'create', 'dev', 'catalog', 'build']))
    for (const c of run) expect(commands, `weft ${c}`).toContain(c)
    const step = section('3. Build what was asked')
    expect(step).toMatch(/^weft catalog +#/m)
    expect(step).toMatch(/^weft catalog [A-Z]\w+( [A-Z]\w+)* +#/m)
    expect(step).toContain('Never read the catalogue file whole')
    expect(step).toContain('Do not know the name? `weft catalog --find <word>`')
    // The JSON is only the fallback, after the commands.
    expect(step.indexOf('/catalog.json')).toBeGreaterThan(step.indexOf('Without `weft`'))
    expect(text.indexOf('/catalog.json')).toBe(text.indexOf(`${SITE}/catalog.json`) + SITE.length)
  })

  it("repeats the rules of the app's AGENTS.md word for word, and the components skill agrees with them", () => {
    const rules = rulesOf(appAgents)
    expect(section('4. Rules')).toContain(rules.map((r) => `- ${r}`).join('\n'))
    // Each rule's point, as the template and the skill both put it.
    const points: [string, RegExp][] = [
      ['raw controls', /raw `<button>`, `<input>`, `<select>` or `<dialog>`.*hand-roll a menu, a popover or a date field/],
      ['the catalogue', /weft catalog/],
      ['layout', /`(Page|Section|Stack|Grid)`.*`(Page|Section|Stack|Grid)`.*`(Page|Section|Stack|Grid)`.*`(Page|Section|Stack|Grid)`/],
      ['Vapor', /<script setup vapor lang="ts">.*Options API.*getCurrentInstance/],
      ['no imports', /need no import|without importing/],
      ['tokens', /--weft-\*/],
      ['no mobile variant', /mobile (variant|version)/],
    ]
    for (const [point, re] of points) {
      expect(rules.some((r) => re.test(r)), `AGENTS.md.tmpl: ${point}`).toBe(true)
      expect(skill.split('\n').some((l) => re.test(l)), `weft-components: ${point}`).toBe(true)
    }
    // Go, as the app's rules and the weft-go skill put it.
    const go = 'Go: use the standard library and `github.com/flakerimi/weft/warp`; add a dependency only for what the standard library cannot do, such as a database driver.'
    expect(rules).toContain(go)
    expect(weftGo).toContain(go)
    expect(rules.some((r) => /Do not edit `\.weft\/`/.test(r))).toBe(true)
  })

  it('says as not built only what Coming next lists, and no command weft has', () => {
    const headings = [...comingNext.matchAll(/^## (.+)$/gm)].map((m) => m[1]!)
    const listed = commandsNotYet(comingNext)
    const step = section('6. What is not built yet')
    expect(step).toContain(`${SITE}/docs/coming-next`)
    // The template has the database and the login since W5.
    expect(step).not.toContain('The app template has no database')
    expect(step).toContain(`- The commands ${listed.slice(0, -1).join(', ')} and ${listed.at(-1)}.`)
    for (const c of listed) expect(commands, c).not.toContain(c.replace(/`/g, '').split(' ')[1])
    const names = new Set(catalog.components.map((c) => c.name))
    for (const n of NOT_YET) {
      expect(headings, n.see).toContain(n.see)
      expect(step).toContain(`- ${n.missing}: ${n.instead}.`)
      for (const [, code] of n.missing.matchAll(/`([^`]+)`/g)) {
        expect(comingNext, `${n.see}: ${code}`).toContain(code)
        const [, command] = /^weft ([a-z]+)/.exec(code!) ?? []
        if (command) expect(commands, code).not.toContain(command)
        expect(names.has(code!), `${code} is a component today`).toBe(false)
      }
      for (const [, code] of n.instead.matchAll(/`([A-Z]\w+)`/g)) expect(names.has(code!), `${n.see}: ${code}`).toBe(true)
    }
  })

  it('reads the version from WEFT_SITE_VERSION, and has none without it', () => {
    expect(siteVersion('0.0.6')).toBe('0.0.6')
    expect(siteVersion('0.1.0-alpha.1')).toBe('0.1.0-alpha.1')
    expect(siteVersion(undefined)).toBeUndefined()
    expect(siteVersion('')).toBeUndefined()
    expect(() => siteVersion('v1')).toThrow(/^index\.md: WEFT_SITE_VERSION is "v1", not a release version; /)
  })

  it('refuses rules it cannot repeat', () => {
    expect(() => rulesOf('# AGENTS.md\n')).toThrow(/has no "## Rules" section/)
    expect(() => rulesOf('## Rules\n\n- [[.Name]] is an app\n')).toThrow(/\[\[ \]\]/)
    expect(() => rulesOf('## Rules\n\n- one\n  two\n')).toThrow(/one line/)
  })
})

// The prompt the landing hero offers to paste into a coding agent: it sends
// the agent to /index.md, and asks only for what that page tells it to do.
describe('the prompt for agents', () => {
  const appAgents = readFileSync(new URL('../../templates/app/AGENTS.md.tmpl', import.meta.url), 'utf8')
  const page = indexMd({ ...input, appAgents })
  // What the page gives the agent to run: each line of a sh block, and each command in backticks.
  const blocks = [...page.matchAll(/^```sh\n([\s\S]*?)^```$/gm)].flatMap((m) => m[1]!.split('\n'))
  const given = [...blocks, ...[...page.matchAll(/`([^`\n]+)`/g)].map((m) => m[1]!)].map((l) => l.trim())

  it('is one line that sends the agent to /index.md and ends where the person types what to build', () => {
    expect(AGENT_PROMPT).toBe(
      'Read https://weft.base.al/index.md and follow it: install weft, create the app, and learn the components with weft catalog. Then build: ',
    )
    expect(AGENT_PROMPT).not.toMatch(/\n/)
    expect(AGENT_PROMPT.endsWith('Then build: ')).toBe(true)
    expect([...AGENT_PROMPT.matchAll(/https?:\/\/\S+/g)].map((m) => m[0])).toEqual([`${SITE}/index.md`])
  })

  it('asks only for what /index.md gives a command for', () => {
    // Each weft command it names is one the page tells the agent to run.
    const named = [...AGENT_PROMPT.matchAll(/\bweft ([a-z]+)\b/g)].map((m) => m[1]!)
    expect(named).toEqual(['catalog'])
    for (const c of named) expect(given.some((l) => l.startsWith(`weft ${c}`)), `weft ${c}`).toBe(true)
    // And each step it asks for, in its own words.
    const steps: [string, string][] = [
      ['install weft', `curl -fsSL ${SITE}/install | sh`],
      ['create the app', 'weft create <name>'],
      ['learn the components with weft catalog', 'weft catalog Card Table Form'],
    ]
    for (const [step, command] of steps) {
      expect(AGENT_PROMPT, step).toContain(step)
      expect(given.some((l) => l.startsWith(command)), `${step}: ${command}`).toBe(true)
    }
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
