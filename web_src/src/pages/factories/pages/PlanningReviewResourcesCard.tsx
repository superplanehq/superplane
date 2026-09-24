import { AgentResourcesEditor } from "./AgentResourcesEditor";

export function PlanningReviewResourcesCard({
  organizationId,
  factoryId,
  factoryKey,
  disabledIds,
  disabledTools,
  onDisabledIdsChange,
  onDisabledToolsChange,
}: {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  disabledIds: string[];
  disabledTools?: Record<string, string[]>;
  onDisabledIdsChange: (ids: string[]) => void;
  onDisabledToolsChange?: (tools: Record<string, string[]>) => void;
}) {
  return (
    <AgentResourcesEditor
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      disabledIds={disabledIds}
      disabledTools={disabledTools ?? {}}
      onDisabledIdsChange={onDisabledIdsChange}
      onDisabledToolsChange={onDisabledToolsChange ?? (() => undefined)}
    />
  );
}
