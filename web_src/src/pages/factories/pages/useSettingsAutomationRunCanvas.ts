import type { ActionsAction, CanvasesCanvasRun, TriggersTrigger } from "@/api-client";
import { useCanvas, useDescribeRun, useEventExecutions, useTriggers } from "@/hooks/useCanvasData";
import { useComponents } from "@/hooks/useComponentData";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useMe } from "@/hooks/useMe";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";

import { useRunCanvasData, useRunCanvasPresentation } from "@/pages/app/useRunCanvasData";
import { useRunParticipantFitRequest } from "@/pages/app/useRunParticipantFitRequest";
import { useSelectedRunCanvas } from "@/pages/app/useSelectedRunCanvas";
import { isValidRunId } from "@/pages/app/workflowPageHelpers";

import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface SettingsAutomationRunCanvasInput {
  organizationId?: string;
  canvasId?: string;
  selectedRunId: string | null;
  selectedRunFromList: CanvasesCanvasRun | null;
  liveGraph: IntakeAutomationGraph;
}

export interface SettingsAutomationRunCanvas {
  graph: IntakeAutomationGraph;
  isRunInspectionMode: boolean;
  runCanvasLoading: boolean;
  selectedRun: CanvasesCanvasRun | null;
  runParticipantNodeIds?: string[];
  fitAllRequest: number | null;
  fitAllFocusNodeIds?: string[];
}

/** Prepares the settings automation canvas for the selected ListRuns row. */
export function useSettingsAutomationRunCanvas({
  organizationId,
  canvasId,
  selectedRunId,
  selectedRunFromList,
  liveGraph,
}: SettingsAutomationRunCanvasInput): SettingsAutomationRunCanvas {
  const isRunInspectionMode = Boolean(selectedRunId);
  const queryClient = useQueryClient();
  const { data: me } = useMe();
  const catalog = useSettingsAutomationCatalog(organizationId);
  const selected = useSettingsAutomationSelectedRun({
    organizationId,
    canvasId,
    selectedRunId,
    selectedRunFromList,
    isRunInspectionMode,
  });

  const runCanvasData = useRunCanvasData({
    isRunInspectionMode,
    selectedRun: selected.run,
    selectedRunCanvas: selected.canvas,
    canvasLoading: selected.canvasLoading,
    triggersLoading: catalog.isLoading,
    componentsLoading: catalog.isLoading,
    isSelectedRunVersionLoading: selected.isVersionLoading,
    allTriggers: catalog.triggers,
    allComponents: catalog.components,
    canvasId,
    queryClient,
    me,
    visibleNodeExecutionsMap: {},
    selectedRunFullExecutions: selected.fullExecutions,
  });

  const presentation = useRunCanvasPresentation({
    isRunInspectionMode,
    selectedRun: selected.run,
    runCanvasData,
    liveNodes: liveGraph.nodes,
    liveEdges: liveGraph.edges,
    isSelectedRunLoading: selected.isLoading,
    isSelectedRunVersionLoading: selected.isVersionLoading,
    isSelectedRunExecutionsLoading: selected.isExecutionsLoading,
  });

  const runParticipantFit = useRunParticipantFitRequest({
    isRunInspectionMode,
    selectedRunId,
    runCanvasLoading: presentation.runCanvasLoading,
    runCanvasData,
  });
  useFitSelectedSettingsRun(
    selectedRunId,
    runParticipantFit.requestParticipantFit,
    runParticipantFit.clearParticipantFit,
  );

  return {
    graph: {
      ...liveGraph,
      nodes: presentation.nodes,
      edges: presentation.edges,
    },
    isRunInspectionMode,
    runCanvasLoading: presentation.runCanvasLoading,
    selectedRun: selected.run,
    runParticipantNodeIds: runParticipantFit.participantNodeIds,
    fitAllRequest: runParticipantFit.fitRequest,
    fitAllFocusNodeIds: runParticipantFit.participantNodeIds,
  };
}

