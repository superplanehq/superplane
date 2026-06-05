import { buildEnv, compileTemplate, evalTemplate } from "./widget/celExpr";

/**
 * Cheap predicate: does this string contain any `{{ ... }}` template segment?
 * Used to short-circuit the CEL compile pipeline for vanilla markdown so
 * static panels (the majority case) stay zero-cost.
 */
const TEMPLATE_RE = /\{\{[\s\S]*?\}\}/;

/** Stringify a CEL-evaluated value for inline insertion into markdown. */
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
