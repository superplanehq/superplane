import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  Activity,
  Bot,
  Check,
  CheckCircle2,
  ChevronRight,
  CircleDot,
  Clock3,
  Code2,
  GitBranch,
  GitPullRequest,
  Inbox,
  ListTodo,
  Plus,
  Rocket,
  Settings2,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { NewWorkDialog } from "./NewWorkDialog";
import type {
  CreateWorkRequest,
  DeliveryEvent,
  FactoryMetric,
  FactoryStage,
  FactoryStageState,
  FactoryWorkItem,
  ThroughputPoint,
  WorkItemStatus,
  WorkspacePageData,
  WorkspaceProject,
} from "./types";

interface SoftwareFactoryHomeProps {
  data: WorkspacePageData;
  onCreateWork?: (request: CreateWorkRequest) => void;
}

const stageIcons: Record<string, LucideIcon> = {
  intake: Inbox,
  plan: ListTodo,
  build: Code2,
  verify: ShieldCheck,
  deliver: Rocket,
};

const sectionClassName =
  "overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900";

export function SoftwareFactoryHome({ data, onCreateWork }: SoftwareFactoryHomeProps) {
  const [newWorkOpen, setNewWorkOpen] = useState(false);
  const navigate = useNavigate();

  return (
    <>
      <div className="mx-auto w-full max-w-[1400px] px-4 py-5 sm:px-6 sm:py-6 lg:px-8">
        <ProjectHeader project={data.project} onNewWork={() => setNewWorkOpen(true)} />
        <MetricsBand metrics={data.metrics} />
        <div className="mt-4 space-y-4">
          <FactoryFlow stages={data.stages} />
          <div className="grid gap-4 md:grid-cols-[minmax(0,1.65fr)_minmax(15rem,0.75fr)]">
            <ActiveWork workItems={data.workItems} onOpenWorkItem={(workItem) => navigate(`work/${workItem.id}`)} />
            <aside className="space-y-4">
              <ThroughputChart points={data.throughput} />
              <RecentDelivery deliveries={data.recentDeliveries} />
            </aside>
          </div>
        </div>
      </div>
      <NewWorkDialog open={newWorkOpen} onOpenChange={setNewWorkOpen} onCreateWork={onCreateWork} />
    </>
  );
}

function ProjectHeader({ project, onNewWork }: { project: WorkspaceProject; onNewWork: () => void }) {
  return (
    <header className="mb-5 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        <div className="mb-1.5 flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400">
          <span>{project.factoryName}</span>
          <span aria-hidden="true" className="text-slate-300 dark:text-gray-600">
            /
          </span>
          <span className="inline-flex items-center gap-1.5 text-emerald-700 dark:text-emerald-400">
            <span className="size-1.5 rounded-full bg-emerald-500" />
            Operational
          </span>
        </div>
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-gray-100">{project.name}</h1>
        <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-gray-400">
          <span className="inline-flex items-center gap-1.5">
            <GitBranch className="size-3.5" />
            {project.repository}
          </span>
          <span>Default branch: {project.defaultBranch}</span>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button type="button" variant="outline" size="icon-sm" aria-label="Factory settings">
              <Settings2 />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Factory settings</TooltipContent>
        </Tooltip>
        <Button type="button" onClick={onNewWork}>
          <Plus />
          New work
        </Button>
      </div>
    </header>
  );
}

function MetricsBand({ metrics }: { metrics: FactoryMetric[] }) {
  return (
    <section
      aria-label="Factory metrics"
      className={cn(
        sectionClassName,
        "grid grid-cols-2 divide-x divide-y divide-slate-200 md:grid-cols-4 md:divide-y-0 dark:divide-gray-700/70",
      )}
    >
      {metrics.map((metric) => (
        <div key={metric.label} className="min-w-0 px-4 py-3.5">
          <p className="truncate text-xs text-slate-500 dark:text-gray-400">{metric.label}</p>
          <p className="mt-1 text-xl font-semibold text-slate-900 dark:text-gray-100">{metric.value}</p>
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
      ))}
    </section>
  );
}

