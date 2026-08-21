import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { resolveFactoryAppCanvasSubtitle, resolveFactoryLineName } from "../../lib/factoryAppCanvasCopy";
import { resolveFactoryAppBackNav } from "../../lib/factoryAppNav";
import { factoryAppConfigurePath, parseFactoryAppNavFrom } from "../../lib/factoryPagePaths";
import { canvasKeyForAutomation } from "./splitRunCanvases";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
import {
  fixtureForSplitRunPage,
  phaseForSplitRunCanvas,
  readSplitRunQuery,
  resolveSplitRunOrder,
  splitRunPageTitle,
  splitRunPhaseOnRoute,
} from "./splitRunPageModel";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import { useSplitRunPanePercent } from "./useSplitRunPanePercent";

export function useFactoryAppSplitRunPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { appId = "" } = useParams<{ appId: string }>();
  const [searchParams] = useSearchParams();
  const [nodeId, setNodeId] = useState<string | null>(null);
  const split = useSplitRunPanePercent();
  const query = readSplitRunQuery(searchParams);
  const lineName = useMemo(() => resolveFactoryLineName(factory?.lines, query.lineId), [factory?.lines, query.lineId]);
  const { data: workOrders = [], isLoading } = useFactoryWorkOrders(organizationId, factoryId);
  const order = useMemo(
    () => resolveSplitRunOrder(workOrders, query.orderNumber, query.runId, isLoading),
    [isLoading, query.orderNumber, query.runId, workOrders],
  );
  const { data: orderChecks = [] } = useWorkOrderChecks(organizationId, factoryId, order?.id ?? "");
  const fixture = useMemo(
    () => fixtureForSplitRunPage(order, orderChecks, query.lineId),
    [order, orderChecks, query.lineId],
  );
  const canvasKey = query.canvasKey ?? canvasKeyForAutomation({ id: appId });
  const phase = useMemo(
    () => splitRunPhaseOnRoute(phaseForSplitRunCanvas(fixture, canvasKey), appId),
    [appId, canvasKey, fixture],
  );
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const visual = useMemo(() => resolveSplitRunVisual(phase, live), [live, phase]);
  const canvas = visual.canvas;
  const stream = visual.stream;
  const back = useMemo(
    () =>
      resolveFactoryAppBackNav(organizationId, factoryKey, {
        from: query.from,
        appId,
        appName: canvas.title,
        lineId: query.lineId,
        orderNumber: query.orderNumber,
        lineName,
        orderTitle: order?.title,
      }),
    [
      appId,
      canvas.title,
      factoryKey,
      lineName,
      order?.title,
      organizationId,
      query.from,
      query.lineId,
      query.orderNumber,
    ],
  );
  const editHref = factoryAppConfigurePath(organizationId, factoryKey, appId, {
    from: parseFactoryAppNavFrom(query.from),
    lineId: query.lineId ?? undefined,
    runId: query.runId ?? undefined,
    orderNumber: query.orderNumber ?? undefined,
  });

  usePageTitle([splitRunPageTitle(!order, isLoading, canvas.title), factory?.name ?? "Workspace"]);

  return {
    back,
    canvas,
    editHref,
    fixture,
    isLoading,
    liveError: live.isError,
    nodeId,
    phase,
    setNodeId,
    split,
    stream,
    subtitle: resolveFactoryAppCanvasSubtitle({ factoryName: factory?.name }),
  };
}
