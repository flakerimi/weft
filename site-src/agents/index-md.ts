// /index.md: the page an agent is sent to ("use weft.base.al/index.md to
// build me an app") and can act on with nothing else. It is a plain function
// of what llms.txt is made from, plus the Weft version it describes, the app
// template's AGENTS.md (whose rules it repeats word for word) and Coming next
// (whose commands it lists as not built). The /skills page writes it
// (pages/skills/index.vue). It is short on purpose: an agent pays for every
// line, and the rest is on disk after weft create.
import { SITE, type AgentInput } from './llms.ts'
import { groupsOf } from '../gallery/model.ts'

export interface IndexInput extends AgentInput {
  /** The release the page describes: see siteVersion. */
  version: string
  /** templates/app/AGENTS.md.tmpl, as it is on disk. */
  appAgents: string
}

/** A release version, as scripts/version.sh accepts it. */
const RELEASE = /^\d+\.\d+\.\d+(?:-[a-z0-9.]+)?$/

/**
 * The Weft release the site describes: WEFT_SITE_VERSION, which
 * scripts/publish-site.sh sets to the version it publishes, else the last
 * release site/releases.txt lists, the one the live site serves.
 */
export function siteVersion(env: string | undefined, releases: string): string {
  if (env) {
    if (!RELEASE.test(env)) throw new Error(`index.md: WEFT_SITE_VERSION is "${env}", not a release version; set it to one like 0.0.6, or unset it to use the last line of site/releases.txt.`)
    return env
  }
  const last = releases.trim().split('\n').at(-1)?.trim() ?? ''
  if (!RELEASE.test(last)) throw new Error(`index.md: site/releases.txt ends with "${last}", not a release version; end it with the latest published release, or set WEFT_SITE_VERSION.`)
  return last
}

