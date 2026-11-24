// What the site writes for agents, as plain functions of what the pages
// read: llms.txt (the llms.txt convention: a title, a summary, then sections
// of links), llms-full.txt (every doc and skill as its file has it), and the
// commands that install a skill in Claude Code. The /skills page writes the
// two files, and /index.md (index-md.ts) beside them (pages/skills/index.vue).
import { posix } from 'node:path'
import { groupsOf, slugOf, type CatalogEntry } from '../gallery/model.ts'

/** The site's address: llms.txt and the skills link to it in full, since they are read away from it. */
export const SITE = 'https://weft.base.al'

/** A doc or a skill, as `useContent` gives it. */
export interface AgentPage {
  path: string
  /** From the site root: content/docs/…, ../skills/<name>/SKILL.md. */
  file: string
  source: string
  frontmatter: { title: string; description: string }
}

export interface AgentInput {
  docs: AgentPage[]
  skills: AgentPage[]
  components: Pick<CatalogEntry, 'name' | 'group' | 'purpose'>[]
}

/** A page's file from the repository's root, as the site's folder is site-src/. */
export const repoFile = (file: string) => posix.normalize(`site-src/${file}`)

/** A skill's name: the last segment of its route, /skills/<name>. */
const nameOf = (skill: AgentPage) => skill.path.slice(skill.path.lastIndexOf('/') + 1)

/** llms.txt: what Weft is, the page to start from, then every doc, component and skill, with a line on each. */
export function llmsTxt({ docs, skills, components }: AgentInput): string {
  const item = (label: string, path: string, note: string) => `- [${label}](${SITE}${path}): ${note}`
  const lines = [
    '# Weft',
    '',
    `> Weft is one UI system for Go and Vue apps: ${components.length} Vapor components in @weft/ui that adapt to touch or pointer, landing or app, and the width they get; @weft/vite, a page build that renders pages to HTML and ships JavaScript only for islands; and the weft Go package, whose weft.Serve serves the build and an API from one binary.`,
    '',
    'Weft is pre-alpha: its packages are not on npm. The weft command carries them, and `weft create <name>` makes an app with them, which `weft dev` runs and `weft build` builds into one binary. The docs describe the code as it is; Coming next lists what is planned but not built, so check it before looking for a command or a component. The skills are Claude Code skills, raw Markdown to install as SKILL.md; each component has its catalogue entry as JSON.',
    '',
    '## Start here',
    '',
    item('Build an app with Weft', '/index.md', 'the whole brief for a coding agent asked to build an app: install weft, create the app, look components up, the rules, and what is not built yet.'),
    '',
    '## Docs',
    '',
    ...docs.map((d) => item(d.frontmatter.title, d.path, d.frontmatter.description)),
    '',
    '## Components',
    '',
    item('catalog.json', '/catalog.json', 'every component below in one JSON file: purpose, when to use it and when not, what adapts, props, events, slots and examples.'),
    ...groupsOf(components).flatMap((g) =>
      g.components.map((c) => `${item(c.name, `/components/${slugOf(c.name)}`, c.purpose)} ([JSON](${SITE}/components/${slugOf(c.name)}.json))`),
    ),
    '',
    '## Skills',
    '',
    ...skills.map((s) => item(nameOf(s), `${s.path}.md`, s.frontmatter.description)),
    '',
    '## Optional',
    '',
    item('llms-full.txt', '/llms-full.txt', 'the full text of every doc and skill above, in one file.'),
    item('Skills', '/skills', 'how to install the skills in Claude Code.'),
    item('Parity with Nuxt UI', '/components/parity.md', "how Weft's components compare with Nuxt UI 4's, component by component."),
    item('Sizes', '/components/sizes.md', 'what each component costs a page, brotli.'),
    '',
  ]
  return lines.join('\n')
}

/**
 * llms-full.txt: every doc, in the docs' order, then every skill, each after
 * the lines naming its page and file. Each rule has a blank line before it:
 * read as Markdown, a line of text right above a line of = is a heading.
 */
export function llmsFull({ docs, skills }: AgentInput): string {
  const rule = '='.repeat(80)
  const parts = [...docs, ...skills].map((p) =>
    [rule, `Source: ${SITE}${p.path}`, `File: ${repoFile(p.file)}`, '', rule, '', p.source.endsWith('\n') ? p.source : p.source + '\n'].join('\n'),
  )
  return [
    '# Weft: the docs and skills in full',
    '',
    `Every doc and skill on ${SITE}, each as its file is in the Weft repository, after the lines naming its page and its file. The index is ${SITE}/llms.txt.`,
    '',
    ...parts,
  ].join('\n')
}

/**
 * The one-line command that installs `names` into `dir`, a skills folder:
 * `.claude/skills` for one project, `~/.claude/skills` for every one. Each
 * skill is a folder holding its SKILL.md, fetched from the site.
 */
export function installCommand(names: string[], dir: string): string {
  if (names.length === 1) {
    const [name] = names
    return `mkdir -p ${dir}/${name} && curl -fsSL ${SITE}/skills/${name}.md -o ${dir}/${name}/SKILL.md`
  }
  return `for s in ${names.join(' ')}; do mkdir -p ${dir}/$s && curl -fsSL ${SITE}/skills/$s.md -o ${dir}/$s/SKILL.md; done`
}
