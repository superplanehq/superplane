import { AgentResourcesEditor } from "./AgentResourcesEditor";

export function PlanningReviewResourcesCard({
  organizationId,
  factoryId,
  factoryKey,
  disabledIds,
  disabledTools,
  enabledTools,
  onDisabledIdsChange,
  onDisabledToolsChange,
  onEnabledToolsChange,
}: {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  disabledIds: string[];
  disabledTools?: Record<string, string[]>;
  enabledTools?: Record<string, string[]>;
  onDisabledIdsChange: (ids: string[]) => void;
  onDisabledToolsChange?: (tools: Record<string, string[]>) => void;
  onEnabledToolsChange?: (tools: Record<string, string[]>) => void;
}) {
  return (
    <AgentResourcesEditor
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      disabledIds={disabledIds}
      disabledTools={disabledTools ?? {}}
      enabledTools={enabledTools ?? {}}
      onDisabledIdsChange={onDisabledIdsChange}
      onDisabledToolsChange={onDisabledToolsChange ?? (() => undefined)}
      onEnabledToolsChange={onEnabledToolsChange ?? (() => undefined)}
    />
  );
}
