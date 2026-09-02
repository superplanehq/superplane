import type { CanvasesCanvas } from "@/api-client";
import { DEFAULT_FACTORY_ID, listFactoryDefinitions, type FactoryDefinition } from "@/pages/home/factories";

/**
 * Legacy installations do not have explicit template metadata. Match their
 * stable entrypoint node ID so the reset action remains available.
 */
export function resolveFactoryAppTemplate(canvas: CanvasesCanvas | null | undefined): FactoryDefinition | null {
  const nodeIds = new Set((canvas?.spec?.nodes ?? []).map((node) => node.id).filter((id): id is string => Boolean(id)));
  if (nodeIds.size === 0) return null;

  return (
    listFactoryDefinitions().find(
      (definition) => definition.id !== DEFAULT_FACTORY_ID && nodeIds.has(definition.run.nodeId),
    ) ?? null
  );
}

/**
 * Controls whether the reset action is visible. The backend remains
 * authoritative and materializes the defaults. Stable template metadata is
 * used for newly generated apps; node IDs keep existing installations working.
 */
export function hasFactoryAppDefaults(canvas: CanvasesCanvas | null | undefined): boolean {
  const nodes = canvas?.spec?.nodes ?? [];
  if (resolveFactoryAppTemplate(canvas)) return true;
  if (nodes.some((node) => node.component === "onWorkOrder")) return true;

  if (
    nodes.some((node) => {
      const template = node.metadata?.factoryTemplate;
      return Boolean(template && typeof template === "object");
    })
  ) {
    return true;
  }

  const nodeIds = new Set(nodes.map((node) => node.id));
  return (
    (nodeIds.has("trigger") && nodeIds.has("create-work-order")) ||
    (nodeIds.has("on-issue-labeled") && nodeIds.has("create-work-order"))
  );
}
