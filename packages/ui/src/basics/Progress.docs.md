---
group: basics
purpose: A bar showing how far a task has got, or that it is under way when unknown.
use: For uploads, imports, multi-step work and quotas, where the reader waits or watches.
avoid: For a short wait inside a control, use Spinner; for loading content, use Skeleton.
adapts:
  input: Nothing changes.
  surface: Nothing changes.
  width: Nothing changes; the bar fills the width it is given.
---
Bind the value with `v-model` (or `model-value`), from 0 to `max` (100);
values outside are clamped. `null` or no value is indeterminate: a bar that
sweeps, which stands still at full length, dimmed, under reduced motion.
`status` shows the percentage beside the bar.

It is a `progressbar` with `aria-valuemin`, `aria-valuemax` and, when
determinate, `aria-valuenow`. Name it with `label` ("Uploading report.pdf")
or an `aria-labelledby`. The bar fills along the inline direction, so from
the right in a right-to-left document.
