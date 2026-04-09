import { useMemo } from "react";
import { useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { useTriggers, useWidgets } from "@/hooks/useCanvasData";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { Registry } from "@/lib/index/registry";

export function useRegistry(organizationId: string) {
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId);
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId);
  const { data: widgets = [], isLoading: widgetsLoading } = useWidgets();
  const { data: availableIntegrations = [], isLoading: integrationsLoading } = useAvailableIntegrations();

  const registry = useMemo(
    () =>
      new Registry({
        triggers,
        blueprints,
        components,
        widgets,
        availableIntegrations,
      }),
    [triggers, blueprints, components, widgets, availableIntegrations],
  );

  const loading = triggersLoading || blueprintsLoading || componentsLoading || widgetsLoading || integrationsLoading;
  return {
    registry,
    loading,
  };
}
