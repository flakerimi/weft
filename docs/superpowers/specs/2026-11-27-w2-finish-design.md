# W2 Finished: `enum` Typed as Its Values and `useParams()` Typed from the Routes

Date: 2026-11-27
Extends `2026-11-23-warp-design.md` and `2026-11-25-warp-slice-design.md`. Closes the two remaining W2 items for `0.1.0` (`docs/roadmap.md:66-67`, `site-src/content/docs/coming-next.md:29-30`).

## Intent

The Warp slice delivered `Describe`, `weft gen`, `weft routes`, and `useParams()`, leaving two type-safety seams open:

1. An `enum` field or parameter in Go (`warp:"enum=draft|live"`) is generated as `string` in `web/.weft-types/api.ts`, so a typo like `'published'` compiles and is caught only at submit or request time.
2. `useParams()` in `@weft/ui` returns `Record<string, string>`, so an island on `pages/products/[id].vue` that reads `useParams().productId` instead of `useParams().id` gets `undefined` with no compile-time check.

This design closes both seams without inventing a coupling between page files and API routes.

## Goals

- An `enum` rule in a `warp` tag (`enum=draft|live` on a string field or parameter, or `enum=1|2|3` on a number field or parameter) types the generated TypeScript interface property, path parameter, or query parameter as the union of its literal values (`'draft' | 'live'`, `1 | 2 | 3`, or with `| null` for a pointer), while keeping `schema`'s runtime `FieldRules` shape (`kind: 'string' | 'number'`) unchanged.
- `warp` allows `enum` on both text and number fields, checks each number choice at registration and validation time, and refuses two `enum` rules on one field with a single-sentence registration mistake.
- `@weft/ui`'s `useParams('/products/:id')` takes an optional route pattern from `WeftRouteParams` (populated in `.weft-types/components.d.ts` by `@weft/vite`'s `weftAutoImport()` and `weft-types` from the dynamic pages under `pages/`) and returns `{ id: string }`, so an unknown pattern or a parameter name the page does not have is a TypeScript error.
- Calling `useParams()` with no argument continues to return `Record<string, string>`.
- Calling `useParams(pattern)` at run time on a page whose `data-weft-route` differs from `pattern` returns `{}` and warns once in development (never in production) naming both patterns.

## Non-goals

- Calling an API on another origin than the pages (`setBase` remains same-origin; that stays in `coming-next.md`).
- Coercing URL parameters to numbers inside `useParams`: a URL path segment is always a string, and converting it (such as `productId(useParams('/products/:id').id)`) stays the page's job.

## Design

### 1. `enum` on text and numbers in `warp` (`warp/validate.go`)

- **Allowed kinds:** `enum` is valid on text (`string`) and numbers (any `int`, `uint` or `float` kind, or a pointer to one). On any other kind, registration reports:
  `<Type>.<Field> has the rule "enum", which is for text and numbers; remove it or make the field a string or a number.`
- **One `enum` per field:** If a field's `warp` tag has more than one `enum` rule, registration reports:
  `<Type>.<Field> has the rule "enum" more than once; list every value in one enum=a|b.`
- **Parsing items:**
  - Pipe-separated non-empty items are collected into `r.items`. If `len(r.items) == 0`, registration reports `<Type>.<Field> has "enum=" with nothing to choose from; write enum=a|b.` as today.
  - When the field is a number (`isNumber`), each item in `r.items` is parsed with `strconv.ParseFloat(item, 64)` (and checked not `NaN` or `Inf`) into `r.nums []float64`. If any item is not a finite number, registration reports:
    `<Type>.<Field> has %q; write numbers separated by | after enum=.`
- **Validating values (`checkRule`):**
  - On a string (`k == reflect.String`), `v.String()` must equal one of `r.items`.
  - On a number (`isNumberKind(k)`), `toFloat(v)` must equal one of `r.nums`.
  - On failure, both return `"Choose one of: " + strings.Join(r.items, ", ") + "."`.

### 2. `enum` in `Form` (`packages/ui/src/forms/schema.ts`) and `weft gen` (`cmd/weft/gen.go`)

