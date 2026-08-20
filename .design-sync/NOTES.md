# design-sync notes — superplane / web_src

Durable findings for future syncs. Read this and `config.json` before doing anything.

## The big one: web_src is an app, not a component library

`web_src/package.json` is `private`, version `0.0.0`, with **no `main`/`module`/`exports`**
and no library `dist/` — `npm run build` emits a static *site* into
`../pkg/web/assets/dist`. The storybook shape needs a real entry to bundle into
`window.Superplane` (`synthEntry` is package-shape only), so this repo needs three
repo-side files that would not exist for a normal design system:

| File | Why it exists |
|---|---|
| `web_src/ds-entry.ts` | The library entry the package lacks — a barrel of the DS surface. |
| `web_src/ds-preview-provider.tsx` | `cfg.provider` chain, distilled from `.storybook/preview.tsx`. |
| `web_src/ds-styles.css` | Compiled Tailwind, copied out of the reference storybook build. |

`web_src/package.json` also gained `"types": "ds-entry.ts"` (see below).

## [GENERAL] Fixes that cost real debugging — do not rediscover these

- **The entry must live INSIDE `web_src`.** The repo root has no `package.json`, so
  with the barrel at `.design-sync/ds-entry.ts` the converter's walk-up went past the
  root and inferred `/Users/<user>` as PKG_DIR. Symptoms: `! tsconfig: … not found`,
  `[DTS_REACT] @types/react not found`, and every `@/…` import unresolved. `cfgPath()`
  resolves `cfg.tsconfig`/`cfg.cssEntry` against **PKG_DIR**, not cwd — that is why
  `cfg.tsconfig` is `"tsconfig.app.json"` and not `"web_src/tsconfig.app.json"`.
- **The barrel must use RELATIVE imports, not `@/`.** esbuild resolves the alias via
  the converter's tsconfig-paths plugin, but **ts-morph does not** — component
  discovery reads the types tree, so with `@/` re-exports it found
  `exported PascalCase symbols: 0`, dropped all titles as `[TITLE_UNMAPPED]`, and
  emitted **0 component previews** while still exiting 0. Relative imports fixed it.
  A build that "succeeds" with 0 previews is this bug.
- **`"types": "ds-entry.ts"` in `web_src/package.json` is load-bearing.**
  `findTypesRoot()` reads `pkgJson.types || pkgJson.typings` and has **no cfg
  override**. Without it there is no types root, so `exportedNames()` returns empty
  and every storybook title is dropped. Inert for a private package.
- **MSW breaks bundled decorators → `cfg.provider` is mandatory.**
  `.storybook/preview.tsx` calls `initialize()` from `msw-storybook-addon`; the
  converter's inert msw stub makes `worker.start()` return a non-thenable, so **all**
  previews died with `TypeError: worker.start(...).then is not a function`. Fix is the
  documented one: set `cfg.provider`, which skips decorator bundling. `DsProvider`
  reproduces QueryClientProvider + ThemeContext + the Material Symbols `<link>`.
- **`cfg.cssEntry` is required — the storybook CSS fallback will NOT fire.** The
  scrape only triggers when `_ds_bundle.css` is missing or `<500B`; esbuild produced
  1729 B of genuine-but-incomplete CSS, which is over the threshold, so the ~545 KB
  compiled Tailwind sheet was never picked up and designs would have shipped
  essentially unstyled. **The compare oracle cannot catch this** — both panels would
  look wrong identically. `ds-styles.css` is that sheet, copied from
  `.design-sync/sb-reference/assets/` (largest `*.css`).
- **`@tanstack/react-query` must be shimmed, and its exports put on the global.**
  Story previews bundle their own copy by default (`_preview/SettingsTab.js` was
  1.7 MB), producing a second react-query instance whose context `DsProvider` never
  set — `Error: No QueryClient set` and `root empty` on every component using a query
  hook. `cfg.storyImports.shim: ["@tanstack/react-query"]` + re-exporting the hooks
  from the barrel gives one instance. Also shrinks every preview substantially.

## Scope: ~49 of ~72 components — forced by the 12 MB upload cap

The full story surface bundles to **31.4 MB** (`[FILE_TOO_LARGE]`, cap is 12 MB).
`web_src` is an application, so page-level components pull in the whole app;
mermaid + d3 + cytoscape + katex dominate, reached transitively via
`components/AgentSidebar/widgets/RichMessage` and `pages/app/Markdown.tsx` — so
excluding `MermaidWidget` from the barrel alone changes **nothing**.

