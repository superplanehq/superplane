import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";
import type { StartTemplate } from "@/pages/app/mappers/start/templatePayload";

/** A Start trigger template as exposed by `node.configuration.templates`. */
export interface ConsoleTriggerTemplate {
  name: string;
}

/**
 * Read the (named) trigger templates declared on a node's configuration.
 * Returns an empty array when the node is undefined, has no templates, or
 * exposes templates without names. Shared by the dashboard form editor and
 * `buildConsoleTriggerParameters` so both agree on which template is the
 * default and what default payload it carries.
 */
export function getTriggerTemplates(node: ComponentsNode | undefined): ConsoleTriggerTemplate[] {
  if (!node) return [];
  const config = node.configuration as { templates?: StartTemplate[] } | undefined;
  const templates = config?.templates;
  if (!templates || templates.length === 0) return [];
  const out: ConsoleTriggerTemplate[] = [];
  for (const tpl of templates) {
    if (!tpl?.name) continue;
    out.push({ name: tpl.name });
  }
  return out;
}

/**
 * Resolve the full `StartTemplate` (including optional `parameters` /
 * `payload`) declared on a node's configuration. Used by the console Run
 * dialog to render the parameter form and preview the payload it will
 * submit. When `templateName` is provided, the matching template is
 * returned; otherwise we fall back to the first template — matching the
 * default rule used by {@link buildConsoleTriggerParameters}. Returns
 * `undefined` when the node is undefined or has no templates.
 */
export function resolveStartTemplate(
  node: ComponentsNode | undefined,
  templateName?: string,
): StartTemplate | undefined {
  if (!node) return undefined;
  const config = node.configuration as { templates?: StartTemplate[] } | undefined;
  const templates = config?.templates;
  if (!templates || templates.length === 0) return undefined;
  if (templateName) {
    const match = templates.find((tpl) => tpl?.name === templateName);
    if (match) return match;
  }
  return templates.find((tpl) => Boolean(tpl?.name));
}

/**
 * True when the resolved Start template declares at least one input
 * parameter. Widget run controls use this to decide whether a Run click must
 * open the confirm dialog (to collect field values) or can fire the trigger
 * directly.
 */
export function triggerHasParameters(node: ComponentsNode | undefined, templateName?: string): boolean {
  const template = resolveStartTemplate(node, templateName);
  return (template?.parameters?.length ?? 0) > 0;
}

/**
 * Derive the `parameters` body the gRPC `InvokeNodeTriggerHook` endpoint
 * expects when the dashboard fires a quick Run on a referenced node.
 *
 * Trigger hooks are typed per-component. For the standard Start trigger
 * the `run` hook requires `{ template }` — passing an empty object causes
 * the backend validator to reject with `field 'template' is required`.
 * To keep the dashboard's Run button frictionless, we pick the first
 * template defined on the node by default.
 *
 * When the referenced node does not expose templates, we fall back to an
 * empty parameters object so the API can surface its own validation error
 * rather than us guessing the shape. We intentionally key off the presence
 * of `configuration.templates` instead of only `component === "start"` so
 * dashboard references still work if the API omits or renames the component
 * field while preserving the same Start-trigger configuration shape.
 */
export function buildConsoleTriggerParameters(
  node: ComponentsNode | undefined,
  hookName: string,
  templateName?: string,
): Record<string, unknown> {
  if (!node || hookName !== "run") return {};
  const templates = getTriggerTemplates(node);
  if (templates.length === 0) return {};
  const template = (templateName ? templates.find((t) => t.name === templateName) : undefined) ?? templates[0];
  return { template: template.name };
}
