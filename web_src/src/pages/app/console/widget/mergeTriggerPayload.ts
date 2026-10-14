import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

import { buildConsoleTriggerParameters } from "../consoleTriggerParameters";
import { buildEnv, compileTemplate, evalTemplate, DOLLAR_REWRITE_IDENTIFIER } from "./celExpr";
import { deepMergeObjects, setNestedString } from "./nestedPayload";

/**
 * Resolve the default payload when an action does not specify explicit
 * payloadTemplates. Prefers `row.payload` when present as an object (common for
 * run/execution/event rows), or falls back to user-defined fields on `row`
 * (stripping internal framework and metadata properties).
 */
export function resolveDefaultRowPayload(row: Record<string, unknown>): Record<string, unknown> {
  if (row.payload && typeof row.payload === "object" && !Array.isArray(row.payload)) {
    return row.payload as Record<string, unknown>;
  }
  const {
    id: _id,
    namespace: _ns,
    createdAt: _ca,
    updatedAt: _ua,
    finishedAt: _fa,
    state: _st,
    result: _res,
    resultReason: _rr,
    resultMessage: _rm,
    nodeName: _nn,
    status: _stat,
    durationMs: _dur,
    $: _dollar,
    [DOLLAR_REWRITE_IDENTIFIER]: _dollarRewrite,
    ...rest
  } = row;
  void _id;
  void _ns;
  void _ca;
  void _ua;
  void _fa;
  void _st;
  void _res;
  void _rr;
  void _rm;
  void _nn;
  void _stat;
  void _dur;
  void _dollar;
  void _dollarRewrite;
  return rest;
}

/**
 * Merge a Start trigger's template defaults with row-derived payload fields.
 * `payloadTemplates` maps dot-paths to literal strings or `{{ cel }}` templates.
 */
export function buildRowPayloadFromTemplates(
  payloadTemplates: Record<string, string> | undefined,
  row: Record<string, unknown>,
): Record<string, unknown> {
  if (!payloadTemplates) return {};
  const env = buildEnv();
  const stringify = (value: unknown) => (value == null ? "" : String(value));
  const out: Record<string, unknown> = {};
  for (const [path, templateRaw] of Object.entries(payloadTemplates)) {
    const value = templateRaw.includes("{{")
      ? evalTemplate(compileTemplate(templateRaw), row, env, stringify)
      : templateRaw;
    setNestedString(out, path, value);
  }
  return out;
}

/**
 * Build the hook parameters for a row action. Always deep-merges the
 * row-derived `payload` map into the base parameters so authors can wire
 * per-row values into the trigger using `{{ row_field }}` templates,
 * including for the default `run` hook. The merged shape stays flat at
 * the top level (`{ template, ...rowPayload }`) so the backend can resolve
 * `{{ parameters.<dot.path> }}` placeholders declared in the template
 * configuration via `InvokeNodeTriggerHook`'s expression resolver.
 *
 * When `payloadTemplates` is omitted or empty, falls back to the row's
 * selected payload so that row actions trigger with the clicked row's data.
 */
export function mergeTriggerParameters(
  node: ComponentsNode | undefined,
  hookName: string,
  templateName: string | undefined,
  row: Record<string, unknown>,
  payloadTemplates?: Record<string, string>,
): Record<string, unknown> {
  const base = buildConsoleTriggerParameters(node, hookName, templateName);
  const rowPayload =
    payloadTemplates && Object.keys(payloadTemplates).length > 0
      ? buildRowPayloadFromTemplates(payloadTemplates, row)
      : resolveDefaultRowPayload(row);
  return deepMergeObjects(base, rowPayload);
}
