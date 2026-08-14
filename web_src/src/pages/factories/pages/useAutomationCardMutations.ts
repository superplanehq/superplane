import type { FactoryApp } from "@/api-client";
import { canvasKeys, useCreateCanvas, useDeleteCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { automationsPath, factoryAppConfigurePath } from "../lib/factoryPagePaths";
import type { AutomationCardActions } from "./automationCardActions";
import { duplicateAutomationCanvas } from "./duplicateAutomationCanvas";

export function useAutomationCardMutations(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  canCreateApp: boolean;
  canUpdateApp: boolean;
  canDeleteApp: boolean;
  selectedAppId: string | undefined;
}) {
  const { organizationId, factoryId, factoryKey, canCreateApp, canUpdateApp, canDeleteApp, selectedAppId } = args;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createCanvas = useCreateCanvas(organizationId);
  const deleteCanvas = useDeleteCanvas(organizationId);
  const [isDuplicating, setIsDuplicating] = useState(false);
  // Reuse a canvas created on a failed stage/commit so retry does not spawn empties.
  const pendingDuplicateCanvasIds = useRef(new Map<string, string>());

  const invalidateFactoryApps = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
  }, [factoryId, organizationId, queryClient]);

  const handleEditAutomation = useCallback(
    (app: FactoryApp) => {
      if (!app.id) {
        return;
      }
      navigate(factoryAppConfigurePath(organizationId, factoryKey, app.id, { from: "automations" }));
    },
    [factoryKey, navigate, organizationId],
  );

  const handleDuplicateAutomation = useCallback(
    async (app: FactoryApp) => {
      if (isDuplicating || !app.id) {
        return;
      }
      setIsDuplicating(true);
      try {
        const canvasId = await duplicateAutomationCanvas({
          factoryId,
          app,
          createCanvas: createCanvas.mutateAsync,
          pendingCanvasId: pendingDuplicateCanvasIds.current.get(app.id),
          onCanvasCreated: (createdCanvasId) => {
            pendingDuplicateCanvasIds.current.set(app.id!, createdCanvasId);
          },
        });
        pendingDuplicateCanvasIds.current.delete(app.id);
        invalidateFactoryApps();
        queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
        showSuccessToast("Automation duplicated.");
        navigate(factoryAppConfigurePath(organizationId, factoryKey, canvasId, { from: "automations" }));
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to duplicate automation"));
      } finally {
        setIsDuplicating(false);
      }
    },
    [
      createCanvas.mutateAsync,
      factoryId,
      factoryKey,
      invalidateFactoryApps,
      isDuplicating,
      navigate,
      organizationId,
      queryClient,
    ],
  );

  const handleDeleteAutomation = useCallback(
    async (app: FactoryApp) => {
      if (!app.id) {
        return;
      }
      try {
        await deleteCanvas.mutateAsync(app.id);
        invalidateFactoryApps();
        showSuccessToast("Automation deleted.");
        if (selectedAppId === app.id) {
          navigate(automationsPath(organizationId, factoryKey));
        }
      } catch (error) {
        showErrorToast("Failed to delete automation");
        throw error;
      }
    },
    [deleteCanvas, factoryKey, invalidateFactoryApps, navigate, organizationId, selectedAppId],
  );

  const actionsForApp = useCallback(
    (app: FactoryApp): AutomationCardActions => ({
      onEdit: () => handleEditAutomation(app),
      onDuplicate: () => handleDuplicateAutomation(app),
      onDelete: () => handleDeleteAutomation(app),
      canEdit: canUpdateApp,
      canDuplicate: canCreateApp,
      canDelete: canDeleteApp,
      isDuplicating,
      isDeleting: deleteCanvas.isPending && deleteCanvas.variables === app.id,
    }),
    [
      canCreateApp,
      canDeleteApp,
      canUpdateApp,
      deleteCanvas.isPending,
      deleteCanvas.variables,
      handleDeleteAutomation,
      handleDuplicateAutomation,
      handleEditAutomation,
      isDuplicating,
    ],
  );

  return {
    createCanvas,
    invalidateFactoryApps,
    actionsForApp,
  };
}
