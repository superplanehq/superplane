import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  Bot,
  CheckCircle2,
  ChevronRight,
  CircleAlert,
  CircleX,
  Clock3,
  GitBranch,
  LoaderCircle,
  MessageCircleQuestion,
  RotateCcw,
  type LucideIcon,
} from "lucide-react";

import type { FactoryWorkItem, WorkItemStatus } from "./types";

interface FactoryWorkDashboardProps {
  workItems: FactoryWorkItem[];
  onOpenWorkItem: (workItem: FactoryWorkItem) => void;
}

interface WorkTableSectionProps {
  title: string;
  description: string;
  emptyMessage: string;
  items: FactoryWorkItem[];
  status: Exclude<WorkItemStatus, "attention">;
  onOpenWorkItem: (workItem: FactoryWorkItem) => void;
}

const sectionClassName =
  "overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900";

const statusIcons: Record<Exclude<WorkItemStatus, "attention">, LucideIcon> = {
  running: LoaderCircle,
  done: CheckCircle2,
  failed: CircleX,
};

export function FactoryWorkDashboard({ workItems, onOpenWorkItem }: FactoryWorkDashboardProps) {
  const attentionItems = workItems.filter((workItem) => workItem.status === "attention");
  const runningItems = workItems.filter((workItem) => workItem.status === "running");
  const completedItems = workItems.filter((workItem) => workItem.status === "done");
  const failedItems = workItems.filter((workItem) => workItem.status === "failed");

  return (
    <div className="space-y-4">
      <FactoryAttentionSection items={attentionItems} onOpenWorkItem={onOpenWorkItem} />
      <WorkTableSection
        title="Running"
        description="Factory work currently moving through build, review, and verification."
        emptyMessage="No factory work is running."
        items={runningItems}
        status="running"
        onOpenWorkItem={onOpenWorkItem}
      />
      <WorkTableSection
        title="Recently done"
        description="Work delivered to the repository."
        emptyMessage="No work has completed yet."
        items={completedItems}
        status="done"
        onOpenWorkItem={onOpenWorkItem}
      />
      <WorkTableSection
        title="Unsuccessful"
        description="Runs that stopped and may need intervention."
        emptyMessage="No unsuccessful work items."
        items={failedItems}
        status="failed"
        onOpenWorkItem={onOpenWorkItem}
      />
    </div>
  );
}

export function FactoryAttentionSection({
  items,
  onOpenWorkItem,
}: {
  items: FactoryWorkItem[];
  onOpenWorkItem: (workItem: FactoryWorkItem) => void;
}) {
  return (
    <section
      aria-labelledby="needs-attention-heading"
      className={cn(sectionClassName, "border-amber-300 dark:border-amber-800")}
    >
      <div className="flex min-h-16 items-center justify-between gap-4 border-b border-amber-200 bg-amber-50/70 px-4 py-3 sm:px-5 dark:border-amber-900 dark:bg-amber-950/25">
        <div className="flex min-w-0 items-start gap-3">
          <span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300">
            <MessageCircleQuestion className="size-4" />
          </span>
          <div className="min-w-0">
            <h2 id="needs-attention-heading" className="text-sm font-semibold text-slate-900 dark:text-gray-100">
              Needs attention
            </h2>
            <p className="mt-0.5 text-xs text-slate-600 dark:text-gray-400">
              Work paused for a decision, approval, or clarification.
            </p>
          </div>
        </div>
        <Badge
          variant="outline"
          className="shrink-0 border-amber-300 bg-white text-amber-800 dark:border-amber-700 dark:bg-gray-900 dark:text-amber-300"
        >
          {items.length} waiting
        </Badge>
      </div>

      {items.length === 0 ? (
        <EmptyWorkState icon={CheckCircle2} message="No work needs your input." tone="done" />
      ) : (
        <div className="divide-y divide-slate-200 dark:divide-gray-700/70">
          {items.map((workItem) => (
            <button
              key={workItem.id}
              type="button"
              onClick={() => onOpenWorkItem(workItem)}
              aria-label={`Review ${workItem.id}: ${workItem.title}`}
              className="group grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-4 py-3.5 text-left transition-colors hover:bg-amber-50/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-amber-500 sm:px-5 dark:hover:bg-amber-950/20"
            >
              <span className="flex min-w-0 items-start gap-3">
                <CircleAlert className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
                <span className="min-w-0">
                  <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
                    <span className="text-xs font-medium text-slate-500 dark:text-gray-400">{workItem.id}</span>
                    <span className="text-sm font-medium text-slate-900 dark:text-gray-100">{workItem.title}</span>
                    <Badge variant="outline" className="text-[10px] text-slate-600 dark:text-gray-300">
                      {workItem.stage}
                    </Badge>
                  </span>
                  <span className="mt-1 block text-xs text-slate-600 dark:text-gray-400">{workItem.detail}</span>
                </span>
              </span>
              <span className="flex shrink-0 items-center gap-3">
                <span className="hidden text-right sm:block">
                  <span className="block text-xs text-slate-500 dark:text-gray-400">{workItem.updatedAt}</span>
                  <span className="mt-0.5 flex items-center justify-end gap-1 text-[11px] text-slate-500 dark:text-gray-400">
                    <GitBranch className="size-3" />
                    {workItem.branch}
                  </span>
                </span>
                <ChevronRight className="size-4 text-slate-400 transition-transform group-hover:translate-x-0.5" />
              </span>
            </button>
          ))}
        </div>
      )}
    </section>
  );
}

