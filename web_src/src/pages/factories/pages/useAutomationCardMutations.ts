import type { CanvasesCanvasSummary, FactoryApp } from "@/api-client";
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

type PendingDuplicateCanvas = {
  canvasId: string;
  name: string;
};

function collectExistingCanvasNames(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
  factoryId: string,
  sessionNames: Iterable<string>,
): string[] {
  const names = new Set<string>([...sessionNames]);
  const canvases = queryClient.getQueryData<CanvasesCanvasSummary[]>(canvasKeys.list(organizationId));
  for (const canvas of canvases ?? []) {
    if (canvas.name?.trim()) {
      names.add(canvas.name.trim());
    }
  }
  const apps = queryClient.getQueryData<FactoryApp[]>(factoryAppsKey(organizationId, factoryId));
  for (const app of apps ?? []) {
    if (app.name?.trim()) {
      names.add(app.name.trim());
    }
  }
  return [...names];
}

async function runDuplicateAutomation(args: {
  app: FactoryApp;
  factoryId: string;
  factoryKey: string;
  organizationId: string;
  createCanvas: ReturnType<typeof useCreateCanvas>["mutateAsync"];
  queryClient: ReturnType<typeof useQueryClient>;
  pendingDuplicateCanvases: Map<string, PendingDuplicateCanvas>;
  sessionDuplicateNames: Set<string>;
  invalidateFactoryApps: () => void;
  navigate: ReturnType<typeof useNavigate>;
}) {
  const { app } = args;
  if (!app.id) {
    return;
  }
  const pending = args.pendingDuplicateCanvases.get(app.id);
  const canvasId = await duplicateAutomationCanvas({
    factoryId: args.factoryId,
    app,
    createCanvas: args.createCanvas,
    existingCanvasNames: collectExistingCanvasNames(
      args.queryClient,
      args.organizationId,
      args.factoryId,
      args.sessionDuplicateNames,
    ),
    pendingCanvasId: pending?.canvasId,
    pendingCanvasName: pending?.name,
    onCanvasCreated: (createdCanvasId, createdName) => {
      args.pendingDuplicateCanvases.set(app.id!, {
        canvasId: createdCanvasId,
        name: createdName,
      });
      args.sessionDuplicateNames.add(createdName);
    },
  });
  args.pendingDuplicateCanvases.delete(app.id);
  args.invalidateFactoryApps();
  args.queryClient.removeQueries({ queryKey: canvasKeys.detail(args.organizationId, canvasId) });
  showSuccessToast("Automation duplicated.");
  args.navigate(factoryAppConfigurePath(args.organizationId, args.factoryKey, canvasId, { from: "automations" }));
}

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
  const pendingDuplicateCanvases = useRef(new Map<string, PendingDuplicateCanvas>());
  const sessionDuplicateNames = useRef(new Set<string>());

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
        await runDuplicateAutomation({
          app,
          factoryId,
          factoryKey,
          organizationId,
          createCanvas: createCanvas.mutateAsync,
          queryClient,
          pendingDuplicateCanvases: pendingDuplicateCanvases.current,
          sessionDuplicateNames: sessionDuplicateNames.current,
          invalidateFactoryApps,
          navigate,
        });
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
