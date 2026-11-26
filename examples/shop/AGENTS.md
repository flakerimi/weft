# AGENTS.md

shop is a Weft app: Go serves the API (a `warp.App`, from `github.com/flakerimi/weft/warp`, with a list of modules) and the pages the build writes, in one binary. Weft is unpacked in `.weft/`, which `weft` writes and you do not edit. It is the example app in the Weft repository; run it with `scripts/shop.sh` from the repository's root.

## Commands

```sh
weft dev        # run the app, rebuilding pages and Go as you save
weft build      # build the pages, then the binary
weft gen        # write web/.weft-types/api.ts from the Go types
weft routes     # every route, with its module, name and permission
weft catalog    # the components; weft catalog <Name> for one, --find <word> to search
go test -tags dev ./...   # the Go tests
pnpm typecheck  # check the Vue files against the generated client
```

Go commands need `-tags dev` until the first `weft build`: without it `dist.go` embeds `web/dist`, which does not exist yet.

## Layout

| Path | What it is |
| --- | --- |
| `main.go`, `dist.go`, `dist_dev.go` | The server. `main.go` makes the `warp.App`, imports the SQLite driver and serves it; `dist.go` embeds `web/dist`; with `-tags dev` `dist_dev.go` embeds nothing |
| `app/modules.go` | The one list of the app's modules: `data`, `auth`, `products` |
| `app/products/` | The products: the model, its five routes through `data.Resource`, and the story test |
| `.env`, `.env.sample` | The settings; `.env` is not committed, `.env.sample` is and says what each one is |
| `web/pages/` | One page per `.vue` file; its path is the route. `products/[id].vue` is built once and served for every id |
| `web/components/` | The islands (`LoginForm`, `SetupForm`, `ProductsTable`, `ProductView`, `SessionBar`) and `ShopHeader` |
| `web/session.ts` | `useSession()`: who is signed in, asked once per page |
| `web/.weft-types/api.ts` | The typed client `weft gen` writes; every call to the server goes through it |

## Rules

- Call the server only through `api` from `web/.weft-types/api.ts`; never write `fetch` or a URL under `/api`.
- Use a Weft component wherever one exists: never write a raw `<button>`, `<input>`, `<select>` or `<dialog>`.
- Before you write markup, run `weft catalog` for the list of components, then `weft catalog <Name>` for each one you use. Never read `.weft/catalog/components.json` whole: it is too large.
- Every component is `<script setup vapor lang="ts">`. Never use the Options API, `getCurrentInstance` or VNode render functions, nor another UI library.
- Styles use only `--weft-*` tokens, each with a fallback.
- Go: use the standard library and `github.com/flakerimi/weft/warp`; the one other dependency is the database driver.
- Do not edit `.weft/` or `web/.weft-types/`: `weft` writes them.

More for agents: https://weft.base.al/index.md and https://weft.base.al/llms.txt