function FactoryFlow({ stages }: { stages: FactoryStage[] }) {
  return (
    <section className={sectionClassName}>
      <SectionHeader
        title="Factory flow"
        description="The path every change takes from intake to delivery."
        trailing={
          <Badge
            variant="outline"
            className="border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300"
          >
            <Activity />
            Live
          </Badge>
        }
      />
      <div className="overflow-x-auto px-4 py-5 sm:px-5">
        <div className="grid min-w-[620px] grid-cols-5">
          {stages.map((stage, index) => (
            <FactoryStageItem key={stage.id} stage={stage} hasConnector={index < stages.length - 1} />
          ))}
        </div>
      </div>
    </section>
  );
}

function FactoryStageItem({ stage, hasConnector }: { stage: FactoryStage; hasConnector: boolean }) {
  const Icon = stageIcons[stage.id] ?? CircleDot;

  return (
    <div className="relative min-w-0">
      <div
        aria-hidden="true"
        className={cn(
          "absolute top-4 left-[calc(50%+1rem)] h-px w-[calc(100%-2rem)]",
          hasConnector ? stageConnectorClassName(stage.state) : "hidden",
        )}
      />
      <div className="relative flex flex-col items-center px-2 text-center">
        <div
          className={cn("flex size-8 items-center justify-center rounded-full border", stageIconClassName(stage.state))}
        >
          {stage.state === "complete" ? <Check className="size-4" /> : <Icon className="size-4" />}
        </div>
        <p className="mt-2 text-sm font-medium text-slate-800 dark:text-gray-200">{stage.label}</p>
        <p className="mt-0.5 text-xs text-slate-500 dark:text-gray-400">{stage.detail}</p>
      </div>
    </div>
  );
}

function ActiveWork({
  workItems,
  onOpenWorkItem,
}: {
  workItems: FactoryWorkItem[];
  onOpenWorkItem: (workItem: FactoryWorkItem) => void;
}) {
  return (
    <section className={sectionClassName}>
      <SectionHeader
        title="Active work"
        description={`${workItems.length} work items are moving through the factory.`}
        trailing={
          <Button type="button" size="sm" variant="ghost">
            View all
            <ChevronRight />
          </Button>
        }
      />
      <div className="divide-y divide-slate-200 dark:divide-gray-700/70">
        {workItems.map((workItem) => (
          <WorkItemRow key={workItem.id} workItem={workItem} onOpen={() => onOpenWorkItem(workItem)} />
        ))}
      </div>
    </section>
  );
}

function WorkItemRow({ workItem, onOpen }: { workItem: FactoryWorkItem; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={`Open ${workItem.id}: ${workItem.title}`}
      className="group grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-4 py-3 text-left transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400 sm:px-5 dark:hover:bg-gray-800/70"
    >
      <span className="flex min-w-0 items-start gap-3">
        <span className={cn("mt-1.5 size-2 shrink-0 rounded-full", workItemStatusDotClassName(workItem.status))} />
        <span className="min-w-0">
          <span className="flex items-center gap-2">
            <span className="shrink-0 text-xs font-medium text-slate-500 dark:text-gray-400">{workItem.id}</span>
            <span className="truncate text-sm font-medium text-slate-900 dark:text-gray-100">{workItem.title}</span>
          </span>
          <span className="mt-1 flex items-center gap-1.5 truncate text-xs text-slate-500 dark:text-gray-400">
            <GitBranch className="size-3 shrink-0" />
            {workItem.branch}
          </span>
        </span>
      </span>

      <span className="flex shrink-0 items-center gap-4">
        <span className="hidden text-right sm:block">
          <span className="block text-xs font-medium text-slate-700 dark:text-gray-300">{workItem.stage}</span>
          <span className="mt-0.5 flex items-center justify-end gap-1 text-xs text-slate-500 dark:text-gray-400">
            <Bot className="size-3" />
            {workItem.agentCount} agents
          </span>
        </span>
        <span className="flex w-12 items-center justify-end gap-1 text-xs text-slate-500 dark:text-gray-400">
          <Clock3 className="size-3" />
          {workItem.elapsed}
        </span>
        <ChevronRight className="size-4 text-slate-300 transition-transform group-hover:translate-x-0.5 dark:text-gray-600" />
      </span>
    </button>
  );
}