/** The bullets of a Markdown file's `## Rules` section, each as one line. */
export function rulesOf(markdown: string): string[] {
  const section = markdown.split(/^## Rules\n/m)[1]
  if (section === undefined) throw new Error('index.md: templates/app/AGENTS.md.tmpl has no "## Rules" section; give it one, as a list, since /index.md repeats it.')
  // The list is the first run of lines after the heading that are not blank.
  const lines = section.replace(/^\n+/, '').split('\n')
  const block = lines.slice(0, lines.indexOf('') === -1 ? lines.length : lines.indexOf(''))
  if (block.length === 0 || !block.every((l) => l.startsWith('- '))) {
    throw new Error('index.md: the "## Rules" section of templates/app/AGENTS.md.tmpl is not a list of rules, each on one line; write it so, since /index.md repeats it.')
  }
  const rules = block.map((l) => l.slice(2))
  if (rules.some((r) => r.includes('[['))) throw new Error('index.md: a rule in templates/app/AGENTS.md.tmpl uses a template value; keep the rules free of [[ ]], since /index.md repeats them.')
  return rules
}

/** The first cells of Coming next's Commands table: the commands not built yet. */
export function commandsNotYet(comingNext: string): string[] {
  const section = comingNext.split(/^## Commands\n/m)[1]?.split(/^## /m)[0] ?? ''
  const rows = section.split('\n').filter((l) => l.startsWith('| `'))
  if (rows.length === 0) throw new Error('index.md: content/docs/coming-next.md has no Commands table; keep one, since /index.md lists the commands it names.')
  return rows.map((r) => r.split('|')[1]!.trim())
}

/**
 * What /index.md says is not built yet, beside the commands: each names the
 * heading of Coming next it stands for (`see`), what is missing, and what to
 * use today. A test checks each against Coming next and the catalogue.
 */
export const NOT_YET: { see: string; missing: string; instead: string }[] = [
  { see: 'Login and permissions', missing: 'Login, sessions and permissions (`weft.Auth`, `weft.Require`)', instead: 'write them in `api/` yourself' },
  { see: 'The Go-to-Vue contract', missing: 'Types and form schemas made from Go structs (`weft gen`, `weft.Validate`, `weft.Resource`)', instead: "write the TypeScript types and the `Form`'s schema by hand" },
  { see: 'App blocks and live data', missing: '`DataTable`, `EntityDialog`, the charts and `useLive`', instead: 'use `Table`, a `Dialog` with a `Form`, and `StatCard`' },
  { see: 'App routes', missing: 'A client-side router and `layouts/`', instead: 'each page is a document of its own; share a layout as a component the pages use' },
  { see: 'Landing blocks', missing: '`Hero`, `Pricing` and the other landing blocks', instead: 'compose them from `Section`, `Grid` and `Card`' },
]

/** /index.md: the whole brief for an agent asked to build an app with Weft. */
export function indexMd({ docs, skills, components, version, appAgents }: IndexInput): string {
  const comingNext = docs.find((d) => d.path === '/docs/coming-next')
  if (!comingNext) throw new Error('index.md: there is no doc at /docs/coming-next; /index.md lists what it says is not built, so keep content/docs/coming-next.md.')
  const groups = groupsOf(components)
  const commands = commandsNotYet(comingNext.source)
  const names = skills.map((s) => s.path.slice(s.path.lastIndexOf('/') + 1))
  const list = (items: string[]) => (items.length < 2 ? items.join('') : `${items.slice(0, -1).join(', ')} and ${items.at(-1)}`)
  return [
    '# Build an app with Weft',
    '',
    `You were sent here to build something with Weft. This page is the whole brief, for a coding agent working in a terminal: follow it from the top. It describes Weft ${version}.`,
    '',
    `Weft is one UI system for Go and Vue: ${components.length} components that adapt to touch or pointer and to the width they get, a page build that ships JavaScript only where a page is interactive, and a Go server, in one binary. You write Vue pages and a Go API; \`weft\` does the rest.`,
    '',
    '## 1. Check what is installed',
    '',
    '```sh',
    'weft version',
    '```',
    '',
    `- No \`weft\`: \`curl -fsSL ${SITE}/install | sh\` puts it in \`~/.local/bin\`.`,
    `- Older than ${version}: \`weft update\`.`,
    '- You also need Go 1.26, Node 22.18 or later and pnpm 10. `weft create` says which is missing.',
    '',
    '## 2. Create the app',
    '',
    '```sh',
    'weft create <name>   # a lowercase letter, then lowercase letters, digits and hyphens',
    'cd <name>',
    'weft dev             # runs until stopped: start it in the background',
    '```',
    '',
    '`weft create` writes the app and runs `pnpm install`. `weft dev` prints the address to open, such as `weft dev: http://localhost:5173`, builds the Go code again when a `.go` file changes, and reloads the browser when a file in `web/` does. Then read the app\'s `AGENTS.md`: the layout, the commands and the rules of that app.',
    '',
    '## 3. Build what was asked',
    '',
    'List the components, then read each one you will use:',
    '',
    '```sh',
    `weft catalog                   # ${components.length} components in ${groups.length} groups, one line each`,
    'weft catalog Card Table Form   # each one: props, events, slots, how it adapts, an example',
    '```',
    '',
    `Never read the catalogue file whole: it is too large. Without \`weft\`, fetch one entry, \`${SITE}/components/<slug>.json\` (\`DatePicker\` is \`date-picker\`); \`${SITE}/catalog.json\` is the whole file.`,
    '',
    '| To make | Do this |',
    '| --- | --- |',
    '| A page | Add `web/pages/<path>.vue`. Its path is the route: `index.vue` is the folder, and `[slug].vue` is one page per value its `getPaths()` returns. |',
    '| Something interactive | Write it in `web/components/<Name>.vue`, import it in the page, and render it with `<Island when="load" :component="Name" />`. Only islands run in the browser: a page without one ships no JavaScript. |',
    '| An API route | Add a handler in `api/api.go` on its `http.ServeMux`, with the whole path, such as `GET /api/items`. Everything under `/api` goes to it. |',
    '| A look | Set `"theme"` in the `"weft"` object of `package.json` to `neutral`, `lisa` or `weft`, then set `--weft-*` tokens in a stylesheet a page imports. |',
    '',
    '## 4. Rules',
    '',
    'As the app\'s `AGENTS.md` has them:',
    '',
    ...rulesOf(appAgents).map((r) => `- ${r}`),
    '',
    '## 5. Check your work',
    '',
    '```sh',
    'pnpm typecheck            # the Vue files',
    'go test -tags dev ./...   # the Go tests; -tags dev until the first weft build',
    'weft build                # the pages, then one binary named after the app',
    './<name>                  # serves on port 8080, or on $PORT',
    '```',
    '',
    '## 6. What is not built yet',
    '',
    `Do not look for these: write the part yourself, in the app, with the components that exist. ${SITE}/docs/coming-next has the plan.`,
    '',
    `- The commands ${list(commands.map((c) => `\`${c.replace(/`/g, '')}\``))}.`,
    ...NOT_YET.map((n) => `- ${n.missing}: ${n.instead}.`),
    '',
    '## More',
    '',
    `- Every doc, component and skill, a line each: ${SITE}/llms.txt`,
    `- Every doc and skill in full, in one file: ${SITE}/llms-full.txt`,
    `- Claude Code skills (${list(names)}): ${SITE}/skills`,
    '',
  ].join('\n')
}
