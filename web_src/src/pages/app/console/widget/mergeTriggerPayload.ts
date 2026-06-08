import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

import { buildConsoleTriggerParameters } from "../consoleTriggerParameters";
import { buildEnv, compileTemplate, evalTemplate } from "./celExpr";
import { deepMergeObjects, setNestedString } from "./nestedPayload";

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
 */
export function mergeTriggerParameters(
  node: ComponentsNode | undefined,
  hookName: string,
  templateName: string | undefined,
  row: Record<string, unknown>,
  payloadTemplates?: Record<string, string>,
): Record<string, unknown> {
  const base = buildConsoleTriggerParameters(node, hookName, templateName);
  const rowPayload = buildRowPayloadFromTemplates(payloadTemplates, row);
  return deepMergeObjects(base, rowPayload);
}
