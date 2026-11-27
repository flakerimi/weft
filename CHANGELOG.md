# Changelog

What each release of Weft brought. Every 0.0.x release may change any name;
from 1.0.0 on, what is removed is deprecated first (see `docs/roadmap.md`).
A release is published at weft.base.al and as a GitHub Release of the same
name, with the same archives and `SHA256SUMS`, from 0.0.12 on. Releases before
0.0.12 have no GitHub Release: they were published at weft.base.al, and
`site/releases.txt` lists them (only 0.0.1 and 0.0.2 have a tag).

## Unreleased

- On a phone (under 640 px) the landing's phone picture follows the install
  command closely, 64 px below the copy, over a shorter band of the weave
  (700 px), in place of a 180 px gap over an 816 px band.

## 0.0.19

- On Windows, `weft dev` and the commands that run child processes put each
  child in a Job Object (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`) while suspended,
  so stopping a child or ending `weft dev` (even with `TerminateProcess`) ends
  its whole process tree.
- On Windows, `weft dev` gives the Go server and the page server a stdin pipe
  each, so the page server no longer waits behind the Go server's read and
  starts; a Job Object that cannot be made is said in its own sentence.
- The license is held by Flakerim Ismani and Basecode LLC, and the site's
  footer links basecode.llc over https.

## 0.0.18

- `weft scaffold <name> <field[:type[:modifier...]]>...` writes a resource:
  its Go package (model, routes and a story test, added to `app/modules.go`)
  and a `DataTable` page with a record page, or the API alone with `--api`,
  then runs `weft gen`. A field's type is written, never guessed (`string`
  when none is given), `enum=a,b` among them; the modifiers are `nullable`,
  `required`, `unique`, `min=` and `max=`; the plural can be given with
  `--plural`. What it cannot write is refused in one sentence, writing
  nothing. `date` and `datetime` fields need `--api` until 0.2.0.
- The app `weft create` makes is in the example shop's shape: `data`, `auth`
  and `hello`, the login and setup pages, a fresh `AUTH_SECRET` per app, and
  the SQLite driver.
- A fresh app's `go test ./...` builds before the first `weft build`, from
  the page build's empty marker in `web/dist`; a binary built before
  `weft build` refuses to start instead of serving nothing.
- A `Footer` in an island keeps its width through hydration; `useWidth`
  reads a stored width back only inside an island.
- The CSP docs collect the speculation rules' hash with the pre-paint ones;
  `'inline-speculation-rules'` is not offered, since Chrome ignores it beside
  a hash.
- An integer `enum=` choice must be written as its type prints it (`+5`, `-0`,
  `05` and `-05` are refused), and an unsigned one out of range is told its
  range.
- A page counts as run-time only from `export const runtime = true` or a
  `const` exported as `runtime`.

## 0.0.17

- `DataTable`: a resource's list with server paging, sorting, a debounced
  search and filters, creating and editing through `EntityDialog` and
  deleting through `ConfirmDialog`, with focus kept on a row action after a
  delete.
- `EntityDialog`: controls made from a schema, `fields` as an ordered
  `EntityFieldConfig[]`, only changed fields sent on update, and server errors
  shown on their fields or in one `Alert` read out to screen readers.
- `Table` keeps its search debounce with an inline `fetch`, and `refresh()`
  settles when a newer request supersedes it. `Form` focuses the first
  invalid field once it is no longer busy.
- `scripts/publish-site.sh` and `rollback-site.sh` refuse `--dry-run` anywhere
  but first, an option they do not know, and a second argument;
  `github-release.sh` the same past its optional folder; each before anything
  is built, tagged or pushed.

## 0.0.16

- The older width-adaptive components (`Stepper`, `Timeline`, `Tabs` and
  others) are measured before paint and keep their width through hydration;
  every width measure reads one box, and `useViewportWidth()` sees the
  viewport as its media query does.
- Text on a tint of the text colour is drawn in the full text colour; the
  selected pill tab is marked by its tone, or a ring in the neutral tone.
- The page dev server's reload waits at most a second.

## 0.0.15

- An `enum` field is typed as the union of its values in the generated
  client, and `useParams('/pattern')` is typed from the routes in `pages/`.

## 0.0.14

- `useWidth`'s hydration lookup is keyed by `useId`, and the docs say how to
  collect the CSP hashes of several pre-paint scripts.

## 0.0.13

- Landing blocks: `Hero`, `FeatureGrid`, `Pricing`, `Testimonials`, `Faq` and
  `Cta`.
- `useWidth` measures before the first paint, with one page-wide pre-paint
  script, so width-adaptive components adapt on static pages.

## 0.0.12

- A publish tags its commit `v<version>` and makes the GitHub Release with
  the archives and `SHA256SUMS` the site serves; an app's `go.mod` requires
  `github.com/flakerimi/weft` at the release's tag, from GitHub.
