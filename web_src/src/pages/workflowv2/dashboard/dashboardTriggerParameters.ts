import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client/types.gen";

/**
 * Derive the `parameters` body the gRPC `InvokeNodeTriggerHook` endpoint
 * expects when the dashboard fires a quick Run on a referenced node.
 *
 * Trigger hooks are typed per-component. For the standard Start trigger
 * the `run` hook requires `{ template, payload }` — passing an empty
 * object causes the backend validator to reject with `field 'template'
 * is required`. To keep the dashboard's Run button frictionless, we
 * mirror the canvas view: pick the first template defined on the node
 * and seed `payload` with that template's default payload. Authors who
 * need to invoke a different template or override the payload can run
 * the trigger from the canvas card instead, which opens the full
 * payload editor.
 *
 * When the referenced node does not expose templates, we fall back to an
 * empty parameters object so the API can surface its own validation error
 * rather than us guessing the shape. We intentionally key off the presence
 * of `configuration.templates` instead of only `component === "start"` so
 * dashboard references still work if the API omits or renames the component
 * field while preserving the same Start-trigger configuration shape.
 */
export function buildDashboardTriggerParameters(
  node: ComponentsNode | undefined,
  hookName: string,
  templateName?: string,
): Record<string, unknown> {
  if (!node || hookName !== "run") return {};
  const config = node.configuration as { templates?: Array<{ name?: string; payload?: unknown }> } | undefined;
  const templates = config?.templates;
  if (!templates || templates.length === 0) return {};
  const template = (templateName ? templates.find((t) => t.name === templateName) : undefined) ?? templates[0];
  if (!template?.name) return {};
  const payload =
    template.payload && typeof template.payload === "object" && !Array.isArray(template.payload)
      ? (template.payload as Record<string, unknown>)
      : {};
  return { template: template.name, payload };
}
