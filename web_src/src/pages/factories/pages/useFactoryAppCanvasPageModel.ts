import { usePermissions } from "@/contexts/usePermissions";
import type { FactoryConfigureActions } from "@/pages/app";
import { useCallback, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useFactoryAppCanvasEditActions } from "./useFactoryAppCanvasEditActions";
import { useFactoryAppCanvasRoute } from "./useFactoryAppCanvasRoute";
import { useFactoryAppConfigureTitle } from "./useFactoryAppConfigureTitle";

function resolveCanRenameAutomation(
  permissionsLoading: boolean,
  canAct: (resource: string, action: string) => boolean,
) {
  return !permissionsLoading && canAct("canvases", "update");
}

export function useFactoryAppCanvasPageModel() {
  const route = useFactoryAppCanvasRoute();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const configureActionsRef = useRef<FactoryConfigureActions | null>(null);
  const [configureBusy, setConfigureBusy] = useState(false);
  const canRename = resolveCanRenameAutomation(permissionsLoading, canAct);
  const navigateDone = useCallback(() => {
    navigate(route.back.href);
  }, [navigate, route.back.href]);
  const {
    title,
    configureBusy: titleConfigureBusy,
    handleDraftTitleChange,
    handleConfigureSave,
    handleConfigureDiscard,
    clearDraftTitle,
  } = useFactoryAppConfigureTitle({
    organizationId: route.organizationId,
    factoryId: route.factoryId,
    appId: route.appId,
    isConfigure: route.isConfigure,
    canRename,
    savedName: route.canvas?.metadata?.name,
    configureBusy,
    configureActionsRef,
    onDone: navigateDone,
  });
  const handleConfigureDone = useCallback(() => {
    clearDraftTitle();
    navigateDone();
  }, [clearDraftTitle, navigateDone]);
  const handleConfigureBusyChange = useCallback((busy: boolean) => {
    setConfigureBusy(busy);
  }, []);
  const editActions = useFactoryAppCanvasEditActions({
    organizationId: route.organizationId,
    factoryKey: route.factoryKey,
    appId: route.appId,
    from: route.from,
    lineId: route.lineId,
    orderNumber: route.orderNumber,
    setSearchParams: route.setSearchParams,
    navigate,
  });

  return {
    organizationId: route.organizationId,
    factoryId: route.factoryId,
    factoryKey: route.factoryKey,
    appId: route.appId,
    canvas: route.canvas,
    canvasLoading: route.canvasLoading,
    isConfigure: route.isConfigure,
    configureBusy: titleConfigureBusy,
    configureActionsRef,
    back: route.back,
    title,
    subtitle: route.subtitle,
    shouldRedirect: route.shouldRedirect,
    canRename,
    handleDraftTitleChange,
    handleConfigureSave,
    handleConfigureDiscard,
    handleConfigureDone,
    handleConfigureBusyChange,
    runId: route.runId,
    lineId: route.lineId,
    agentPromptOpen: route.agentPromptOpen,
    yamlViewOpen: route.yamlViewOpen,
    ...editActions,
  };
}
