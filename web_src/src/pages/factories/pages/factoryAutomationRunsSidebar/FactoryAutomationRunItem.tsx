import type { CanvasesCanvasRun, FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { Link } from "@/components/Link/link";
import { formatTimeAgo } from "@/lib/date";
import { isNormalClick } from "@/lib/linkHelpers";
import { cn } from "@/lib/utils";
import { analyzedWorkOrderId } from "../../lib/backlogAnalysis";
import {
  findWorkOrderForAutomationRun,
  listFactoryAutomationRuns,
  type FactoryAutomationRunCard,
} from "../../lib/factoryAutomationStatus";
import { buildWorkOrderListEntry } from "../../lib/workOrderListModel";
import { isActiveCanvasRun } from "../../lib/workOrderPullRequest";
import { WorkOrderCard, type WorkOrderCardContext } from "../../workOrders/WorkOrderCard";
import { StatusTick } from "../automationsPageParts";
import { factoryAutomationRunCardSurfaceClass } from "./factoryAutomationRunCardSurface";
import { factoryAutomationRunTitle } from "./factoryAutomationRunSearch";

export function FactoryAutomationRunItem({
  run,
  workOrders,
  factory,
  workOrderCardContext,
  runHref,
  isSelected,
  onSelect,
}: {
  run: CanvasesCanvasRun;
  workOrders: FactoriesWorkOrder[];
  factory: FactoriesFactory | null | undefined;
  workOrderCardContext: WorkOrderCardContext | null;
  runHref: string;
  isSelected: boolean;
  onSelect: (runId: string) => void;
}) {
  const runId = run.id ?? "";
  const order = findWorkOrderForAutomationRun(workOrders, runId, run);
  const selectRun = () => {
    if (runId) {
      onSelect(runId);
    }
  };

  if (order && workOrderCardContext) {
    const entry = buildWorkOrderListEntry(order, factory);
    const isAnalyzing = analyzedWorkOrderId(run) === order.id && isActiveCanvasRun(run);
    return (
      <div data-testid="factory-automation-runs-row" className="px-2 py-1.5">
        <WorkOrderCard
          {...workOrderCardContext}
          entry={entry}
          isAnalyzing={isAnalyzing}
          selected={isSelected}
          className={factoryAutomationRunCardSurfaceClass(isSelected)}
          onOpen={selectRun}
        />
      </div>
    );
  }

  return <FactoryAutomationRunFallback run={run} runHref={runHref} isSelected={isSelected} onSelect={selectRun} />;
}

function FactoryAutomationRunFallback({
  run,
  runHref,
  isSelected,
  onSelect,
}: {
  run: CanvasesCanvasRun;
  runHref: string;
  isSelected: boolean;
  onSelect: () => void;
}) {
  const cards = listFactoryAutomationRuns([run]);
  const card = cards[0];
  const title = card?.title ?? factoryAutomationRunTitle(run);

  return (
    <div data-testid="factory-automation-runs-row" className="px-2 py-1.5">
      <Link
        href={runHref}
        data-testid={run.id ? `factory-automation-run-${run.id}` : undefined}
        data-selected={isSelected || undefined}
        className={cn(
          "block w-full rounded-md border border-border bg-card px-3 py-2.5 text-left transition-colors",
          factoryAutomationRunCardSurfaceClass(isSelected),
        )}
        aria-label={title}
        onClick={(event) => {
          if (isNormalClick(event)) {
            event.preventDefault();
            onSelect();
          }
        }}
      >
        <div className="truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</div>
        {card ? <FactoryAutomationRunFallbackMeta card={card} /> : null}
      </Link>
    </div>
  );
}

function FactoryAutomationRunFallbackMeta({ card }: { card: FactoryAutomationRunCard }) {
  const timestamp = card.updatedAt ?? card.finishedAt ?? card.createdAt;
  const timeLabel =
    card.tick === "queued" && !timestamp ? "next" : timestamp ? formatTimeAgo(new Date(timestamp), false) : null;

  return (
    <div className="mt-1.5 flex items-center gap-1.5 text-[12px] text-muted-foreground">
      <StatusTick tick={card.tick} size="sm" />
      <span>
        {card.label}
        {timeLabel ? ` · ${timeLabel}` : ""}
      </span>
    </div>
  );
}