Scoped to `src/ui/**` + `src/components/ui/**` the bundle is **~5.9 MB**. The dropped
titles (`Charts`, `Mermaid`, `NodeChips`, `RichMessage`, `RubricWidget`, `RunChips`,
`AutoCompleteInput`, `CanvasCard`, `Timestamp`, `Board`, …) appear as
`[TITLE_UNMAPPED]` — that warning is **expected here**, not a regression.

## Two component families (collision policy)

The repo has two parallel implementations of the same primitives:

- `src/components/ui/*` — 18 files, **zero stories**, but what production imports
  (button 132 files, input 62, select 38, tooltip 31).
- `src/ui/*` — carries the "shadcn Primitives" stories, but for those primitives is
  barely used in the app.

One flat global means one winner per name. Policy (user decision, 2026-07-28):
**follow app usage** — `src/components/ui/*` wins everything **except `Checkbox`**,
pinned to `src/ui/checkbox` (10 app files vs 5). Losers are listed in
`cfg.storyImports.bundle` so their stories compile their own copy into the preview
instead of silently rendering the winner. `src/components/ui/*` ships **unverified**
(floor cards) — it has no stories to grade against.

## Rebuilding the barrel (`web_src/ds-entry.ts`) — full recipe

Generator lives in the session scratchpad, not the repo; if it is gone, the steps are:

1. Surface = every local component a story imports, **minus** fixtures/mocks/harnesses
   (`__fixtures__ __mocks__ __stories__ storybooks/fixtures test/ pages/home/factories
   handlers`) — they are not components and several pull `.md`/`.yaml?raw` assets the
   component bundle has no loader for — **plus** all of `src/components/ui/*`.
2. Resolve collisions per the policy above (`components/ui` wins; Checkbox pinned).
3. **Narrow to `@/ui/**` + `@/components/ui/**`** — the 12 MB cap.
4. **Rewrite `@/` → `./src/`** (and `@/canvas/` → `./src/pages/canvas/`).
5. Append `export { DsProvider } from "./ds-preview-provider";` and the
   `@tanstack/react-query` re-exports (QueryClient, QueryClientProvider,
   useInfiniteQuery, useMutation, useQueries, useQuery, useQueryClient).
6. Ensure `web_src/package.json` has `"types": "ds-entry.ts"`.
7. Re-copy `ds-styles.css` from the freshly built `sb-reference/assets/` (largest css).

## Re-sync risks

- **`ds-preview-provider.tsx` drifts from `.storybook/preview.tsx` silently.** It is a
  hand-distilled copy. If preview.tsx gains a provider, previews keep rendering — just
  without that context — and carried-forward grades will not catch it. Diff the two on
  every sync.
- **`ds-styles.css` is a stale copy of a build artifact.** It is the compiled Tailwind
  sheet from `sb-reference/assets/*.css` (hashed filename, so it must be re-copied by
  hand). Any styling change is invisible to designs until it is re-copied. Rebuild
  `sb-reference` **and** re-copy this file together.
- **`web_src/src/api-client/` is generated and gitignored.** It is a build prerequisite
  (`make openapi.web.client.gen`, or copy `api/swagger/superplane.swagger.json` from a
  built checkout and run `npm run generate:api`). A fresh clone cannot build without it.
- **Material Symbols is a runtime CDN font** (`fonts.googleapis.com`), injected by
  `DsProvider`. Not vendored, so icon glyphs depend on network egress at render time.
  `[FONT_MISSING]` is invisible to the compare oracle — check icons visually.
- **The 12 MB cap is close to load-bearing.** At ~5.9 MB there is headroom, but adding
  any page/canvas/agent component re-introduces the heavy dependency graph. Measure
  before widening scope.
- **`docs: 0/N components matched`** — no doc bodies found, so `.prompt.md` files are
  thin. If docs exist somewhere, set `cfg.docsMap`.
- **`[RENDER_THIN]` on Tooltip** is expected (overlay variants render identically);
  it wants a hand-authored `.design-sync/previews/Tooltip.tsx`, not a config knob.

## History

- 2026-07-28: first sync attempt reached a clean build+validate (48/49 render clean) on
  commit `ca2c91203`, then the worktree was deliberately discarded before any upload.
  Restored on `main` @ `db7fafa0e`; the reference storybook and roster were rebuilt
  because `main` had moved (new stories, e.g. `AgentSuggestionsHoverCard`).
- The Claude Design project `90cd76d1-c41a-4ead-8e90-83c80d715e66`
  ("Superplane Design System") has existed since the first attempt and was **empty and
  un-anchored** at restore time — the documented safe state.
