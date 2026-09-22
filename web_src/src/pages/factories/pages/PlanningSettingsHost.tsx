import { Button } from "@/components/ui/button";
import { useFactory, useFactoryAutomations, useUpdateFactory } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import { factoryAppConfigurePath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { findBacklogAutomationApp } from "../lib/linePhaseRuns";
import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PLANNING_SETTINGS_COPY,
  planningSettingsFromFactory,
  planningSettingsToApi,
  type PlanningSettingsTab,
} from "./planningSettingsModel";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";

const REFINE_TASK_NODE_ID = "refine-task";

interface PlanningSettingsHostProps {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  initialTab?: PlanningSettingsTab;
  onClose: () => void;
}

export function PlanningSettingsHost({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  initialTab = "general",
  onClose,
}: PlanningSettingsHostProps) {
  const factoryQuery = useFactory(organizationId, factoryId);
  const appsQuery = useFactoryAutomations(organizationId, factoryId);
  const canvasId = findBacklogAutomationApp(appsQuery.data ?? [])?.id;

  if (factoryQuery.isPending) {
    return (
      <PopupShell testId="planning-settings" canvas fixed onDismiss={onClose}>
        <PopupHeader title={PLANNING_SETTINGS_COPY.title} onClose={onClose} />
        <p className="workspace-body-text px-6 py-6 text-muted-foreground">{PLANNING_SETTINGS_COPY.loading}</p>
      </PopupShell>
    );
  }

  if (factoryQuery.isError || !factoryQuery.data) {
    return (
      <PopupShell testId="planning-settings" canvas fixed onDismiss={onClose}>
        <PopupHeader title={PLANNING_SETTINGS_COPY.title} onClose={onClose} />
        <div className="flex items-center gap-3 px-6 py-6">
          <p className="workspace-body-text text-destructive">{PLANNING_SETTINGS_COPY.loadError}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void factoryQuery.refetch()}>
            {PLANNING_SETTINGS_COPY.retry}
          </Button>
        </div>
      </PopupShell>
    );
  }

  return (
    <PlanningSettingsLoaded
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      lineId={lineId}
      canvasId={canvasId}
      settings={planningSettingsFromFactory(factoryQuery.data)}
      title={findBacklogAutomationApp(appsQuery.data ?? [])?.name?.trim() || PLANNING_SETTINGS_COPY.title}
      initialTab={initialTab}
      onClose={onClose}
    />
  );
}

function PlanningSettingsLoaded({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  canvasId,
  settings,
  title,
  initialTab,
  onClose,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  canvasId?: string;
  settings: ReturnType<typeof planningSettingsFromFactory>;
  title: string;
  initialTab?: PlanningSettingsTab;
  onClose: () => void;
}) {
  const [saveError, setSaveError] = useState<string | undefined>();
  const automation = useIntakeAutomationCanvas(organizationId, canvasId);
  const agent = useColumnCanvasAgentEditor(organizationId, canvasId, {
    preferredAgentNodeId: REFINE_TASK_NODE_ID,
  });
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const editAutomationHref = canvasId
    ? factoryAppConfigurePath(organizationId, factoryKey, canvasId, { from: "lines", lineId })
    : undefined;

  return (
    <PlanningSettingsPopup
      title={title}
      settings={settings}
      automationGraph={automation.graph}
      automationLoading={automation.isLoading}
      automationError={automation.isError}
      onRetryAutomation={() => void automation.refetch()}
      savePending={updateFactory.isPending}
      saveError={saveError}
      onSave={async (next) => {
        setSaveError(undefined);
        try {
          await updateFactory.mutateAsync({ planning: planningSettingsToApi(next) });
        } catch (error) {
          const message = getApiErrorMessage(error, PLANNING_SETTINGS_COPY.saveError);
          setSaveError(message);
          throw error;
        }
      }}
      editAutomationHref={editAutomationHref}
      canvasId={canvasId}
      runHrefFor={
        canvasId
          ? (runId) => factoryAppRunPath(organizationId, factoryKey, canvasId, runId, { from: "lines", lineId })
          : undefined
      }
      agent={
        agent.agentNode
          ? {
              draft: agent.draft ?? undefined,
              isLoading: agent.isLoading || !agent.draft,
              organizationId,
              onSave: agent.save,
              showVisualEvidenceSetting: false,
            }
          : undefined
      }
      onClose={onClose}
      initialTab={initialTab}
    />
  );
}
