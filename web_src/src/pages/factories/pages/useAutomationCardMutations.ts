import type { FactoryApp } from "@/api-client";
import { useCreateCanvas, useDeleteCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useNavigate } from "react-router";
import { automationsPath, factoryAppConfigurePath } from "../lib/factoryPagePaths";
import { duplicateAutomationName, type AutomationCardActions } from "./automationCardActions";

export function useAutomationCardMutations(args: {
  organizationId: string;
  factoryId: string;
  canCreateApp: boolean;
  canUpdateApp: boolean;
  canDeleteApp: boolean;
  selectedAppId: string | undefined;
}) {
  const { organizationId, factoryId, canCreateApp, canUpdateApp, canDeleteApp, selectedAppId } = args;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createCanvas = useCreateCanvas(organizationId);
  const deleteCanvas = useDeleteCanvas(organizationId);

  const invalidateFactoryApps = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
  }, [factoryId, organizationId, queryClient]);

  const handleEditAutomation = useCallback(
    (app: FactoryApp) => {
      if (!app.id) {
        return;
      }
      navigate(factoryAppConfigurePath(organizationId, factoryId, app.id, { from: "automations" }));
    },
    [factoryId, navigate, organizationId],
  );

  const handleDuplicateAutomation = useCallback(
    async (app: FactoryApp) => {
      try {
        const result = await createCanvas.mutateAsync({
          name: duplicateAutomationName(app.name),
          description: app.description ?? "",
          factoryId,
          method: "ui",
        });
        const canvasId = result?.data?.canvas?.metadata?.id;
        invalidateFactoryApps();
        if (!canvasId) {
          return;
        }
        showSuccessToast("Automation duplicated.");
        navigate(factoryAppConfigurePath(organizationId, factoryId, canvasId, { from: "automations" }));
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to duplicate automation"));
      }
    },
    [createCanvas, factoryId, invalidateFactoryApps, navigate, organizationId],
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
          navigate(automationsPath(organizationId, factoryId));
        }
      } catch (error) {
        showErrorToast("Failed to delete automation");
        throw error;
      }
    },
    [deleteCanvas, factoryId, invalidateFactoryApps, navigate, organizationId, selectedAppId],
  );

  const actionsForApp = useCallback(
    (app: FactoryApp): AutomationCardActions => ({
      onEdit: () => handleEditAutomation(app),
      onDuplicate: () => handleDuplicateAutomation(app),
      onDelete: () => handleDeleteAutomation(app),
      canEdit: canUpdateApp,
      canDuplicate: canCreateApp,
      canDelete: canDeleteApp,
      isDuplicating: createCanvas.isPending,
      isDeleting: deleteCanvas.isPending && deleteCanvas.variables === app.id,
    }),
    [
      canCreateApp,
      canDeleteApp,
      canUpdateApp,
      createCanvas.isPending,
      deleteCanvas.isPending,
      deleteCanvas.variables,
      handleDeleteAutomation,
      handleDuplicateAutomation,
      handleEditAutomation,
    ],
  );

  return {
    createCanvas,
    invalidateFactoryApps,
    actionsForApp,
  };
}
