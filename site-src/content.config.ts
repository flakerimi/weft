import type { ContentConfig } from '@weft/vite'

// content/docs/ is served at /docs. Its pages need only the fields every
// collection has: title, description and order (the sidebar's order).
// The skills are the repository's skills/<name>/SKILL.md, as Claude Code
// reads them: each named by `name`, with no other field than its
// description. They are served at /skills/<name>.
export default {
  docs: {},
  skills: { dir: '../skills', index: 'SKILL.md', titleField: 'name' },
} satisfies ContentConfig