function useFitSelectedSettingsRun(
  selectedRunId: string | null,
  requestParticipantFit: (runId: string) => void,
  clearParticipantFit: () => void,
) {
  useEffect(() => {
    if (!selectedRunId) {
      clearParticipantFit();
      return;
    }
    requestParticipantFit(selectedRunId);
  }, [clearParticipantFit, requestParticipantFit, selectedRunId]);
}

function useSettingsAutomationSelectedRun({
  organizationId,
  canvasId,
  selectedRunId,
  selectedRunFromList,
  isRunInspectionMode,
}: {
  organizationId?: string;
  canvasId?: string;
  selectedRunId: string | null;
  selectedRunFromList: CanvasesCanvasRun | null;
  isRunInspectionMode: boolean;
}) {
  const canvasQuery = useCanvas(organizationId ?? "", canvasId ?? "", {
    enabled: Boolean(organizationId && canvasId),
  });
  const liveCanvas = canvasQuery.data;
  const describeEnabled = canDescribeSelectedRun(isRunInspectionMode, selectedRunId);
  const describedRunQuery = useDescribeRun(canvasId ?? "", selectedRunId, describeEnabled);
  const selectedRun = resolveSelectedRun(selectedRunId, selectedRunFromList, describedRunQuery.data?.run);
  const selectedRunExecutionsQuery = useEventExecutions(canvasId ?? "", selectedRun?.rootEvent?.id ?? null);
  const { selectedRunCanvas, isSelectedRunVersionLoading } = useSelectedRunCanvas({
    organizationId: organizationId ?? "",
    canvasId: canvasId ?? "",
    selectedRun,
    isRunInspectionMode,
    liveCanvasVersionId: liveCanvas?.metadata?.liveVersionId,
    canvas: liveCanvas,
    liveCanvas,
  });

  return {
    run: selectedRun,
    canvas: selectedRunCanvas,
    canvasLoading: isCanvasQueryPending(canvasQuery.isPending, liveCanvas),
    isLoading: isDescribeRunPending(describeEnabled, selectedRun, describedRunQuery.isLoading),
    isVersionLoading: isSelectedRunVersionLoading,
    isExecutionsLoading: selectedRunExecutionsQuery.isLoading,
    fullExecutions: selectedRunExecutionsQuery.data?.executions,
  };
}

function isCanvasQueryPending(isPending: boolean, canvas: unknown): boolean {
  return Boolean(isPending && !canvas);
}

function canDescribeSelectedRun(isRunInspectionMode: boolean, selectedRunId: string | null): boolean {
  return Boolean(isRunInspectionMode && selectedRunId && isValidRunId(selectedRunId));
}

function isDescribeRunPending(
  describeEnabled: boolean,
  selectedRun: CanvasesCanvasRun | null,
  isLoading: boolean,
): boolean {
  return Boolean(describeEnabled && !selectedRun && isLoading);
}

function resolveSelectedRun(
  selectedRunId: string | null,
  selectedRunFromList: CanvasesCanvasRun | null,
  describedRun: CanvasesCanvasRun | undefined,
): CanvasesCanvasRun | null {
  if (!selectedRunId) {
    return null;
  }
  if (selectedRunFromList?.id === selectedRunId) {
    return selectedRunFromList;
  }
  return describedRun ?? selectedRunFromList;
}

function useSettingsAutomationCatalog(organizationId: string | undefined) {
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId ?? "");
  const { data: integrations = [], isLoading: integrationsLoading } = useAvailableIntegrations();

  return useMemo(() => {
    const allTriggers: TriggersTrigger[] = [...triggers];
    const allComponents: ActionsAction[] = [...components];
    for (const integration of integrations) {
      if (!integration.capabilities) {
        continue;
      }
      allTriggers.push(...triggersFromCapabilities(integration.capabilities));
      allComponents.push(...actionsFromCapabilities(integration.capabilities));
    }
    return {
      triggers: allTriggers,
      components: allComponents,
      isLoading: triggersLoading || componentsLoading || integrationsLoading,
    };
  }, [components, integrations, integrationsLoading, componentsLoading, triggers, triggersLoading]);
}
