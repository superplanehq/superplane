import {
  CircleCheck,
  CircleDashed,
  CircleX,
  Clock3,
  Factory,
  GitBranch,
  LoaderCircle,
  Pause,
  Plus,
  Workflow,
} from "lucide-react";
import { useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";

import type {
  Automation,
  SoftwareFactory,
  WorkOrder,
  WorkOrderAutomation,
  WorkOrderAutomationState,
  WorkOrderState,
} from "./factoryTypes";
import { WorkOrderStateBadge } from "./WorkOrderStateBadge";

type WorkOrderScope = "all" | "mine";
type WorkOrderStateFilter = "all" | WorkOrderState;

const stateFilters: Array<{ label: string; value: WorkOrderStateFilter }> = [
  { label: "All statuses", value: "all" },
  { label: "Draft", value: "draft" },
  { label: "Ready", value: "ready" },
  { label: "Running", value: "running" },
  { label: "Successful", value: "successful" },
  { label: "Unsuccessful", value: "unsuccessful" },
];

const automationStatePresentation: Record<
  Automation["state"],
  { label: string; className: string; icon: typeof Workflow }
> = {
  idle: { label: "Idle", className: "text-slate-500 dark:text-gray-400", icon: CircleDashed },
  running: { label: "Running", className: "text-violet-700 dark:text-violet-300", icon: LoaderCircle },
  paused: { label: "Paused", className: "text-amber-700 dark:text-amber-300", icon: Pause },
};

const workOrderAutomationPresentation: Record<
  WorkOrderAutomationState,
  { label: string; className: string; icon: typeof Workflow }
> = {
  planned: { label: "Planned", className: "text-slate-500 dark:text-gray-400", icon: Clock3 },
  running: { label: "Running", className: "text-violet-700 dark:text-violet-300", icon: LoaderCircle },
  done: { label: "Done", className: "text-emerald-700 dark:text-emerald-300", icon: CircleCheck },
  failed: { label: "Failed", className: "text-red-700 dark:text-red-300", icon: CircleX },
};

interface SoftwareFactoryPageProps {
  factory: SoftwareFactory;
  workOrders: WorkOrder[];
  automations: Automation[];
  currentUserId: string;
  onNewWorkOrder: () => void;
  onOpenWorkOrder: (workOrder: WorkOrder) => void;
  onCreateAutomation: () => void;
  onOpenAutomation: (automation: Automation) => void;
}

export function SoftwareFactoryPage({
  factory,
  workOrders,
  automations,
  currentUserId,
  onNewWorkOrder,
  onOpenWorkOrder,
  onCreateAutomation,
  onOpenAutomation,
}: SoftwareFactoryPageProps) {
  return (
    <div className="min-h-screen bg-slate-50 text-slate-950 dark:bg-gray-950 dark:text-gray-100">
      <main className="mx-auto w-full max-w-[1480px] px-4 py-6 sm:px-6 sm:py-8 lg:px-10">
        <FactoryHeader factory={factory} onNewWorkOrder={onNewWorkOrder} />

        <div className="mt-8 grid items-start gap-8 xl:grid-cols-[minmax(0,1fr)_360px]">
          <WorkOrdersSection workOrders={workOrders} currentUserId={currentUserId} onOpenWorkOrder={onOpenWorkOrder} />
          <AutomationsSection
            automations={automations}
            onCreateAutomation={onCreateAutomation}
            onOpenAutomation={onOpenAutomation}
          />
        </div>
      </main>
    </div>
  );
}

function FactoryHeader({ factory, onNewWorkOrder }: { factory: SoftwareFactory; onNewWorkOrder: () => void }) {
  return (
    <header className="flex flex-col gap-5 border-b border-slate-200 pb-7 dark:border-gray-800 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        <div className="mb-2 flex items-center gap-2 text-xs font-medium text-slate-500 dark:text-gray-400">
          <Factory className="size-4" aria-hidden />
          Software Factory
        </div>
        <h1 className="text-2xl font-semibold text-slate-950 dark:text-white">{factory.name}</h1>
        {factory.description && (
          <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-600 dark:text-gray-400">{factory.description}</p>
        )}
      </div>
      <Button type="button" onClick={onNewWorkOrder}>
        <Plus aria-hidden />
        New Work Order
      </Button>
    </header>
  );
}

function WorkOrdersSection({
  workOrders,
  currentUserId,
  onOpenWorkOrder,
}: {
  workOrders: WorkOrder[];
  currentUserId: string;
  onOpenWorkOrder: (workOrder: WorkOrder) => void;
}) {
  const [scope, setScope] = useState<WorkOrderScope>("all");
  const [stateFilter, setStateFilter] = useState<WorkOrderStateFilter>("all");
  const filteredWorkOrders = useMemo(
    () =>
      workOrders.filter(
        (workOrder) =>
          (scope === "all" || workOrder.createdByUserId === currentUserId) &&
          (stateFilter === "all" || workOrder.state === stateFilter),
      ),
    [currentUserId, scope, stateFilter, workOrders],
  );
  const hasActiveFilters = scope !== "all" || stateFilter !== "all";

  return (
    <section aria-labelledby="work-orders-heading">
      <SectionHeading
        id="work-orders-heading"
        title="Work Orders"
        description="The implementation queue for this Factory."
        count={filteredWorkOrders.length}
      />

      <WorkOrderFilters
        scope={scope}
        stateFilter={stateFilter}
        onScopeChange={setScope}
        onStateChange={setStateFilter}
      />

      {filteredWorkOrders.length === 0 ? (
        <EmptyWorkOrders filtered={hasActiveFilters} />
      ) : (
        <div className="mt-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900">
          {filteredWorkOrders.map((workOrder, index) => (
            <WorkOrderRow
              key={workOrder.id}
              workOrder={workOrder}
              hasDivider={index > 0}
              onOpen={() => onOpenWorkOrder(workOrder)}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function WorkOrderFilters({
  scope,
  stateFilter,
  onScopeChange,
  onStateChange,
}: {
  scope: WorkOrderScope;
  stateFilter: WorkOrderStateFilter;
  onScopeChange: (scope: WorkOrderScope) => void;
  onStateChange: (state: WorkOrderStateFilter) => void;
}) {
  return (
    <div className="mt-4 flex flex-col gap-3 border-y border-slate-200 py-3 dark:border-gray-800 lg:flex-row lg:items-center lg:justify-between">
      <FilterGroup label="Owner">
        <FilterButton label="All" value="all" selected={scope === "all"} onSelect={onScopeChange} />
        <FilterButton label="Mine" value="mine" selected={scope === "mine"} onSelect={onScopeChange} />
      </FilterGroup>

      <FilterGroup label="Status">
        {stateFilters.map((filter) => (
          <FilterButton
            key={filter.value}
            label={filter.label}
            value={filter.value}
            selected={stateFilter === filter.value}
            onSelect={onStateChange}
          />
        ))}
      </FilterGroup>
    </div>
  );
}

function FilterGroup({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <span className="w-12 text-xs font-medium text-slate-500 dark:text-gray-400 lg:w-auto">{label}</span>
      <div className="inline-flex flex-wrap gap-1" role="group" aria-label={`${label} filter`}>
        {children}
      </div>
    </div>
  );
}

function FilterButton<T extends string>({
  label,
  value,
  selected,
  onSelect,
}: {
  label: string;
  value: T;
  selected: boolean;
  onSelect: (value: T) => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={() => onSelect(value)}
      className={cn(
        "h-7 rounded-md px-2.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400",
        selected
          ? "bg-slate-900 text-white dark:bg-gray-100 dark:text-gray-950"
          : "text-slate-600 hover:bg-slate-100 hover:text-slate-950 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
      )}
    >
      {label}
    </button>
  );
}

function WorkOrderRow({
  workOrder,
  hasDivider,
  onOpen,
}: {
  workOrder: WorkOrder;
  hasDivider: boolean;
  onOpen: () => void;
}) {
  return (
    <article
      className={cn(
        "grid min-h-24 gap-3 px-4 py-4 transition-colors hover:bg-slate-50 dark:hover:bg-gray-800/60 sm:grid-cols-[minmax(0,1fr)_140px] sm:items-center sm:px-5",
        hasDivider && "border-t border-slate-200 dark:border-gray-800",
      )}
    >
      <div className="min-w-0">
        <button
          type="button"
          onClick={onOpen}
          className="block max-w-full truncate text-left text-sm font-medium text-slate-950 hover:underline dark:text-gray-100"
        >
          {workOrder.title}
        </button>
        <p className="mt-1 line-clamp-2 text-xs leading-5 text-slate-500 dark:text-gray-400">{workOrder.description}</p>
        <WorkOrderAutomations automations={workOrder.automations} />
        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500 dark:text-gray-500">
          <span>Created by {workOrder.createdByName}</span>
          <span>Updated {formatTimeAgo(new Date(workOrder.updatedAt))}</span>
          {workOrder.primaryPullRequest && (
            <a
              href={workOrder.primaryPullRequest.url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-blue-700 hover:underline dark:text-blue-400"
            >
              <GitBranch className="size-3.5" aria-hidden />
              {workOrder.primaryPullRequest.repository} #{workOrder.primaryPullRequest.number}
            </a>
          )}
        </div>
      </div>

      <div className="sm:justify-self-start">
        <WorkOrderStateBadge state={workOrder.state} />
      </div>
    </article>
  );
}

function WorkOrderAutomations({ automations }: { automations: WorkOrderAutomation[] }) {
  if (automations.length === 0) return null;

  return (
    <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1.5">
      {automations.map((automation) => (
        <WorkOrderAutomationStatus key={automation.id} automation={automation} />
      ))}
    </div>
  );
}

function WorkOrderAutomationStatus({ automation }: { automation: WorkOrderAutomation }) {
  const presentation = workOrderAutomationPresentation[automation.state];
  const StateIcon = presentation.icon;

  return (
    <span
      className="inline-flex items-center gap-1.5 text-xs text-slate-700 dark:text-gray-300"
      aria-label={`${automation.name}: ${presentation.label}`}
    >
      <StateIcon
        className={cn(
          "size-3.5",
          presentation.className,
          automation.state === "running" && "animate-spin motion-reduce:animate-none",
        )}
        aria-hidden
      />
      <span>{automation.name}</span>
      <span className={presentation.className}>· {presentation.label}</span>
    </span>
  );
}

function EmptyWorkOrders({ filtered }: { filtered: boolean }) {
  return (
    <div className="mt-4 flex min-h-48 flex-col items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white px-6 text-center dark:border-gray-700 dark:bg-gray-900">
      <GitBranch className="size-6 text-slate-400" aria-hidden />
      <p className="mt-3 text-sm font-medium">{filtered ? "No matching Work Orders" : "No Work Orders yet"}</p>
      <p className="mt-1 max-w-sm text-xs leading-5 text-slate-500 dark:text-gray-400">
        {filtered
          ? "Try another owner or status filter."
          : "Create a draft when you have one concrete software change ready to delegate."}
      </p>
    </div>
  );
}

function AutomationsSection({
  automations,
  onCreateAutomation,
  onOpenAutomation,
}: {
  automations: Automation[];
  onCreateAutomation: () => void;
  onOpenAutomation: (automation: Automation) => void;
}) {
  return (
    <section aria-labelledby="automations-heading">
      <div className="flex items-start justify-between gap-4">
        <SectionHeading
          id="automations-heading"
          title="Automations"
          description="Canvas workflows that process approved work."
          count={automations.length}
        />
        <Button type="button" size="sm" variant="outline" onClick={onCreateAutomation}>
          <Plus aria-hidden />
          New
        </Button>
      </div>

      <div className="mt-4 space-y-3">
        {automations.length === 0 ? (
          <div className="flex min-h-48 flex-col items-center justify-center rounded-lg border border-slate-200 bg-white px-6 text-center dark:border-gray-800 dark:bg-gray-900">
            <Workflow className="size-6 text-slate-400" aria-hidden />
            <p className="mt-3 text-sm font-medium">No Automation configured</p>
            <p className="mt-1 text-xs leading-5 text-slate-500 dark:text-gray-400">
              Add one Canvas that listens for ready Work Orders.
            </p>
            <Button type="button" size="sm" className="mt-4" onClick={onCreateAutomation}>
              <Plus aria-hidden />
              New Automation
            </Button>
          </div>
        ) : (
          automations.map((automation) => (
            <AutomationRow key={automation.id} automation={automation} onOpen={() => onOpenAutomation(automation)} />
          ))
        )}
      </div>
    </section>
  );
}

function AutomationRow({ automation, onOpen }: { automation: Automation; onOpen: () => void }) {
  const statePresentation = automationStatePresentation[automation.state];
  const StateIcon = statePresentation.icon;

  return (
    <article className="rounded-lg border border-slate-200 bg-white px-4 py-4 dark:border-gray-800 dark:bg-gray-900">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <button
            type="button"
            onClick={onOpen}
            className="truncate text-left text-sm font-medium text-slate-950 hover:underline dark:text-gray-100"
          >
            {automation.name}
          </button>
          <p className="mt-1 text-xs leading-5 text-slate-500 dark:text-gray-400">{automation.description}</p>
        </div>
        <span
          className={cn("inline-flex shrink-0 items-center gap-1 text-xs font-medium", statePresentation.className)}
        >
          <StateIcon
            className={cn("size-3.5", automation.state === "running" && "animate-spin motion-reduce:animate-none")}
            aria-hidden
          />
          {statePresentation.label}
        </span>
      </div>
      <dl className="mt-4 grid grid-cols-2 border-t border-slate-100 pt-3 dark:border-gray-800">
        <AutomationMetric count={automation.runningCount} label="Running now" />
        <AutomationMetric count={automation.queuedCount} label="In queue" divided />
      </dl>
    </article>
  );
}

function AutomationMetric({ count, label, divided = false }: { count: number; label: string; divided?: boolean }) {
  return (
    <div
      className={cn(divided && "border-l border-slate-100 pl-4 dark:border-gray-800")}
      aria-label={`${count} ${label.toLowerCase()}`}
    >
      <dt className="text-xs text-slate-500 dark:text-gray-400">{label}</dt>
      <dd className="mt-1 text-lg font-semibold text-slate-950 dark:text-gray-100">{count}</dd>
    </div>
  );
}

function SectionHeading({
  id,
  title,
  description,
  count,
}: {
  id: string;
  title: string;
  description: string;
  count: number;
}) {
  return (
    <div>
      <div className="flex items-center gap-2">
        <h2 id={id} className="text-base font-semibold text-slate-950 dark:text-gray-100">
          {title}
        </h2>
        <span className="text-xs text-slate-400 dark:text-gray-500">{count}</span>
      </div>
      <p className="mt-1 text-xs text-slate-500 dark:text-gray-400">{description}</p>
    </div>
  );
}
