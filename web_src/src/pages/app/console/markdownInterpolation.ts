import { buildEnv, compileTemplate, evalTemplate } from "./widget/celExpr";

/**
 * Cheap predicate: does this string contain any `{{ ... }}` template segment?
 * Used to short-circuit the CEL compile pipeline for vanilla markdown so
 * static panels (the majority case) stay zero-cost.
 */
const TEMPLATE_RE = /\{\{[\s\S]*?\}\}/;

/**
 * Matches a `$["Node"]` run-node reference (not a bare `$`, so currency
 * literals like `"R$"` don't trip it). Mirrors `RUN_NODE_REF_RE` in
 * `useWidgetData.ts` / `useMarkdownVariables.ts`.
 */
const RUN_NODE_REF_RE = /\$\s*\[/;

/**
 * Stringify a CEL-evaluated value for inline insertion into markdown / HTML.
 *
 * Arrays (and other non-scalar objects) fall back to JSON so a stray
 * `{{ tags }}` or `{{ row }}` reference shows something inspectable
 * (`["a","b"]`) instead of a silently flattened blob or `[object Object]`.
 * Authors who want to splice list elements into the output do so explicitly
 * with the `join(list, sep)` builtin — `join(list, "")` for seamless fragment
 * concatenation (e.g. `{{ join(tags.map(t, "<p>" + t.name + "</p>"), "") }}`)
 * or any other separator they choose.
 */
function stringifyMarkdownValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

/**
 * Interpolate `{{ name.field }}` (and the related `{{ name.$["Node"].data.x }}`
 * canvas-style references) inside a markdown body or title. `vars` is the map
 * produced by `useMarkdownVariables` and is merged onto the CEL globals via
 * `buildEnv`. When the input has no `{{ }}` segments it is returned verbatim
 * to keep the static-panel render path allocation-free.
 *
 * Errors and unresolved values render as empty strings — matching how
 * `evalTemplate` already degrades, and avoiding noisy stack traces inside
 * rendered markdown.
 */
export function interpolateMarkdownTemplate(input: string | undefined, vars: Record<string, unknown>): string {
  if (!input) return "";
  if (!TEMPLATE_RE.test(input)) return input;
  const template = compileTemplate(input);
  if (!template.hasExpr) return input;
  const env = buildEnv(vars);
  return evalTemplate(template, {}, env, stringifyMarkdownValue);
}

/**
 * Whether the input contains at least one `{{ ... }}` expression segment that
 * depends on the resolved variable map. Callers use this to decide whether a
 * still-loading variable map (e.g. per-run executions backing `{{ run.$[...] }}`)
 * would interpolate to empty fields — in which case they should hold a loading
 * state instead of rendering a half-resolved string. Plain text with no
 * templates is stable regardless of variable loading, so it returns `false`.
 */
export function markdownTemplateHasExpressions(input: string | undefined): boolean {
  if (!input) return false;
  if (!TEMPLATE_RE.test(input)) return false;
  return compileTemplate(input).hasExpr;
}

/**
 * Whether the template text references a run node's output via `$["Node"]`.
 * Used to gate the per-run execution side-load (and the matching loading
 * state): text without a `$[` reference resolves fully without those
 * executions, so it should never wait on them.
 */
export function markdownTemplateReferencesRunNode(input: string | undefined): boolean {
  if (!input) return false;
  return RUN_NODE_REF_RE.test(input);
}

/**
 * Decide whether a piece of templated markdown (a title or body) should be
 * held in a loading state, given the two-phase loading exposed by
 * `useMarkdownVariables`:
 *
 *  - `baseLoading` — the memory / run queries every variable depends on. While
 *    these are in flight any templated text is unresolved, so gate it.
 *  - `sideloadLoading` — the per-run execution side-load that only backs
 *    `$["Node"]` references. Text that doesn't reference a run node resolves
 *    fully without it, so it must NOT be gated on this phase. Otherwise a
 *    title like `{{ run.status }}` would flash the panel id while an unrelated
 *    body's `$[...]` executions settle.
 *
 * Static text (no `{{ }}` expressions) is always stable and never gated.
 */
export function markdownTextIsLoading(
  input: string | undefined,
  baseLoading: boolean,
  sideloadLoading: boolean,
): boolean {
  if (!markdownTemplateHasExpressions(input)) return false;
  if (baseLoading) return true;
  return sideloadLoading && markdownTemplateReferencesRunNode(input);
}
