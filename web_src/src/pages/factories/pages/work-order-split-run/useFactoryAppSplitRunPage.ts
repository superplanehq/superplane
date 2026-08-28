import { useFactoryPullRequests, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useFactoryBacklogAnalysis } from "@/hooks/useBacklogAnalysisRuns";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { useCallback, useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { resolveFactoryAppCanvasSubtitle, resolveFactoryLineName } from "../../lib/factoryAppCanvasCopy";
import { resolveFactoryAppBackNav } from "../../lib/factoryAppNav";
import { factoryAppConfigurePath, parseFactoryAppNavFrom } from "../../lib/factoryPagePaths";
import { useWorkOrderPRFeedbackLog } from "../useWorkOrderPRFeedbackRunHref";
import { attachArtifactsToStream } from "./attachStreamArtifacts";
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
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";

function useSplitRunPageSelection(
  organizationId: string,
  factoryId: string,
  lines: Array<{ id?: string; name?: string }> | undefined,
) {
  const [searchParams] = useSearchParams();
  const query = readSplitRunQuery(searchParams);
  const lineName = useMemo(() => resolveFactoryLineName(lines, query.lineId), [lines, query.lineId]);
  const { data: workOrders = [], isLoading } = useFactoryWorkOrders(organizationId, factoryId);
  const order = useMemo(
    () => resolveSplitRunOrder(workOrders, query.orderNumber, query.runId, isLoading),
    [isLoading, query.orderNumber, query.runId, workOrders],
  );
  return { isLoading, lineName, order, query };
}

function useSplitRunWorkOrderExtras(
  organizationId: string,
  factoryId: string,
  order: ReturnType<typeof useSplitRunPageSelection>["order"],
) {
  const orderId = order?.id ?? "";
  const { data: orderChecks = [] } = useWorkOrderChecks(organizationId, factoryId, orderId);
  const { data: pullRequests = [] } = useFactoryPullRequests(
    organizationId,
    factoryId,
    orderId ? { workOrderIds: [orderId] } : undefined,
  );
  const { data: handlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const prFeedbackRuns = useWorkOrderPRFeedbackLog(order ? pullRequests : [], handlers);
  const { runsByWorkOrder } = useFactoryBacklogAnalysis(organizationId, factoryId);
  const analysisRuns = orderId ? (runsByWorkOrder.get(orderId) ?? []) : [];
  return { orderChecks, prFeedbackRuns, analysisRuns };
}

export function useFactoryAppSplitRunPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { appId = "" } = useParams<{ appId: string }>();
  const [nodeId, setNodeId] = useState<string | null>(null);
  const split = useSplitRunPanePercent();
  const { isLoading, lineName, order, query } = useSplitRunPageSelection(organizationId, factoryId, factory?.lines);
  const { orderChecks, prFeedbackRuns, analysisRuns } = useSplitRunWorkOrderExtras(organizationId, factoryId, order);
  const fixture = useMemo(
    () => fixtureForSplitRunPage(order, orderChecks, query.lineId, prFeedbackRuns, analysisRuns),
    [order, orderChecks, prFeedbackRuns, analysisRuns, query.lineId],
  );
  const canvasKey = query.canvasKey ?? canvasKeyForAutomation({ id: appId });
  const phase = useMemo(
    () => splitRunPhaseOnRoute(phaseForSplitRunCanvas(fixture, canvasKey, query.runId), appId),
    [appId, canvasKey, fixture, query.runId],
  );
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, order?.id);
  const visual = useMemo(() => resolveSplitRunVisual(phase, live, { demoArtifacts: false }), [live, phase]);
  const stream = useMemo(() => attachArtifactsToStream(visual.stream, artifactIndex), [artifactIndex, visual.stream]);
  const back = useMemo(
    () =>
      resolveFactoryAppBackNav(organizationId, factoryKey, {
        from: query.from,
        appId,
        appName: visual.canvas.title,
        lineId: query.lineId,
        orderNumber: query.orderNumber,
        lineName,
        orderTitle: order?.title,
      }),
    [
      appId,
      factoryKey,
      lineName,
      order?.title,
      organizationId,
      query.from,
      query.lineId,
      query.orderNumber,
      visual.canvas.title,
    ],
  );
  const configureNav = useMemo(
    () => ({
      from: parseFactoryAppNavFrom(query.from),
      lineId: query.lineId ?? undefined,
      runId: query.runId ?? undefined,
      orderNumber: query.orderNumber ?? undefined,
    }),
    [query.from, query.lineId, query.orderNumber, query.runId],
  );
  const editHref = factoryAppConfigurePath(organizationId, factoryKey, appId, configureNav);
  const nodeEditHref = useCallback(
    (nodeId: string) => factoryAppConfigurePath(organizationId, factoryKey, appId, { ...configureNav, nodeId }),
    [appId, configureNav, factoryKey, organizationId],
  );

  usePageTitle([splitRunPageTitle(!order, isLoading, visual.canvas.title), factory?.name ?? "Workspace"]);

  return {
    back,
    canvas: visual.canvas,
    editHref,
    fixture,
    isLoading,
    liveError: live.isError,
    nodeId,
    nodeEditHref,
    organizationId,
    phase,
    setNodeId,
    split,
    stream,
    subtitle: resolveFactoryAppCanvasSubtitle({ factoryName: factory?.name }),
  };
}
