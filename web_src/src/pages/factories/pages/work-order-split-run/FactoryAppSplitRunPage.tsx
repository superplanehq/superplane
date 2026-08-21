import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { cn } from "@/lib/utils";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useCallback, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { resolveFactoryAppCanvasSubtitle, resolveFactoryLineName } from "../../lib/factoryAppCanvasCopy";
import { resolveFactoryAppBackNav } from "../../lib/factoryAppNav";
import { factoryAppConfigurePath, parseFactoryAppNavFrom } from "../../lib/factoryPagePaths";
import { findWorkOrderByRunId, resolveWorkOrderByNumber } from "../../lib/workOrderNumberResolution";
import { FactoryAppCanvasHeader } from "../FactoryAppCanvasHeader";
import { CompactLineCanvas } from "./CompactLineCanvas";
import { PhaseLogCard } from "./PhaseLogCard";
import {
  canvasKeyForPhase,
  parseSplitRunCanvasKey,
  richStreamForCanvas,
  splitRunCanvasForPhase,
} from "./splitRunCanvases";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { useSplitRunPanePercent } from "./useSplitRunPanePercent";

/**
 * Copy of the factory Automation Run page. The body is a resizable split:
 * the selected canvas log on the left, that canvas on the right.
 */
export function FactoryAppSplitRunPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { appId = "" } = useParams<{ appId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [nodeId, setNodeId] = useState<string | null>(null);
  const split = useSplitRunPanePercent();
  const from = searchParams.get("from");
  const lineId = searchParams.get("lineId");
  const runId = searchParams.get("run");
  const orderNumber = searchParams.get("orderNumber") ?? searchParams.get("orderId");
  const canvasKey = parseSplitRunCanvasKey(searchParams.get("canvas")) ?? "implementation";
  const lineName = useMemo(() => resolveFactoryLineName(factory?.lines, lineId), [factory?.lines, lineId]);
  const { data: workOrders = [], isLoading } = useFactoryWorkOrders(organizationId, factoryId);
  const order = useMemo(() => {
    const byNumber = resolveWorkOrderByNumber(workOrders, orderNumber ?? undefined, isLoading).order;
    return byNumber ?? findWorkOrderByRunId(workOrders, runId) ?? null;
  }, [isLoading, orderNumber, runId, workOrders]);
  const { data: orderChecks = [] } = useWorkOrderChecks(organizationId, factoryId, order?.id ?? "");
  const fixture = useMemo(
    () => (order ? splitRunFixtureForWorkOrder(order, { checks: orderChecks }) : SPLIT_RUN_RUNNING),
    [order, orderChecks],
  );
  const phase = useMemo(() => {
    return (
      fixture.phases.find((entry) => canvasKeyForPhase(entry) === canvasKey) ??
      fixture.phases[0] ??
      SPLIT_RUN_RUNNING.phases.find((entry) => entry.id === "implement")!
    );
  }, [canvasKey, fixture]);
  const canvas = useMemo(() => splitRunCanvasForPhase(phase), [phase]);
  const stream = useMemo(() => richStreamForCanvas(canvas), [canvas]);
  const back = useMemo(
    () =>
      resolveFactoryAppBackNav(organizationId, factoryKey, {
        from,
        appId,
        appName: canvas.title,
        lineId,
        orderNumber,
        lineName,
        orderTitle: order?.title,
      }),
    [appId, canvas.title, factoryKey, from, lineId, lineName, order?.title, organizationId, orderNumber],
  );
  const subtitle = resolveFactoryAppCanvasSubtitle({ factoryName: factory?.name });
  const editHref = factoryAppConfigurePath(organizationId, factoryKey, appId, {
    from: parseFactoryAppNavFrom(from),
    lineId: lineId ?? undefined,
    runId: runId ?? undefined,
    orderNumber: orderNumber ?? undefined,
  });
  const handleOpenVisualEditor = useCallback(() => {
    navigate(editHref);
  }, [editHref, navigate]);

  usePageTitle([canvas.title, factory?.name ?? "Workspace"]);

  return (
    <div className="absolute inset-0 flex flex-col bg-background" data-testid="factory-app-split-run-page">
      <FactoryAppCanvasHeader
        backHref={back.href}
        backLabel={back.label}
        title={canvas.title}
        subtitle={subtitle}
        isConfigure={false}
        configureBusy={false}
        onDiscard={() => undefined}
        onSave={() => undefined}
        onOpenVisualEditor={handleOpenVisualEditor}
      />
      <div ref={split.containerRef} className="flex min-h-0 flex-1 overflow-hidden">
        <aside
          className="flex min-h-0 min-w-[12rem] flex-col border-r border-border bg-muted/25"
          style={{ width: `${split.percent}%` }}
          aria-label="Log"
        >
          <ol className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
            <li>
              <PhaseLogCard
                phase={phase}
                expanded
                collapsible={false}
                stream={stream}
                selectedNodeId={nodeId}
                onSelectNode={setNodeId}
              />
            </li>
          </ol>
        </aside>
        <div
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize log and canvas"
          data-testid="split-run-resize-handle"
          onPointerDown={split.startResize}
          className="group relative z-10 w-2 shrink-0 cursor-col-resize bg-transparent"
        >
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-border",
              split.isResizing && "bg-border",
            )}
          />
        </div>
        <section className="flex min-h-0 min-w-[12rem] flex-1 flex-col" aria-label="Run">
          <CompactLineCanvas canvas={canvas} selectedId={nodeId} onSelect={setNodeId} showHeader={false} />
        </section>
      </div>
    </div>
  );
}
