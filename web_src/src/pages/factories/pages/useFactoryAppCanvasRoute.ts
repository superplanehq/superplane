import { useCanvas } from "@/hooks/useCanvasData";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useMemo } from "react";
import { useParams, useSearchParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  isFactoryAppAgentPromptOpen,
  isFactoryAppConfigureMode,
  isFactoryAppYamlViewOpen,
  resolveFactoryAppCanvasSubtitle,
  resolveFactoryLineName,
} from "../lib/factoryAppCanvasCopy";
import { shouldRedirectFactoryAppCanvas } from "../lib/factoryAppCanvasRedirect";
import { resolveFactoryAppBackNav } from "../lib/factoryAppNav";
import { resolveWorkOrderByNumber } from "../lib/workOrderNumberResolution";

function readFactoryAppOrderRef(searchParams: URLSearchParams): string | null {
  return searchParams.get("orderNumber") ?? searchParams.get("orderId");
}

export function useFactoryAppCanvasRoute() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { appId = "" } = useParams<{ appId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const {
    data: canvas,
    isLoading: canvasLoading,
    error: canvasError,
  } = useCanvas(organizationId, appId, {
    enabled: Boolean(appId),
  });
  const from = searchParams.get("from");
  const lineId = searchParams.get("lineId");
  const runId = searchParams.get("run");
  const orderNumber = readFactoryAppOrderRef(searchParams);
  const isConfigure = isFactoryAppConfigureMode(searchParams);
  const agentPromptOpen = isFactoryAppAgentPromptOpen(searchParams);
  const yamlViewOpen = isFactoryAppYamlViewOpen(searchParams);
  const lineName = useMemo(() => resolveFactoryLineName(factory?.lines, lineId), [factory?.lines, lineId]);
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
  const shouldRedirect = shouldRedirectFactoryAppCanvas({
    appId,
    canvasLoading,
    canvasError,
    belongsToFactory: canvas?.metadata?.factoryId === factoryId,
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
    from,
    lineId,
    runId,
    orderNumber,
    isConfigure,
    agentPromptOpen,
    yamlViewOpen,
    back,
    shouldRedirect,
    subtitle,
    setSearchParams,
  };
}
