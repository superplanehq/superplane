import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey, useFactoryApps, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryAppConfigurePath } from "../lib/factoryPagePaths";

export function useAutomationsPageModel() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { appId: routeAppId } = useParams<{ appId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: apps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const canCreateApp = canAct("canvases", "create");

  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createCanvas = useCreateCanvas(organizationId);

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
      navigate(factoryAppConfigurePath(organizationId, factoryKey, canvasId, { from: "automations" }));
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create automation"));
      throw error;
    }
  };

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
    legacyLineId,
    createCanvas,
    handleCreateAutomation,
    showLegacyRedirect,
  };
}
