import type { ContentConfig } from '@weft/vite'

// content/docs/ is served at /docs. Its pages need only the fields every
// collection has: title, description and order (the sidebar's order).
export default {
  docs: {},
} satisfies ContentConfig