function WorkTableSection({ title, description, emptyMessage, items, status, onOpenWorkItem }: WorkTableSectionProps) {
  const Icon = statusIcons[status];

  return (
    <section aria-labelledby={`${status}-work-heading`} className={sectionClassName}>
      <div className="flex min-h-16 items-center justify-between gap-4 border-b border-slate-200 px-4 py-3 sm:px-5 dark:border-gray-700/70">
        <div className="flex min-w-0 items-start gap-3">
          <span
            className={cn(
              "mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md",
              statusIconClassName(status),
            )}
          >
            <Icon className={cn("size-4", status === "running" && "animate-spin")} />
          </span>
          <div className="min-w-0">
            <h2 id={`${status}-work-heading`} className="text-sm font-semibold text-slate-900 dark:text-gray-100">
              {title}
            </h2>
            <p className="mt-0.5 text-xs text-slate-500 dark:text-gray-400">{description}</p>
          </div>
        </div>
        <span className="shrink-0 text-xs font-medium text-slate-500 tabular-nums dark:text-gray-400">
          {items.length}
        </span>
      </div>

      {items.length === 0 ? (
        <EmptyWorkState icon={Icon} message={emptyMessage} tone={status} />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] border-collapse text-left">
            <thead>
              <tr className="border-b border-slate-200 text-[11px] font-medium text-slate-500 dark:border-gray-700/70 dark:text-gray-400">
                <th scope="col" className="w-[30%] px-4 py-2 font-medium sm:px-5">
                  Work item
                </th>
                <th scope="col" className="w-[12%] px-3 py-2 font-medium">
                  Stage
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  Activity
                </th>
                <th scope="col" className="w-[10%] px-3 py-2 text-right font-medium">
                  Agents
                </th>
                <th scope="col" className="w-[12%] px-4 py-2 text-right font-medium sm:px-5">
                  {status === "running" ? "Elapsed" : "Updated"}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-gray-700/70">
              {items.map((workItem) => (
                <tr key={workItem.id} className="group hover:bg-slate-50 dark:hover:bg-gray-800/60">
                  <th scope="row" className="px-4 py-3 font-normal sm:px-5">
                    <button
                      type="button"
                      onClick={() => onOpenWorkItem(workItem)}
                      className="flex min-w-0 items-center gap-2 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
                    >
                      <span className={cn("size-2 shrink-0 rounded-full", statusDotClassName(status))} />
                      <span className="min-w-0">
                        <span className="block text-[11px] font-medium text-slate-500 dark:text-gray-400">
                          {workItem.id}
                        </span>
                        <span className="block truncate text-xs font-medium text-slate-900 dark:text-gray-100">
                          {workItem.title}
                        </span>
                      </span>
                    </button>
                  </th>
                  <td className="px-3 py-3 text-xs text-slate-700 dark:text-gray-300">{workItem.stage}</td>
                  <td className="px-3 py-3 text-xs text-slate-600 dark:text-gray-400">{workItem.detail}</td>
                  <td className="px-3 py-3 text-right text-xs text-slate-600 tabular-nums dark:text-gray-400">
                    <span className="inline-flex items-center gap-1">
                      <Bot className="size-3" />
                      {workItem.agentCount}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right text-xs text-slate-600 tabular-nums sm:px-5 dark:text-gray-400">
                    <span className="inline-flex items-center gap-1">
                      <Clock3 className="size-3" />
                      {status === "running" ? workItem.elapsed : workItem.updatedAt}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {status === "failed" && items.length > 0 && (
        <div className="flex justify-end border-t border-slate-200 px-4 py-2 sm:px-5 dark:border-gray-700/70">
          <Button type="button" variant="ghost" size="sm">
            <RotateCcw />
            Review failures
          </Button>
        </div>
      )}
    </section>
  );
}

function EmptyWorkState({
  icon: Icon,
  message,
  tone,
}: {
  icon: LucideIcon;
  message: string;
  tone: Exclude<WorkItemStatus, "attention">;
}) {
  return (
    <div className="flex min-h-28 flex-col items-center justify-center px-4 py-5 text-center">
      <span className={cn("flex size-8 items-center justify-center rounded-md", statusIconClassName(tone))}>
        <Icon className="size-4" />
      </span>
      <p className="mt-2 text-xs text-slate-500 dark:text-gray-400">{message}</p>
    </div>
  );
}

function statusIconClassName(status: Exclude<WorkItemStatus, "attention">) {
  if (status === "running") return "bg-sky-50 text-sky-700 dark:bg-sky-950/60 dark:text-sky-300";
  if (status === "done") return "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300";
  return "bg-red-50 text-red-700 dark:bg-red-950/50 dark:text-red-300";
}

function statusDotClassName(status: Exclude<WorkItemStatus, "attention">) {
  if (status === "running") return "bg-sky-500";
  if (status === "done") return "bg-emerald-500";
  return "bg-red-500";
}
