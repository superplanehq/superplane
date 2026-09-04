import { CheckCircle2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Sparkline } from "@/pages/app/console/widget/Sparkline";
import { EmptyState } from "@/ui/emptyState";

import {
  countPillClassName,
  factoryCardClassName,
  factoryPanelClassName,
  mutedTextClassName,
  sectionTitleClassName,
} from "./factoryStyles";
import type { FactorySummary, WorkOrder } from "./factoryTypes";
import { workOrderGroup } from "./factoryTypes";
import { WorkOrderListItem } from "./WorkOrderListItem";

interface FactoryOverviewTabProps {
  summary: FactorySummary;
  workOrders: WorkOrder[];
  onOpenWorkOrder: (workOrder: WorkOrder) => void;
  onSeeAllWorkOrders: () => void;
}

/**
 * PRD: "Overview is the default tab and puts Work Orders requiring attention
 * before summary metrics and other activity." That ordering is the whole point
 * of the tab, so blocked work is rendered first — above the metric row.
 */
export function FactoryOverviewTab({
  summary,
  workOrders,
  onOpenWorkOrder,
  onSeeAllWorkOrders,
}: FactoryOverviewTabProps) {
  const needsAttention = workOrders.filter((wo) => workOrderGroup(wo) === "needs-attention");
  const running = workOrders.filter((wo) => workOrderGroup(wo) === "running");

  return (
    <div className="flex flex-col gap-6">
      <section className={factoryPanelClassName}>
        <div className="mb-3 flex items-center gap-2">
          <h2 className={sectionTitleClassName}>Needs attention</h2>
          {needsAttention.length > 0 && <span className={countPillClassName}>{needsAttention.length}</span>}
        </div>
        {needsAttention.length === 0 ? (
          <EmptyState
            compact
            tone="neutral"
            icon={CheckCircle2}
            title="Nothing is waiting on you"
            description="Work Orders appear here when they need a decision, approval, or clarification."
          />
        ) : (
          <ul className="flex flex-col gap-2.5">
            {needsAttention.map((workOrder) => (
              <WorkOrderListItem key={workOrder.id} workOrder={workOrder} onOpen={onOpenWorkOrder} />
            ))}
          </ul>
        )}
      </section>

      <FactoryMetrics summary={summary} />

      <section className={factoryPanelClassName}>
        <div className="mb-3 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <h2 className={sectionTitleClassName}>Current activity</h2>
            {running.length > 0 && <span className={countPillClassName}>{running.length}</span>}
          </div>
          <button
            type="button"
            onClick={onSeeAllWorkOrders}
            className={cn("text-xs underline underline-offset-4", mutedTextClassName)}
          >
            See all Work Orders
          </button>
        </div>
        {running.length === 0 ? (
          <EmptyState
            compact
            title="No Work Orders running"
            description="Approved work is picked up by an Automation."
          />
        ) : (
          <ul className="flex flex-col gap-2.5">
            {running.map((workOrder) => (
              <WorkOrderListItem key={workOrder.id} workOrder={workOrder} onOpen={onOpenWorkOrder} />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

const usd = (value: number) => `$${value.toFixed(2)}`;

/** PRD requires throughput, success rate, active Work Orders and tracked cost. */
function FactoryMetrics({ summary }: { summary: FactorySummary }) {
  const cost = summary.trackedCost.tokensUsd + summary.trackedCost.computeUsd;

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <MetricTile
        label="Throughput"
        value={`${summary.throughput.value}`}
        caption={summary.throughput.unit}
        trend={summary.throughput.trend}
      />
      <MetricTile
        label="Success rate"
        value={`${Math.round(summary.successRate.value * 100)}%`}
        caption="explicit Work Order outcomes"
        trend={summary.successRate.trend}
      />
      <MetricTile label="Active Work Orders" value={`${summary.activeWorkOrders}`} caption="running or waiting" />
      <MetricTile
        label="Tracked cost"
        value={usd(cost)}
        caption={`${usd(summary.trackedCost.tokensUsd)} tokens · ${usd(summary.trackedCost.computeUsd)} compute`}
      />
    </div>
  );
}

function MetricTile({
  label,
  value,
  caption,
  trend,
}: {
  label: string;
  value: string;
  caption: string;
  trend?: number[];
}) {
  return (
    <div className={cn("px-4 py-3.5", factoryCardClassName)}>
      <p className={cn("text-xs font-medium", mutedTextClassName)}>{label}</p>
      <div className="mt-1.5 flex items-end justify-between gap-3">
        <div className="min-w-0">
          <p className="text-2xl font-semibold leading-none text-slate-900 dark:text-gray-100">{value}</p>
          {/* No truncation: the caption carries the throughput unit, which the
              PRD leaves open — losing it would make the number ambiguous. */}
          <p className={cn("mt-1.5 text-xs leading-snug", mutedTextClassName)}>{caption}</p>
        </div>
        {trend && trend.length > 0 && (
          <Sparkline values={trend} width={84} height={28} className="text-slate-300 dark:text-gray-600" />
        )}
      </div>
    </div>
  );
}
