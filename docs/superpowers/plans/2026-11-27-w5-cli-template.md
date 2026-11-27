# W5 CLI & Template (`weft scaffold` and App Template Frame) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the CLI and template side of Warp phase W5 for `0.1.0` (owned by Fable): move `templates/app/` to the shop's frame (`data`, `auth`, `hello`, login and setup pages, random `AUTH_SECRET` in `.env`, SQLite driver in `main.go` and `go.mod`), and implement `weft scaffold <name> <field[:type[:modifier...]]> ... [--plural <word>] [--api]`.

**Architecture:** `cmd/weft/create.go` generates a 48-byte base64url `AuthSecret` for `.env.tmpl` while `.env.sample.tmpl` keeps a placeholder (and prints a note to set `AUTH_OWNER_EMAIL` in `.env`); `templates/app/` includes `data.Module()`, `auth.Module()` and `hello.Module()`, `_ "modernc.org/sqlite"`, and the login/setup pages and session helpers; `cmd/weft/scaffold.go` parses explicit field types (`string`, `text`, `email`, `url`, `int`, `float`, `bool`, `date`, `datetime`, `enum=a,b`) and modifiers (`?`, `required`, `unique`, `min=N`, `max=N`) with `--plural <word>`, refuses `|` or `(` in enum choices and unsupported/future types with one sentence, writes `app/<plural>/{model,routes,<plural>_test}.go`, edits `app/modules.go` via `go/ast` and `go/format`, writes `web/pages/<plural>/{index,[id]}.vue` and `web/components/{<Plural>Table,<Singular>View}.vue` using `DataTable` (unless `--api`), and runs `weft gen`.

**Tech Stack:** Go 1.26 standard library only in `cmd/weft/`; `modernc.org/sqlite` in `templates/app/go.mod.tmpl`; Vue `3.6.0-rc.16` Vapor SFC templates.

**Spec:** `docs/superpowers/specs/2026-11-27-w5-design.md`.

## Global Constraints

- **Inherited:** every rule in `AGENTS.md` and `docs/superpowers/specs/2026-11-27-w5-design.md`.
- **Standard library only** in `cmd/weft/` and `warp/`.
- **One-sentence errors** that name the thing and say what to do.
- **Explicit types only in `weft scaffold`:** no inference from field names; a field without `:type` is `string`.
- **Never overwrite developer files:** `weft scaffold` checks all target paths and `app/modules.go` before writing anything.
- **Test first:** write the failing test, see it fail for the expected reason, implement, and run the full verification suite before committing.
- **No `Co-Authored-By` lines** in commits.

## Review Focus

1. **Template frame (`weft create`):** A created app has `data.Module()`, `auth.Module()` and `hello.Module()`; `.env` has a fresh 48-byte base64url `AUTH_SECRET` (different on each `weft create`) and `.env.sample` has a placeholder; `GET /api/hello` still answers 200 without a session; `/login` and `/setup` build and guard their password forms before hydration; `go test ./...`, `pnpm typecheck` and `weft build` pass in the created app.
2. **`weft scaffold` argument parsing and refusals:** Refuses missing fields, `id`/`created_at`/`updated_at`, duplicate fields, `required` on `bool` or nullable (`?`) types, `unique` on `bool`/`text`/`date`/`datetime`, relations (`0.3.0`), attachments (`0.4.0`), `json`/`translated` (`0.6.0`), and existing `app/<plural>` or page files before writing anything.
3. **`weft scaffold` generated code and AST edit:** `app/<plural>/{model,routes,<plural>_test}.go` are valid, `gofmt`-clean Go; `data` tags follow the type rule (every scalar except `text` sortable, text searchable, `enum` and `bool` filterable); `app/modules.go` has `<plural>.Module()` added cleanly via `go/ast` + `go/format`; permissions use `data.PerAction(auth.Per("<plural>"))` when `auth` is in `app/modules.go` (and none, with a one-line notice, when `auth` is absent); `--api` writes only the Go files; without `--api`, the generated pages and islands use `DataTable` and `useParams('/<plural>/:id')`, `weft gen` runs at the end, and `go test ./...`, `pnpm typecheck` and `weft build` pass.

---

### Task 1: Move `templates/app/` and `cmd/weft/create.go` to the shop's frame

**Files:** Modify `cmd/weft/create.go`, `cmd/weft/create_test.go`, `templates/app/go.mod.tmpl`, `templates/app/main.go.tmpl`, `templates/app/app/modules.go.tmpl`, `templates/app/app/hello/hello_test.go.tmpl`, `templates/app/env.tmpl`, `templates/app/env.sample.tmpl`, `templates/app/gitignore.tmpl`; add `templates/app/web/answers.ts`, `templates/app/web/session.ts`, `templates/app/web/pages/login.vue`, `templates/app/web/pages/setup.vue`, `templates/app/web/components/LoginForm.vue`, `templates/app/web/components/SetupForm.vue`.

- [ ] **Step 1: Write failing tests** in `cmd/weft/create_test.go` checking that `weft create` writes a 48-byte base64url `AUTH_SECRET` in `.env` (at least 64 characters, different across two runs), `DB_DRIVER=sqlite`, `DB_URL=file:data/<name>.db`, `modernc.org/sqlite` in `go.mod` and `main.go`, `data.Module()` and `auth.Module()` before `hello.Module()` in `app/modules.go`, and the login and setup pages.
- [ ] **Step 2: Run tests to verify failure:** `go test -short ./cmd/weft -run TestCreate`.
- [ ] **Step 3: Implement** `AuthSecret` in `cmd/weft/create.go` and update `templates/app/`.
- [ ] **Step 4: Run tests to verify pass.**

### Task 2: `weft scaffold` parser, pluralizer, Go generator, AST `app/modules.go` editor, and Vue page generator

**Files:** Create `cmd/weft/scaffold.go`, `cmd/weft/scaffold_test.go`; modify `cmd/weft/main.go`.

- [ ] **Step 1: Write failing unit tests** in `cmd/weft/scaffold_test.go` for `pluralOf`, field/type/modifier parsing, milestone refusal sentences (`belongs_to` → `0.3.0`, `image` → `0.4.0`, `json` → `0.6.0`), pre-flight collision refusals, Go model/routes/test generation, `app/modules.go` AST editing (with and without `auth`), and Vue page/component generation (`--api` vs full).
- [ ] **Step 2: Run tests to verify failure:** `go test -short ./cmd/weft -run TestScaffold`.
- [ ] **Step 3: Implement** `cmd/weft/scaffold.go` and register `"scaffold"` in `cmd/weft/main.go`.
- [ ] **Step 4: Run unit and integration tests** (`go test ./cmd/weft -run TestScaffold`) to verify that scaffolding a resource in a created app passes `go test ./...`, `pnpm typecheck` and `weft build`.

### Task 3: Update docs and skills for `weft scaffold`, `DataTable`, `EntityDialog` and the new template

**Files:** Modify `site-src/content/docs/getting-started/create-an-app.md`, `site-src/content/docs/reference/cli.md`, `site-src/content/docs/coming-next.md`, `skills/weft-go/SKILL.md`, `skills/weft-site/SKILL.md`, `docs/roadmap.md`.

- [ ] **Step 1: Update documentation** to describe `weft create` with `data` and `auth`, `weft scaffold`, `DataTable` and `EntityDialog`, and remove W5 from `coming-next.md`.
- [ ] **Step 2: Build the site and run doc/budget tests:** `scripts/site.sh build && go test ./site-src/...`.
