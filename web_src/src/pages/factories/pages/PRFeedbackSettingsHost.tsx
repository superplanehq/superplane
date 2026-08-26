import type { FactoriesFactoryPrFeedbackHandlerRun, FactoriesWorkOrder } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  useCreateFactoryPRFeedbackHandler,
  useDeleteFactoryPRFeedbackHandler,
  useFactoryPRFeedbackHandlerRuns,
  useFactoryPRFeedbackHandlers,
  useUpdateFactoryPRFeedbackHandler,
} from "@/hooks/useFactoryPRFeedbackData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";

import { factoryAppConfigurePath, factoryAppRunPath, workOrderDetailPath } from "../lib/factoryPagePaths";
import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  prFeedbackDraftFromHandler,
  type PRFeedbackDraftSettings,
  type PRFeedbackSettingsTab,
} from "./prFeedbackSettingsModel";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

interface PRFeedbackSettingsHostProps {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  workOrders?: FactoriesWorkOrder[];
  canUpdate: boolean;
  initialTab?: PRFeedbackSettingsTab;
  onClose: () => void;
}

export function PRFeedbackSettingsHost({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  workOrders = [],
  canUpdate,
  initialTab = "general",
  onClose,
}: PRFeedbackSettingsHostProps) {
  const handlersQuery = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const createHandler = useCreateFactoryPRFeedbackHandler(organizationId, factoryId);
  const handler = handlersQuery.data?.[0];

  if (handlersQuery.isPending) {
    return (
      <PopupShell testId="pr-feedback-settings" canvas fixed onDismiss={onClose}>
        <PopupHeader title="PR feedback" onClose={onClose} />
        <p className="workspace-body-text px-6 py-6 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.loading}</p>
      </PopupShell>
    );
  }

  if (handlersQuery.isError) {
    return (
      <PopupShell testId="pr-feedback-settings" canvas fixed onDismiss={onClose}>
        <PopupHeader title="PR feedback" onClose={onClose} />
        <div className="flex items-center gap-3 px-6 py-6">
          <p className="workspace-body-text text-destructive">{PR_FEEDBACK_SETTINGS_COPY.loadError}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void handlersQuery.refetch()}>
            {PR_FEEDBACK_SETTINGS_COPY.retry}
          </Button>
        </div>
      </PopupShell>
    );
  }

  if (!handler?.id) {
    return (
      <PopupShell testId="pr-feedback-settings" canvas fixed onDismiss={onClose}>
        <PopupHeader title="PR feedback" onClose={onClose} />
        <div className="flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6">
          <p className="workspace-body-text text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.emptyBody}</p>
          {canUpdate ? (
            <Button
              type="button"
              disabled={createHandler.isPending}
              onClick={() => {
                createHandler.mutateAsync({}).catch((error) => {
                  showErrorToast(getApiErrorMessage(error, PR_FEEDBACK_SETTINGS_COPY.createError));
                });
              }}
              data-testid="pr-feedback-create"
            >
              {createHandler.isPending ? PR_FEEDBACK_SETTINGS_COPY.creating : PR_FEEDBACK_SETTINGS_COPY.create}
            </Button>
          ) : null}
        </div>
      </PopupShell>
    );
  }

  return (
    <PRFeedbackSettingsLoaded
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      lineId={lineId}
      workOrders={workOrders}
      canUpdate={canUpdate}
      initialTab={initialTab}
      handlerId={handler.id}
      canvasId={handler.canvasId}
      settings={prFeedbackDraftFromHandler(handler)}
      healthy={Boolean(handler.healthy)}
      onClose={onClose}
    />
  );
}

function PRFeedbackSettingsLoaded({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  workOrders,
  canUpdate,
  initialTab,
  handlerId,
  canvasId,
  settings,
  healthy,
  onClose,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  workOrders: FactoriesWorkOrder[];
  canUpdate: boolean;
  initialTab?: PRFeedbackSettingsTab;
  handlerId: string;
  canvasId?: string;
  settings: PRFeedbackDraftSettings;
  healthy: boolean;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const [saveError, setSaveError] = useState<string | undefined>();
  const automation = useIntakeAutomationCanvas(organizationId, canvasId);
  const runsQuery = useFactoryPRFeedbackHandlerRuns(organizationId, factoryId, handlerId);
  const updateHandler = useUpdateFactoryPRFeedbackHandler(organizationId, factoryId);
  const deleteHandler = useDeleteFactoryPRFeedbackHandler(organizationId, factoryId);
  const workOrderHrefFor = useMemo(
    () => workOrderHrefLookup(organizationId, factoryKey, workOrders),
    [factoryKey, organizationId, workOrders],
  );
  const editAutomationHref = canvasId
    ? factoryAppConfigurePath(organizationId, factoryKey, canvasId, { from: "lines", lineId })
    : undefined;

  function openRun(run: FactoriesFactoryPrFeedbackHandlerRun) {
    if (!canvasId || !run.id) {
      return;
    }
    navigate(factoryAppRunPath(organizationId, factoryKey, canvasId, run.id, { from: "lines", lineId }));
  }

  return (
    <PRFeedbackSettingsPopup
      settings={settings}
      healthy={healthy}
      automationGraph={automation.graph}
      automationLoading={automation.isLoading}
      automationError={automation.isError}
      onRetryAutomation={() => void automation.refetch()}
      runs={runsQuery.data ?? []}
      runsLoading={runsQuery.isPending}
      runsError={runsQuery.isError}
      onRetryRuns={() => void runsQuery.refetch()}
      savePending={updateHandler.isPending}
      deletePending={deleteHandler.isPending}
      saveError={saveError}
      onSave={async (next) => {
        setSaveError(undefined);
        try {
          await updateHandler.mutateAsync({
            handlerId,
            name: next.name,
            settings: {
              repository: next.repository,
              mention: next.mention,
              ignoreBots: next.ignoreBots,
            },
          });
        } catch (error) {
          const message = getApiErrorMessage(error, PR_FEEDBACK_SETTINGS_COPY.saveError);
          setSaveError(message);
          throw error;
        }
      }}
      onDelete={
        canUpdate
          ? async () => {
              try {
                await deleteHandler.mutateAsync(handlerId);
                onClose();
              } catch (error) {
                showErrorToast(getApiErrorMessage(error, PR_FEEDBACK_SETTINGS_COPY.saveError));
              }
            }
          : undefined
      }
      onOpenRun={openRun}
      workOrderHrefFor={workOrderHrefFor}
      editAutomationHref={editAutomationHref}
      onClose={onClose}
      initialTab={initialTab}
    />
  );
}

function workOrderHrefLookup(
  organizationId: string,
  factoryKey: string,
  workOrders: FactoriesWorkOrder[],
): (workOrderId: string) => string | undefined {
  const numberById = new Map(
    workOrders.flatMap((order) => (order.id && order.number != null ? [[order.id, order.number]] : [])),
  );
  return (workOrderId: string) => {
    const number = numberById.get(workOrderId);
    return number == null ? undefined : workOrderDetailPath(organizationId, factoryKey, number);
  };
}
