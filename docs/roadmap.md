# The road to Weft 1.0.0

Written 2026-11-27, at 0.0.10. It orders the work the two designs plan
(`docs/superpowers/specs/2026-11-20-weft-design.md`, Weft; and
`2026-11-23-warp-design.md`, Warp, which replaces Weft's phases 4b and 5) into
releases, and says what 1.0.0 promises. What is not built yet is listed, with
its phase, in `site-src/content/docs/coming-next.md`; this file says in which
release it lands. The order is a proposal by Claude (Fable), to be corrected by
the owner; each milestone gets its own design or plan before it is built.

## Where 0.0.10 stands

- **Built:** `@weft/ui` (76 Vapor components, adaptive, with a gallery and a
  catalogue), the page build with islands, Markdown content and run-time pages,
  `warp.Serve` from memory, the `weft` command (`create`, `dev`, `build`, `gen`,
  `routes`, `unpack`, `upgrade`, `catalog`, `update`), and Warp's first slice:
  `warp.App` and modules, typed routes, validation, `Describe` and the typed
  client, `warp/data` on SQLite, `warp/auth` with one owner, `warptest`, and an
  example shop that uses all of it.
- **Runs on:** macOS, Linux and Windows (the full suite passes on all three on
  GitHub's runners; Windows installs from 0.0.21).
- **Distributed:** from weft.base.al (installer, releases with checksums,
  `weft update`); the repository is public since 2026-11-26.
- **Not stable:** every 0.0.x may change any name. Vue is pinned to
  the latest Vue 3.6 build because Vapor is in 3.6, which is not yet a stable
  release; `scripts/vue-update.sh` moves the pins before each release.

## What 1.0.0 promises

1.0.0 is the release after which an app made with Weft keeps working:

1. **Semantic versioning.** Within 1.x no exported Go name, component prop,
   event, slot, CLI command, flag, environment variable, error code or
   sentence a test may match changes incompatibly; what goes is deprecated for
   at least one minor release first, with the deprecation said at build time.
2. **`weft upgrade` from any 1.x to any later 1.x** works on the example apps
   and the template, tested in CI.
3. **Three platforms**, each on amd64 and arm64: macOS, Linux and Windows, with
   the whole suite passing on every push.
4. **On pinned, tested dependencies:** Go's current stable release, Node's
   active LTS, and the latest Vue 3.6 build, pinned exactly and tested. By
   the owner's decision of 2026-11-28, 1.0.0 does not wait for Vue 3.6 to be
   stable: it ships on the pinned release candidate, and moving to Vue 3.6
   stable is an ordinary 1.x update when Vue publishes it.
5. **Reviewed:** a security review of `warp`, `warp/auth`, `warp/data` and the
   page server by someone who did not build them; an accessibility audit of
   every component (keyboard and screen reader, both input modes).
6. **Documented:** every exported Go name, every component, every command and
   every error a user can meet, with a guide per module; the docs describe the
   code as it is, checked by tests.

## Milestones

Each milestone ends with a release from `main`, the suite green on all
platforms it claims, and the docs saying what it brought.

### 0.1.0: the first public release

What a newcomer meets first works, on every platform.

- **Windows supported (done, 0.0.21):** the release workflow's Windows tests
  pass (0.0.19); `weft dev` stops its process trees with a Job Object
  (0.0.19); the installer has a PowerShell form (`/install.ps1`), and
  `scripts/release.sh` builds all six archives by default (0.0.21); the docs
  say Windows is supported. The suite runs on all three platforms on every push to `main`
  (done: `ci.yml` runs Linux on every push and pull request, macOS and
  Windows on every push to `main`).
- **The Go module from GitHub (done, 0.0.12):** with the repository public, an app's `go.mod`
  can require `github.com/flakerimi/weft` at the release's tag instead of
  replacing it with `./.weft/go`; tag each release (`vX.Y.Z`) and attach the
  six archives to a public GitHub Release made from the same build the site
  serves.
- **W2 finished (done):** an `enum` typed as its values (comparing or assigning
  a value outside the list is now a TypeScript error; an optional non-pointer
  struct field includes `''` or `0`), `useParams()` typed from the routes.
- **W5, scaffolding and app blocks (done):** `weft scaffold`, `DataTable`
  (server paging, sorting, filtering over the list shape) and `EntityDialog`;
  the app template takes the shop's frame (data, the login and its pages, and
  `hello`, with no starter resource).
- **Landing blocks (done, 0.0.13)** (Weft phase 7): `Hero`, `FeatureGrid`, `Pricing`,
  `Testimonials`, `Faq`, `Cta`.
- **The benchmark** run on a quiet machine with every series interleaved, and
  Weft not behind on requests a second and time to first byte (Weft phase 7's
  "beats Astro on every row" stays the aim for the rows Weft can win: bytes
  and LCP are a choice of what an island ships, said plainly where it loses).
- **A changelog**, `CHANGELOG.md` (done, 0.0.18), and branch protection on
  `main` (done: no force push or deletion, admins included).

### 0.2.0: accounts (W4, with the middleware it needs)

- Registration, invitations and password reset, with the `mail` module (log
  and SMTP drivers) (done: Tracks A5 and B1).
- Roles and permissions beyond the owner; a session table on the server, so a
  user can sign out everywhere and one session can be ended (done: A2, A4, A6).
- `useSession()` in `@weft/ui`, `perm` on `Button` and row actions, and a
  route guard in Vue (done: C1 to C3; the template's own `web/session.ts` is
  gone, and its pages, and the ones `weft scaffold` writes, use them: E1 to E4).
- Date and time fields in `EntityDialog` and the pages `weft scaffold` writes
  (done: D1 to D4).
- From W7, brought forward because accounts need them: the `middleware` module's
  security headers, request logging, a rate limit and API keys; body-read and
  idle timeouts on the server (done: B2 to B6, and API keys in `auth`, A7).
- The login's open limits closed: a per-account count across addresses (done:
  A3).

