---
group: basics
purpose: A turning ring for work in progress of unknown length.
use: Inside something that is loading, or alone where content will appear; Button shows one itself when loading.
avoid: When the progress is known, use Progress; for the shape of content still loading, use Skeleton.
adapts:
  input: Nothing changes.
  surface: Nothing changes; without a size it scales with the text.
  width: Nothing changes.
---
It turns in the text colour. Without `label` it is decorative
(`aria-hidden`); with one it is a `status` that announces the label
politely. `size` is `sm` (16 px), `md` (20 px) or `lg` (24 px); without it
the spinner is 1.25em. Under reduced motion it turns slowly rather than
freezing, since a still ring reads as stuck.
