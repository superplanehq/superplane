import { usePermissions } from "@/contexts/usePermissions";
import { useCanvas } from "@/hooks/useCanvasData";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import type { FactoryConfigureActions } from "@/pages/app";
import { useCallback, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { resolveFactoryAppBackNav } from "../lib/factoryAppNav";
import {
  isFactoryAppConfigureMode,
  resolveFactoryAppCanvasSubtitle,
  resolveFactoryLineName,
} from "../lib/factoryAppCanvasCopy";
import { shouldRedirectFactoryAppCanvas } from "../lib/factoryAppCanvasRedirect";
import { resolveWorkOrderByNumber } from "../lib/workOrderNumberResolution";
import { useFactoryAppConfigureTitle } from "./useFactoryAppConfigureTitle";

function readFactoryAppOrderRef(searchParams: URLSearchParams): string | null {
  return searchParams.get("orderNumber") ?? searchParams.get("orderId");
}

function resolveCanRenameAutomation(
  permissionsLoading: boolean,
  canAct: (resource: string, action: string) => boolean,
) {
  return permissionsLoading || canAct("canvases", "update");
}

export function useFactoryAppCanvasPageModel() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { appId = "" } = useParams<{ appId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const configureActionsRef = useRef<FactoryConfigureActions | null>(null);
  const [configureBusy, setConfigureBusy] = useState(false);

  const {
    data: canvas,
    isLoading: canvasLoading,
    error: canvasError,
  } = useCanvas(organizationId, appId, {
    enabled: Boolean(appId),
  });
  const canRename = resolveCanRenameAutomation(permissionsLoading, canAct);
  const from = searchParams.get("from");
  const lineId = searchParams.get("lineId");
  const orderNumber = readFactoryAppOrderRef(searchParams);
  const isConfigure = isFactoryAppConfigureMode(searchParams);
  const lineName = useMemo(() => resolveFactoryLineName(factory?.lines, lineId), [factory?.lines, lineId]);

  // Reads the already-fetched work orders list (same query the sidebar's
  // "recent orders" section uses) rather than fetching this one order by id,
  // since all we need here is its title for the back-link label.
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const order = useMemo(
    () => resolveWorkOrderByNumber(workOrders, orderNumber ?? undefined, false).order,
    [workOrders, orderNumber],
  );

  const back = useMemo(
    () =>
      resolveFactoryAppBackNav(organizationId, factoryKey, {
        from,
        appId,
        appName: canvas?.metadata?.name,
        lineId,
        orderNumber,
        lineName,
        orderTitle: order?.title,
      }),
    [appId, canvas?.metadata?.name, factoryKey, from, lineId, lineName, order?.title, organizationId, orderNumber],
  );

  const navigateDone = useCallback(() => {
    navigate(back.href);
  }, [back.href, navigate]);

  const {
    title,
    configureBusy: titleConfigureBusy,
    handleDraftTitleChange,
    handleConfigureSave,
    handleConfigureDiscard,
    clearDraftTitle,
  } = useFactoryAppConfigureTitle({
    organizationId,
    factoryId,
    appId,
    isConfigure,
    canRename,
    savedName: canvas?.metadata?.name,
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

  const belongsToFactory = canvas?.metadata?.factoryId === factoryId;
  const shouldRedirect = shouldRedirectFactoryAppCanvas({
    appId,
    canvasLoading,
    canvasError,
    belongsToFactory,
    hasCanvas: Boolean(canvas),
  });

  const subtitle = resolveFactoryAppCanvasSubtitle({
    isConfigure,
    description: canvas?.metadata?.description?.trim(),
    factoryName: factory?.name,
  });

  return {
    organizationId,
    factoryId,
    factoryKey,
    appId,
    canvas,
    canvasLoading,
    isConfigure,
    configureBusy: titleConfigureBusy,
    configureActionsRef,
    back,
    title,
    subtitle,
    shouldRedirect,
    canRename,
    handleDraftTitleChange,
    handleConfigureSave,
    handleConfigureDiscard,
    handleConfigureDone,
    handleConfigureBusyChange,
  };
}