### 0.3.0: data beyond SQLite (W3)

- PostgreSQL, then MySQL, behind the same `Store`, each tested in CI against a
  real server.
- Relations (`belongs_to`, `has_many`, `many_to_many`) and soft delete.
- `weft db`: how the database differs from the app's records.
- Migrations that can rename and drop, written out and reviewed, beside the
  additive ones Warp makes itself.

### 0.4.0: files and live data (W6)

- The `storage` module: attachments on a record's `image` and `file` fields,
  served under `/api/files/`, with `FileUpload` bound to the field.
- `events` and `live`: server-sent events at `/api/live`, read with `useLive`
  and `useLiveSeries`; shutdown that ends open streams at once.
- Charts that take live data: `LineChart`, `AreaChart`, `BarChart`,
  `DonutChart`, `Sparkline`.

### 0.5.0: app routes and the agent kit

- App routes (Weft phase 4c): a client-side router and `layouts/`, for an app
  whose pages switch without loading a document.
- The agent kit (Weft phase 6): `@weft/eslint-plugin`, `weft check` and
  `weft mcp`, so an agent is told when it writes markup Weft has a component for.

### 0.6.0: the rest of the modules (W7) and site features (6b)

- `jobs` (work on a schedule), `i18n` (translated fields), `docs` (an OpenAPI
  document at `/api/openapi.json`).
- `<Image>`, actions, environment values, a development overlay and
  `weft deploy`.

### 0.9.x: the freeze

No new features; only what 1.0.0 promises.

- Every exported name, prop and command reviewed once more and renamed now if
  it is wrong, since after 1.0.0 it cannot be.
- The security review and the accessibility audit, and their findings fixed.
- Vue: the latest 3.6 build at each release (`scripts/vue-update.sh`, then the
  whole suite); 3.6 stable comes in the release after Vue publishes it (the
  owner's decision of 2026-11-28: 1.0.0 does not wait for it).
- `weft upgrade` tested from 0.9.0 and across the 0.9.x releases.
- Release candidates (`1.0.0-rc.N`) used to build the example apps and the
  site, and one outside app, before 1.0.0 is tagged.

### 1.0.0

Tagged when everything under "What 1.0.0 promises" holds, from a release
candidate that changed nothing but its version.

## How the work is shared

Work is coordinated in `coord.md` at the repository's root (not in git):
who works where, entries addressed by name, and a table of who holds what.
Each milestone is split into pieces that touch different folders, so two
agents can build at once in their own worktrees, each piece reviewed by an
agent that did not build it, by running the code. The owner's standing
orders hold for everyone: finished work whose review and suite pass is
merged, pushed and published to weft.base.al without asking; nothing is
deleted without the owner's yes; no `Co-Authored-By` lines.
