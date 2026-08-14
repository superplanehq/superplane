import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryApps, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryAppConfigurePath } from "../lib/factoryPagePaths";
import { useAutomationCardMutations } from "./useAutomationCardMutations";

export function useAutomationsPageModel() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { appId: routeAppId } = useParams<{ appId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: apps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const canCreateApp = canAct("canvases", "create");
  const canUpdateApp = canAct("canvases", "update");
  const canDeleteApp = canAct("canvases", "delete");

  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();

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

  const { createCanvas, invalidateFactoryApps, actionsForApp } = useAutomationCardMutations({
    organizationId,
    factoryId,
    factoryKey,
    canCreateApp,
    canUpdateApp,
    canDeleteApp,
    selectedAppId: selectedApp?.id,
  });

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
      invalidateFactoryApps();
      if (!canvasId) {
        return;
      }
      showSuccessToast("Automation created.");
      navigate(factoryAppConfigurePath(organizationId, factoryKey, canvasId, { from: "automations" }));
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create automation"));
      throw error;
    }
  };

  const selectedAppActions = selectedApp ? actionsForApp(selectedApp) : null;
  const showLegacyRedirect = Boolean(routeAppId && !appsLoading && !selectedApp);

  return {
    organizationId,
    factoryId,
    factoryKey,
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
