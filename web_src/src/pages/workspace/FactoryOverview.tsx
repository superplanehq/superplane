import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { Activity, Bot, CheckCircle2, CircleX, Clock3, GitPullRequest, WalletCards } from "lucide-react";

import { FactoryAttentionSection } from "./FactoryWorkDashboard";
import type { FactoryAutomation, FactoryMetric, FactoryWorkItem } from "./types";

interface FactoryOverviewProps {
  metrics: FactoryMetric[];
  workItems: FactoryWorkItem[];
  automations: FactoryAutomation[];
  onOpenWorkItem: (workItem: FactoryWorkItem) => void;
}

const panelClassName =
  "overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900";

const metricIcons = {
  throughput: GitPullRequest,
  success: CheckCircle2,
  active: Activity,
  cost: WalletCards,
} as const;

export function FactoryOverview({ metrics, workItems, automations, onOpenWorkItem }: FactoryOverviewProps) {
  const attentionItems = workItems.filter((workItem) => workItem.status === "attention");
  const runningItems = workItems.filter((workItem) => workItem.status === "running");
  const doneItems = workItems.filter((workItem) => workItem.status === "done");
  const failedItems = workItems.filter((workItem) => workItem.status === "failed");
  const activeAutomations = automations.filter((automation) => automation.status === "active");

  return (
    <div className="space-y-4">
      <FactoryAttentionSection items={attentionItems} onOpenWorkItem={onOpenWorkItem} />
      <MetricsBand metrics={metrics} />
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1.45fr)_minmax(18rem,0.55fr)]">
        <OperationalSummary
          running={runningItems.length}
          done={doneItems.length}
          unsuccessful={failedItems.length}
          workItems={workItems}
          onOpenWorkItem={onOpenWorkItem}
        />
        <AutomationSummary active={activeAutomations.length} total={automations.length} automations={automations} />
      </div>
    </div>
  );
}

function MetricsBand({ metrics }: { metrics: FactoryMetric[] }) {
  return (
    <section
      aria-label="Factory metrics"
      className={cn(
        panelClassName,
        "grid grid-cols-2 divide-x divide-y divide-slate-200 md:grid-cols-4 md:divide-y-0 dark:divide-gray-700/70",
      )}
    >
      {metrics.map((metric) => {
        const Icon = metricIcons[metric.id];
        return (
          <div key={metric.id} className="min-w-0 px-4 py-4 sm:px-5">
            <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-gray-400">
              <Icon className="size-3.5" />
              <span className="truncate">{metric.label}</span>
            </div>
            <p className="mt-2 text-xl font-semibold text-slate-900 tabular-nums dark:text-gray-100">{metric.value}</p>
            <p
              className={cn(
                "mt-1 truncate text-xs",
                metric.tone === "positive"
                  ? "text-emerald-700 dark:text-emerald-400"
                  : "text-slate-500 dark:text-gray-400",
              )}
            >
              {metric.detail}
            </p>
          </div>
        );
      })}
    </section>
  );
}

function OperationalSummary({
  running,
  done,
  unsuccessful,
  workItems,
  onOpenWorkItem,
}: {
  running: number;
  done: number;
  unsuccessful: number;
  workItems: FactoryWorkItem[];
  onOpenWorkItem: (workItem: FactoryWorkItem) => void;
}) {
  const recentItems = workItems.filter((workItem) => workItem.status !== "attention").slice(0, 4);

  return (
    <section className={panelClassName}>
      <SectionHeader title="Work order activity" description="Current delivery state across this Factory." />
      <div className="grid grid-cols-3 divide-x divide-slate-200 border-b border-slate-200 dark:divide-gray-700/70 dark:border-gray-700/70">
        <StateCount icon={Activity} label="Running" value={running} tone="running" />
        <StateCount icon={CheckCircle2} label="Recently done" value={done} tone="done" />
        <StateCount icon={CircleX} label="Unsuccessful" value={unsuccessful} tone="failed" />
      </div>
      <div className="divide-y divide-slate-200 dark:divide-gray-700/70">
        {recentItems.map((workItem) => (
          <button
            key={workItem.id}
            type="button"
            onClick={() => onOpenWorkItem(workItem)}
            className="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-4 py-3 text-left transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400 sm:px-5 dark:hover:bg-gray-800/60"
          >
            <span className="min-w-0">
              <span className="flex items-center gap-2">
                <span className="text-[11px] font-medium text-slate-500 dark:text-gray-400">{workItem.id}</span>
                <span className="truncate text-xs font-medium text-slate-900 dark:text-gray-100">{workItem.title}</span>
              </span>
              <span className="mt-0.5 block truncate text-[11px] text-slate-500 dark:text-gray-400">
                {workItem.stage} · {workItem.detail}
              </span>
            </span>
            <span className="inline-flex shrink-0 items-center gap-1 text-[11px] text-slate-500 dark:text-gray-400">
              <Clock3 className="size-3" />
              {workItem.updatedAt}
            </span>
          </button>
        ))}
      </div>
    </section>
  );
}

function StateCount({
  icon: Icon,
  label,
  value,
  tone,
}: {
  icon: typeof Activity;
  label: string;
  value: number;
  tone: "running" | "done" | "failed";
}) {
  const iconClassName =
    tone === "running"
      ? "text-sky-600 dark:text-sky-400"
      : tone === "done"
        ? "text-emerald-600 dark:text-emerald-400"
        : "text-red-600 dark:text-red-400";

  return (
    <div className="min-w-0 px-4 py-3.5 sm:px-5">
      <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-gray-400">
        <Icon className={cn("size-3.5 shrink-0", iconClassName)} />
        <span className="truncate">{label}</span>
      </div>
      <p className="mt-1.5 text-lg font-semibold text-slate-900 tabular-nums dark:text-gray-100">{value}</p>
    </div>
  );
}

function AutomationSummary({
  active,
  total,
  automations,
}: {
  active: number;
  total: number;
  automations: FactoryAutomation[];
}) {
  return (
    <section className={panelClassName}>
      <SectionHeader
        title="Automations"
        description="Canvas-backed workflows."
        trailing={
          <Badge variant="outline" className="text-slate-600 dark:text-gray-300">
            {active}/{total} active
          </Badge>
        }
      />
      <div className="divide-y divide-slate-200 dark:divide-gray-700/70">
        {automations.slice(0, 4).map((automation) => (
          <div key={automation.id} className="flex items-start gap-3 px-4 py-3 sm:px-5">
            <span className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-300">
              <Bot className="size-3.5" />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex items-center justify-between gap-2">
                <p className="truncate text-xs font-medium text-slate-800 dark:text-gray-200">{automation.name}</p>
                <span
                  className={cn(
                    "size-1.5 shrink-0 rounded-full",
                    automation.status === "active"
                      ? "bg-emerald-500"
                      : automation.status === "paused"
                        ? "bg-amber-500"
                        : "bg-slate-400",
                  )}
                />
              </div>
              <p className="mt-0.5 truncate text-[11px] text-slate-500 dark:text-gray-400">{automation.trigger}</p>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

function SectionHeader({
  title,
  description,
  trailing,
}: {
  title: string;
  description: string;
  trailing?: React.ReactNode;
}) {
  return (
    <div className="flex min-h-16 items-center justify-between gap-4 border-b border-slate-200 px-4 py-3 sm:px-5 dark:border-gray-700/70">
      <div className="min-w-0">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">{title}</h2>
        <p className="mt-0.5 truncate text-xs text-slate-500 dark:text-gray-400">{description}</p>
      </div>
      {trailing}
    </div>
  );
}
