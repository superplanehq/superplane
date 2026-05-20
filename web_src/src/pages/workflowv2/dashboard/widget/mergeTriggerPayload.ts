import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

import { buildDashboardTriggerParameters } from "../dashboardTriggerParameters";
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

export function mergeTriggerParameters(
  node: ComponentsNode | undefined,
  hookName: string,
  templateName: string | undefined,
  row: Record<string, unknown>,
  payloadTemplates?: Record<string, string>,
): Record<string, unknown> {
  const base = buildDashboardTriggerParameters(node, hookName, templateName);
  const rowPayload = buildRowPayloadFromTemplates(payloadTemplates, row);
  if (hookName !== "run") {
    return deepMergeObjects(base, rowPayload);
  }
  const basePayload =
    base.payload && typeof base.payload === "object" && !Array.isArray(base.payload)
      ? (base.payload as Record<string, unknown>)
      : {};
  return {
    ...base,
    payload: deepMergeObjects(basePayload, rowPayload),
  };
}