function ThroughputChart({ points }: { points: ThroughputPoint[] }) {
  const maxValue = Math.max(...points.map((point) => point.value), 1);
  const total = points.reduce((sum, point) => sum + point.value, 0);

  return (
    <section className={sectionClassName}>
      <SectionHeader
        title="Throughput"
        description="Merged this week"
        trailing={<span className="text-xl font-semibold text-slate-900 dark:text-gray-100">{total}</span>}
      />
      <div
        role="img"
        aria-label={`Weekly throughput: ${points.map((point) => `${point.label} ${point.value}`).join(", ")}`}
        className="px-4 pt-4 pb-3"
      >
        <div className="flex h-24 items-end gap-2">
          {points.map((point) => (
            <div key={point.label} className="flex h-full min-w-0 flex-1 items-end">
              <div
                className={cn(
                  "w-full rounded-t-sm bg-sky-500 transition-[height] dark:bg-sky-400",
                  point.value === 0 && "h-px bg-slate-200 dark:bg-gray-700",
                )}
                style={point.value === 0 ? undefined : { height: `${Math.max((point.value / maxValue) * 100, 8)}%` }}
              />
            </div>
          ))}
        </div>
        <div className="mt-2 flex gap-2">
          {points.map((point) => (
            <span
              key={point.label}
              className="min-w-0 flex-1 text-center text-[10px] text-slate-500 dark:text-gray-400"
            >
              {point.label.slice(0, 1)}
            </span>
          ))}
        </div>
      </div>
    </section>
  );
}

function RecentDelivery({ deliveries }: { deliveries: DeliveryEvent[] }) {
  return (
    <section className={sectionClassName}>
      <SectionHeader title="Recent delivery" description="Changes merged to main" />
      <div className="divide-y divide-slate-200 dark:divide-gray-700/70">
        {deliveries.map((delivery) => (
          <div key={delivery.id} className="flex items-start gap-3 px-4 py-3">
            <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-400">
              <CheckCircle2 className="size-3.5" />
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-slate-800 dark:text-gray-200">{delivery.title}</p>
              <p className="mt-0.5 flex items-center gap-1 text-xs text-slate-500 dark:text-gray-400">
                <GitPullRequest className="size-3" />
                {delivery.reference}
                <span aria-hidden="true">·</span>
                {delivery.timestamp}
              </p>
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

function stageIconClassName(state: FactoryStageState) {
  if (state === "complete") {
    return "border-emerald-500 bg-emerald-500 text-white dark:border-emerald-400 dark:bg-emerald-500";
  }
  if (state === "active") {
    return "border-sky-500 bg-sky-50 text-sky-700 ring-4 ring-sky-100 dark:border-sky-400 dark:bg-sky-950 dark:text-sky-300 dark:ring-sky-950/70";
  }
  return "border-slate-300 bg-white text-slate-400 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-500";
}

function stageConnectorClassName(state: FactoryStageState) {
  if (state === "complete") return "bg-emerald-400 dark:bg-emerald-600";
  if (state === "active") return "bg-sky-300 dark:bg-sky-700";
  return "bg-slate-200 dark:bg-gray-700";
}

function workItemStatusDotClassName(status: WorkItemStatus) {
  if (status === "running") return "bg-sky-500";
  if (status === "attention") return "bg-red-500";
  return "bg-amber-500";
}
