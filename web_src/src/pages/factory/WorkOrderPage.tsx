import { ArrowLeft, CheckCircle2, CircleDot, Clock3, Factory, GitPullRequest, ListPlus, XCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";

import type { SoftwareFactory, WorkOrder, WorkOrderEvent, WorkOrderEventKind, WorkOrderState } from "./factoryTypes";
import { WorkOrderStateBadge } from "./WorkOrderStateBadge";

interface WorkOrderPageProps {
  factory: SoftwareFactory;
  workOrder: WorkOrder;
  events: WorkOrderEvent[];
  onBack: () => void;
  onApprove: () => void;
}

export function WorkOrderPage({ factory, workOrder, events, onBack, onApprove }: WorkOrderPageProps) {
  const isDraft = workOrder.state === "draft";

  return (
    <div className="min-h-screen bg-slate-50 text-slate-950 dark:bg-gray-950 dark:text-gray-100">
      <main className="mx-auto w-full max-w-[1040px] px-4 py-6 sm:px-6 sm:py-8 lg:px-10">
        <Button type="button" variant="ghost" size="sm" className="-ml-3" onClick={onBack}>
          <ArrowLeft aria-hidden />
          {factory.name}
        </Button>

        <header className="mt-5 border-b border-slate-200 pb-7 dark:border-gray-800">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0">
              <div className="mb-2 flex items-center gap-2 text-xs font-medium text-slate-500 dark:text-gray-400">
                <Factory className="size-4" aria-hidden />
                Work Order
              </div>
              <h1 className="text-2xl font-semibold leading-8 text-slate-950 dark:text-white">{workOrder.title}</h1>
            </div>
            <WorkOrderStateBadge state={workOrder.state} />
          </div>
          <p className="mt-4 max-w-3xl whitespace-pre-wrap text-sm leading-6 text-slate-700 dark:text-gray-300">
            {workOrder.description}
          </p>
          <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-slate-500 dark:text-gray-400">
            <span>Created {formatTimeAgo(new Date(workOrder.createdAt))}</span>
            {workOrder.automations.length > 0 && (
              <span>{workOrder.automations.map((automation) => automation.name).join(", ")}</span>
            )}
            {workOrder.primaryPullRequest && <PullRequestLink workOrder={workOrder} />}
          </div>
        </header>

        {isDraft && (
          <section className="mt-6 flex flex-col gap-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-4 dark:border-amber-900 dark:bg-amber-950/40 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-sm font-medium text-amber-950 dark:text-amber-200">Ready for your approval</h2>
              <p className="mt-1 text-xs leading-5 text-amber-800 dark:text-amber-300">
                Approval makes this Work Order available to the implementation Automation.
              </p>
            </div>
            <Button type="button" size="sm" onClick={onApprove}>
              <ListPlus aria-hidden />
              Approve and queue
            </Button>
          </section>
        )}

        <section className="mt-8" aria-labelledby="history-heading">
          <div className="flex items-end justify-between gap-4">
            <div>
              <h2 id="history-heading" className="text-base font-semibold">
                History
              </h2>
              <p className="mt-1 text-xs text-slate-500 dark:text-gray-400">
                The durable record of this implementation.
              </p>
            </div>
            <span className="text-xs text-slate-400 dark:text-gray-500">
              {events.length} {events.length === 1 ? "event" : "events"}
            </span>
          </div>

          <WorkOrderTimeline events={events} />
        </section>
      </main>
    </div>
  );
}

function PullRequestLink({ workOrder }: { workOrder: WorkOrder }) {
  const pullRequest = workOrder.primaryPullRequest;
  if (!pullRequest) return null;

  return (
    <a
      href={pullRequest.url}
      target="_blank"
      rel="noreferrer"
      className="inline-flex items-center gap-1.5 font-medium text-blue-700 hover:underline dark:text-blue-400"
    >
      <GitPullRequest className="size-3.5" aria-hidden />
      {pullRequest.repository} #{pullRequest.number}
    </a>
  );
}

const eventPresentation = {
  created: { icon: Clock3, label: "Created", tone: "text-slate-600 dark:text-gray-300" },
  approved: { icon: CheckCircle2, label: "Approved", tone: "text-blue-700 dark:text-blue-300" },
  started: { icon: CircleDot, label: "Automation started", tone: "text-violet-700 dark:text-violet-300" },
  "pull-request": { icon: GitPullRequest, label: "Pull request", tone: "text-sky-700 dark:text-sky-300" },
  outcome: { icon: CheckCircle2, label: "Outcome", tone: "text-emerald-700 dark:text-emerald-300" },
} as const satisfies Record<WorkOrderEventKind, { icon: typeof Clock3; label: string; tone: string }>;

function WorkOrderTimeline({ events }: { events: WorkOrderEvent[] }) {
  return (
    <ol className="mt-6">
      {events.map((event, index) => (
        <TimelineEvent key={event.id} event={event} isLast={index === events.length - 1} />
      ))}
    </ol>
  );
}

function TimelineEvent({ event, isLast }: { event: WorkOrderEvent; isLast: boolean }) {
  const presentation = eventPresentation[event.kind];
  const EventIcon = outcomeIcon(event) ?? presentation.icon;

  return (
    <li className="grid grid-cols-[32px_minmax(0,1fr)] gap-3">
      <div className="flex flex-col items-center">
        <span
          className={cn(
            "z-10 flex size-8 items-center justify-center rounded-full border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-900",
            outcomeTone(event) ?? presentation.tone,
          )}
        >
          <EventIcon className="size-4" aria-hidden />
        </span>
        {!isLast && <span className="w-px flex-1 bg-slate-200 dark:bg-gray-800" aria-hidden />}
      </div>

      <article className={cn("min-w-0 pb-7", !isLast && "min-h-24")}>
        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <div>
            <span className={cn("text-xs font-medium", outcomeTone(event) ?? presentation.tone)}>
              {presentation.label}
            </span>
            <h3 className="mt-1 text-sm font-medium text-slate-950 dark:text-gray-100">{event.summary}</h3>
          </div>
          <time className="text-xs text-slate-400 dark:text-gray-500">{formatTimeAgo(new Date(event.occurredAt))}</time>
        </div>
        <p className="mt-1 text-xs text-slate-500 dark:text-gray-400">{event.actor}</p>
        {event.detail && (
          <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-600 dark:text-gray-300">{event.detail}</p>
        )}
        {event.pullRequest && (
          <a
            href={event.pullRequest.url}
            target="_blank"
            rel="noreferrer"
            className="mt-2 inline-flex items-center gap-1.5 text-sm font-medium text-blue-700 hover:underline dark:text-blue-400"
          >
            <GitPullRequest className="size-4" aria-hidden />
            {event.pullRequest.repository} #{event.pullRequest.number}
          </a>
        )}
      </article>
    </li>
  );
}

function outcomeIcon(event: WorkOrderEvent) {
  if (event.kind !== "outcome") return undefined;
  return event.outcome === "unsuccessful" ? XCircle : CheckCircle2;
}

function outcomeTone(event: WorkOrderEvent) {
  if (event.kind !== "outcome") return undefined;
  return outcomeStateTone(event.outcome ?? "successful");
}

function outcomeStateTone(state: Extract<WorkOrderState, "successful" | "unsuccessful">) {
  return state === "successful" ? "text-emerald-700 dark:text-emerald-300" : "text-red-700 dark:text-red-300";
}
