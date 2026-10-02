import type { FactoriesWorkOrder, FactoriesWorkOrderCheck } from "@/api-client";
import {
  useFactoryAutomations,
  useFactoryWorkOrders,
  useWorkOrder,
  useWorkOrderArtifacts,
} from "@/hooks/useFactoryData";
import { useFactoryBacklogAnalysis } from "@/hooks/useBacklogAnalysisRuns";
import { useFactoryMergeConfidenceRuns } from "@/hooks/useMergeConfidenceRuns";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useCallback, useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { resolveFactoryAppCanvasSubtitle, resolveFactoryLineName } from "../../lib/factoryAppCanvasCopy";
import { resolveFactoryAppBackNav } from "../../lib/factoryAppNav";
import { factoryAppConfigurePath, parseFactoryAppNavFrom } from "../../lib/factoryPagePaths";
import { firstWorkOrderPullRequests } from "../../lib/workOrderPullRequest";
import { useWorkOrderPRFeedbackLog } from "../useWorkOrderPRFeedbackRunHref";
import { attachArtifactsToStream } from "./attachStreamArtifacts";
import { canvasKeyForAutomation } from "./splitRunCanvases";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
import { columnAppsFromFactoryApps } from "./splitRunMocks";
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
  liveOrder: FactoriesWorkOrder | undefined,
) {
  const orderId = order?.id ?? "";
  const orderChecks = firstWorkOrderChecks(liveOrder, order);
  const { data: artifacts = [] } = useWorkOrderArtifacts(organizationId, factoryId, orderId);
  const pullRequests = firstWorkOrderPullRequests(liveOrder, order);
  const { data: handlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const { data: factoryApps = [] } = useFactoryAutomations(organizationId, factoryId);
  const prFeedbackRuns = useWorkOrderPRFeedbackLog(order ? pullRequests : [], handlers);
  const { runsByWorkOrder, analyzingOrderIds } = useFactoryBacklogAnalysis(organizationId, factoryId);
  const analysisRuns = orderId ? (runsByWorkOrder.get(orderId) ?? []) : [];
  const mergeConfidence = useFactoryMergeConfidenceRuns(organizationId, factoryId, order ? pullRequests : []);
  // `analyzingOrderIds` also covers the optimistic window where a fresh draft
  // is known to be analyzing before its run appears in `analysisRuns`, so the
  // popup copy and actions match the board card.
  const isAnalyzing = Boolean(orderId && analyzingOrderIds.has(orderId));
  return {
    orderChecks,
    artifacts,
    prFeedbackRuns,
    analysisRuns,
    mergeConfidenceRuns: mergeConfidence.runs,
    mergeConfidenceCanvasIds: mergeConfidence.canvasIds,
    factoryApps,
    isAnalyzing,
  };
}

export function useFactoryAppSplitRunPage() {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const params = useParams<{ appId?: string; automationId?: string }>();
  const appId = params.automationId ?? params.appId ?? "";
  const [nodeId, setNodeId] = useState<string | null>(null);
  const split = useSplitRunPanePercent();
  const { isLoading, lineName, order, query } = useSplitRunPageSelection(organizationId, factoryId, factory?.lines);
  const liveWorkOrder = useWorkOrder(organizationId, factoryId, order?.id ?? "");
  const {
    orderChecks,
    artifacts,
    prFeedbackRuns,
    analysisRuns,
    mergeConfidenceRuns,
    mergeConfidenceCanvasIds,
    factoryApps,
    isAnalyzing,
  } = useSplitRunWorkOrderExtras(organizationId, factoryId, order, liveWorkOrder.data);
  const { resolveUser } = useOrgUserLookup(organizationId);
  const fixture = useMemo(
    () =>
      fixtureForSplitRunPage(order, orderChecks, query.lineId, {
        prFeedbackRuns,
        analysisRuns,
        mergeConfidenceRuns,
        columnApps: columnAppsFromFactoryApps(factoryApps),
        artifacts,
        isAnalyzing,
        resolveUser,
      }),
    [
      order,
      orderChecks,
      artifacts,
      factoryApps,
      prFeedbackRuns,
      analysisRuns,
      mergeConfidenceRuns,
      isAnalyzing,
      query.lineId,
      resolveUser,
    ],
  );
  const canvasKey = query.canvasKey ?? canvasKeyForAutomation({ id: appId });
  const phase = useMemo(
    () => splitRunPhaseOnRoute(phaseForSplitRunCanvas(fixture, canvasKey, query.runId), appId),
    [appId, canvasKey, fixture, query.runId],
  );
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, order?.id);
  const visual = useMemo(() => resolveSplitRunVisual(phase, live, { demoArtifacts: false }), [live, phase]);
  const stream = useMemo(
    () => attachArtifactsToStream(visual.stream, artifactIndex, phase.runId),
    [artifactIndex, phase.runId, visual.stream],
  );
  const back = useMemo(
    () =>
      resolveFactoryAppBackNav(organizationId, routeSegment, {
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
      routeSegment,
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
  const editHref = factoryAppConfigurePath(organizationId, routeSegment, appId, configureNav);
  const nodeEditHref = useCallback(
    (nodeId: string) => factoryAppConfigurePath(organizationId, routeSegment, appId, { ...configureNav, nodeId }),
    [appId, configureNav, organizationId, routeSegment],
  );

  usePageTitle([splitRunPageTitle(!order, isLoading, visual.canvas.title), factory?.name ?? "Workspace"]);

  return {
    back,
    canvas: visual.canvas,
    editHref,
    fixture,
    isLoading,
    liveError: live.isError,
    mergeConfidenceCanvasIds,
    nodeId,
    nodeEditHref,
    organizationId,
    phase,
    setNodeId,
    split,
    stream,
    streamLoading: live.isLoading,
    subtitle: resolveFactoryAppCanvasSubtitle({ factoryName: factory?.name }),
    files: liveWorkOrder.isSuccess ? liveWorkOrder.data?.files : undefined,
  };
}

function firstWorkOrderChecks(
  ...orders: Array<{ checks?: FactoriesWorkOrderCheck[] } | null | undefined>
): FactoriesWorkOrderCheck[] {
  for (const order of orders) {
    if (order?.checks) {
      return order.checks;
    }
  }
  return [];
}