- **`Form` (`schema.ts`):** `FieldRule.enum` remains `string[]` (what `weft gen` writes into `schema` from `r.Arg`). In `validateField`, when `typeof value === 'number'` and `rule.enum` is set, `rule.enum.some((item) => Number(item) === value)` is checked after `textRules`, returning `fail(\`Choose one of: \${rule.enum.join(', ')}.\`)` when none matches, so browser validation and Go validation agree on numeric enums too.
- **`weft gen` (`cmd/weft/gen.go`):**
  - A helper `enumTS(ts string, rules []ruleInfo) string` refines `ts` when `rules` contains an `enum` rule with non-empty pipe-separated values:
    - Let `nullable = strings.HasSuffix(ts, " | null")` and `base = strings.TrimSuffix(ts, " | null")`.
    - If `base == "string"`, each non-empty item `v` in `strings.Split(r.Arg, "|")` is formatted with `quote(v)` (the generator's existing single-quote string literal escaper, handling `'`, `\`, `\n`, control characters, U+2028 and U+2029) and joined with `" | "` (e.g. `'draft' | 'live'`).
    - If `base == "number"`, each non-empty item `v` in `strings.Split(r.Arg, "|")` that parses as a finite `float64` with `strconv.ParseFloat(v, 64)` is formatted with `strconv.FormatFloat(n, 'f', -1, 64)` and joined with `" | "` (e.g. `1 | 2 | 3`).
    - If `nullable` is true, `" | null"` is appended.
  - `enumTS` is used for:
    1. Interface fields in `generate` (`enumTS(f.TS, f.Rules)`).
    2. Route path parameters and query parameters in `writeRoute` (`paramInfo` carries `Rules []ruleInfo \`json:"rules"\``, and `writeRoute` uses `enumTS(p.TS, p.Rules)` and `enumTS(q.TS, q.Rules)`).
  - `fieldRules(f)` continues to compute `kind(f.TS)` from the unrefined `f.TS` (`'string'` or `'number'`), so `schema` keeps `kind: 'string', enum: ['draft', 'live']` (or `kind: 'number', enum: ['1', '2', '3']`).

### 3. `useParams()` typed from the page routes (`packages/ui/src/head/useParams.ts`, `packages/vite/src/auto-import.ts`)

- **Type registry (`packages/ui/src/head/useParams.ts`, re-exported from `packages/ui/src/index.ts`):**
  ```ts
  /**
   * Each dynamic page's route pattern (`/products/:id`, `/docs/:...rest`) mapped
   * to the parameter names it has (`{ id: string }`, `{ rest: string }`).
   * `@weft/vite`'s `weftAutoImport()` and `weft-types` augment this interface
   * in `.weft-types/components.d.ts` from `pages/`.
   */
  export interface WeftRouteParams {}

  export function useParams<P extends keyof WeftRouteParams>(route: P): WeftRouteParams[P]
  export function useParams(route?: undefined): Record<string, string>
  export function useParams(route?: string): Record<string, string>
  ```
- **Runtime check on `route` (`packages/ui/src/head/useParams.ts`):**
  - On the server (`typeof document === 'undefined'`), `useParams` returns `{}` as before.
  - In the browser, if `pattern` (`document.documentElement.dataset.weftRoute`) is missing, `useParams` warns once in development that the page was built ahead and returns `{}` as before.
  - If `route !== undefined && route !== pattern`:
    - In development (`import.meta.env.DEV`), `useParams` warns once per `${route}|${pattern}` pair:
      `useParams: asked for the params of "${route}" on a page whose route is "${pattern}"; pass "${pattern}" or call useParams() with no argument.`
    - It returns `{}` (since the address belongs to another route pattern).
- **Declaration generation (`packages/vite/src/auto-import.ts`):**
  - `routeParamEntries(pagesDir: string): { pattern: string; params: string[] }[]` walks `pagesDir` synchronously (skipping `_` and `.` entries and symlinks, matching `discoverRoutes`'s path mapping where `[name]` is `:name` and `[...name]` is `:...name` and trailing `/index` is stripped), collects every dynamic page (`pattern.includes(':')`), and sorts them by `pattern`.
  - `declaration(names: string[], routes: { pattern: string; params: string[] }[] = [])` writes `declare module '@weft/ui' { interface WeftRouteParams { ... } }` alongside `declare module 'vue' { interface GlobalComponents { ... } }` in `.weft-types/components.d.ts`:
    ```ts
    declare module '@weft/ui' {
      interface WeftRouteParams {
        '/products/:id': { id: string }
      }
    }
    ```
    When a site has no dynamic pages under `pages/`, `interface WeftRouteParams {}` is empty, so `keyof WeftRouteParams` is `never` and passing any pattern argument to `useParams('/...')` is a compile-time type error.
  - Both `writeComponentTypes(root, dts)` (`weft-types --root web`) and `weftAutoImport`'s `buildStart` and `configureServer` hooks pass `join(root, 'pages')` to `routeParamEntries`, and in `configureServer` refresh `.weft-types/components.d.ts` when a `.vue` file under `pages/` is added or unlinked.
- **Example shop (`examples/shop/web/components/ProductView.vue`):**
  - Uses `useParams('/products/:id').id` so the shop exercises the typed overload end to end.
