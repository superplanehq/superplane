import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas, useDeleteCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey, useFactoryApps, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";
import type { FactoryApp } from "@/api-client";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { automationsPath, factoryAppConfigurePath } from "../lib/factoryPagePaths";
import { duplicateAutomationName, type AutomationCardActions } from "./automationCardActions";

export function useAutomationsPageModel() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { appId: routeAppId } = useParams<{ appId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: apps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const canCreateApp = canAct("canvases", "create");
  const canUpdateApp = canAct("canvases", "update");
  const canDeleteApp = canAct("canvases", "delete");

  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createCanvas = useCreateCanvas(organizationId);
  const deleteCanvas = useDeleteCanvas(organizationId);

  const selectedApp = useMemo(() => {
    if (!routeAppId) {
      return null;
    }
    return apps.find((app) => app.id === routeAppId) ?? null;
  }, [apps, routeAppId]);

  const legacyLineId = useMemo(() => {
    if (!routeAppId || !factory?.lines) {
      return undefined;
    }
    return factory.lines.find((line) => line.id === routeAppId)?.id;
  }, [factory?.lines, routeAppId]);

  const handleCreateAutomation = async (input: { name: string; description: string }) => {
    try {
      const result = await createCanvas.mutateAsync({
        name: input.name,
        description: input.description,
        factoryId,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      setCreateOpen(false);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      if (!canvasId) {
        return;
      }
      showSuccessToast("Automation created.");
      navigate(factoryAppConfigurePath(organizationId, factoryId, canvasId, { from: "automations" }));
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create automation"));
      throw error;
    }
  };

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
        if (selectedApp?.id === app.id) {
          navigate(automationsPath(organizationId, factoryId));
        }
      } catch (error) {
        showErrorToast("Failed to delete automation");
        throw error;
      }
    },
    [deleteCanvas, factoryId, invalidateFactoryApps, navigate, organizationId, selectedApp?.id],
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

  const selectedAppActions = selectedApp ? actionsForApp(selectedApp) : null;

  const showLegacyRedirect = Boolean(routeAppId && !appsLoading && !selectedApp);

  return {
    organizationId,
    factoryId,
    factory,
    apps,
    appsLoading,
    workOrders,
    canCreateApp,
    permissionsLoading,
    createOpen,
    setCreateOpen,
    selectedApp,
    selectedAppActions,
    actionsForApp,
    legacyLineId,
    createCanvas,
    handleCreateAutomation,
    showLegacyRedirect,
  };
}
