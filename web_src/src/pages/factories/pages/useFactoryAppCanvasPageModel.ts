import { useCanvas } from "@/hooks/useCanvasData";
import { useWorkOrder } from "@/hooks/useFactoryData";
import type { FactoryConfigureActions } from "@/pages/app";
import { useCallback, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { resolveFactoryAppBackNav } from "../lib/factoryAppNav";
import {
  isFactoryAppConfigureMode,
  resolveFactoryAppCanvasSubtitle,
  resolveFactoryAppCanvasTitle,
  resolveFactoryLineName,
} from "../lib/factoryAppCanvasCopy";
import { shouldRedirectFactoryAppCanvas } from "../lib/factoryAppCanvasRedirect";

export function useFactoryAppCanvasPageModel() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { appId = "" } = useParams<{ appId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const configureActionsRef = useRef<FactoryConfigureActions | null>(null);
  const [configureBusy, setConfigureBusy] = useState(false);

  const {
    data: canvas,
    isLoading: canvasLoading,
    error: canvasError,
  } = useCanvas(organizationId, appId, {
    enabled: Boolean(appId),
  });

  const from = searchParams.get("from");
  const lineId = searchParams.get("lineId");
  const orderId = searchParams.get("orderId");
  const isConfigure = isFactoryAppConfigureMode(searchParams);
  const lineName = useMemo(() => resolveFactoryLineName(factory?.lines, lineId), [factory?.lines, lineId]);

  const { data: order } = useWorkOrder(organizationId, factoryId, orderId ?? "");

  const back = useMemo(
    () =>
      resolveFactoryAppBackNav(organizationId, factoryId, {
        from,
        appId,
        appName: canvas?.metadata?.name,
        lineId,
        orderId,
        lineName,
        orderTitle: order?.title,
      }),
    [appId, canvas?.metadata?.name, factoryId, from, lineId, lineName, order?.title, organizationId, orderId],
  );

  const handleConfigureDone = useCallback(() => {
    navigate(back.href);
  }, [back.href, navigate]);

  const handleConfigureBusyChange = useCallback((busy: boolean) => {
    setConfigureBusy(busy);
  }, []);

  const canvasFactoryId = canvas?.metadata?.factoryId;
  const belongsToFactory = canvasFactoryId === factoryId;
  const shouldRedirect = shouldRedirectFactoryAppCanvas({
    appId,
    canvasLoading,
    canvasError,
    belongsToFactory,
    hasCanvas: Boolean(canvas),
  });

  return {
    organizationId,
    factoryId,
    appId,
    canvas,
    canvasLoading,
    isConfigure,
    configureBusy,
    configureActionsRef,
    back,
    title: resolveFactoryAppCanvasTitle(canvas?.metadata?.name),
    subtitle: resolveFactoryAppCanvasSubtitle({
      isConfigure,
      description: canvas?.metadata?.description?.trim(),
      factoryName: factory?.name,
    }),
    shouldRedirect,
    handleConfigureDone,
    handleConfigureBusyChange,
  };
}
