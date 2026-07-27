import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  Bot,
  Check,
  CheckCircle2,
  Circle,
  ClipboardCheck,
  GitCommitHorizontal,
  MessageSquareMore,
  PencilLine,
  Play,
  UserRound,
  type LucideIcon,
} from "lucide-react";

import type { TimelineEventKind, WorkPlan, WorkPlanStep, WorkTimelineEvent } from "./activeWorkItemTypes";

interface ActiveWorkTimelineProps {
  events: WorkTimelineEvent[];
  plan: WorkPlan;
  planApproved: boolean;
  onApprovePlan: () => void;
  onSteer: (eventTitle: string) => void;
}

const eventIcons: Record<TimelineEventKind, LucideIcon> = {
  request: ClipboardCheck,
  agent: Bot,
  user: UserRound,
  system: GitCommitHorizontal,
  approval: CheckCircle2,
  progress: Play,
};

export function ActiveWorkTimeline({ events, plan, planApproved, onApprovePlan, onSteer }: ActiveWorkTimelineProps) {
  return (
    <ol aria-label="Work chronology" className="pt-6">
      {events.map((event, index) => (
        <TimelineEvent
          key={event.id}
          event={event}
          isLast={index === events.length - 1}
          plan={event.kind === "approval" && event.id === "plan-v2" ? plan : undefined}
          planApproved={planApproved}
          onApprovePlan={onApprovePlan}
          onSteer={onSteer}
        />
      ))}
    </ol>
  );
}

function TimelineEvent({
  event,
  isLast,
  plan,
  planApproved,
  onApprovePlan,
  onSteer,
}: {
  event: WorkTimelineEvent;
  isLast: boolean;
  plan?: WorkPlan;
  planApproved: boolean;
  onApprovePlan: () => void;
  onSteer: (eventTitle: string) => void;
}) {
  const Icon = eventIcons[event.kind];

  return (
    <li className="grid grid-cols-[1.75rem_minmax(0,1fr)] gap-x-2 px-4 sm:grid-cols-[3.5rem_2rem_minmax(0,1fr)] sm:gap-x-3 sm:px-6">
      <time className="hidden pt-1 text-right text-[11px] text-slate-400 tabular-nums sm:block dark:text-gray-500">
        {event.time}
      </time>
      <div className="relative flex justify-center">
        {!isLast ? (
          <span
            aria-hidden="true"
            className="absolute top-7 bottom-0 left-1/2 w-px -translate-x-1/2 bg-slate-200 dark:bg-gray-700"
          />
        ) : null}
        <span
          className={cn(
            "relative z-10 flex size-7 items-center justify-center rounded-full border bg-white dark:bg-gray-900",
            eventMarkerClassName(event.kind),
          )}
        >
          <Icon className="size-3.5" />
        </span>
      </div>

      <article className={cn("min-w-0 pb-8", isLast && "pb-6")}>
        <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1">
          <div className="min-w-0">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-gray-100">{event.title}</h3>
            <p className="mt-0.5 text-xs text-slate-500 dark:text-gray-400">
              <span className="sm:hidden">{event.time} · </span>
              {event.actor}
            </p>
          </div>
          <Button
            type="button"
            size="xs"
            variant="ghost"
            aria-label={`Steer from ${event.title}`}
            onClick={() => onSteer(event.title)}
            className="text-slate-500 dark:text-gray-400"
          >
            <MessageSquareMore />
            Steer from here
          </Button>
        </div>

        <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-700 dark:text-gray-300">{event.description}</p>

        {event.details?.length ? (
          <div className="mt-2 flex flex-wrap gap-1.5">
            {event.details.map((detail) => (
              <Badge
                key={detail}
                variant="outline"
                className="border-slate-200 bg-slate-50 font-normal text-slate-600 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300"
              >
                {detail}
              </Badge>
            ))}
          </div>
        ) : null}

        {plan ? (
          <PlanCheckpoint
            plan={plan}
            approved={planApproved}
            onApprove={onApprovePlan}
            onRequestChanges={() => onSteer(event.title)}
          />
        ) : null}
      </article>
    </li>
  );
}

function PlanCheckpoint({
  plan,
  approved,
  onApprove,
  onRequestChanges,
}: {
  plan: WorkPlan;
  approved: boolean;
  onApprove: () => void;
  onRequestChanges: () => void;
}) {
  return (
    <section className="mt-4 overflow-hidden rounded-lg border border-slate-200 bg-slate-50/60 dark:border-gray-700 dark:bg-gray-950/40">
      <div className="flex flex-col gap-3 border-b border-slate-200 px-4 py-3 sm:flex-row sm:items-start sm:justify-between dark:border-gray-700">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-semibold text-slate-900 dark:text-gray-100">Plan v{plan.version}</span>
            <Badge
              variant="outline"
              className={cn(
                approved
                  ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-300"
                  : "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-300",
              )}
            >
              {approved ? <CheckCircle2 /> : <Circle className="fill-current" />}
              {approved ? "Approved" : "Approval needed"}
            </Badge>
          </div>
          <p className="mt-1 text-xs leading-5 text-slate-600 dark:text-gray-400">{plan.summary}</p>
        </div>
        <span className="shrink-0 text-xs text-slate-500 dark:text-gray-400">
          {plan.changesFromPrevious.length} changes from v{plan.version - 1}
        </span>
      </div>

      <ol className="divide-y divide-slate-200 dark:divide-gray-700">
        {plan.steps.map((step, index) => (
          <PlanStep key={step.id} step={step} index={index} />
        ))}
      </ol>

      {!approved ? (
        <div className="flex flex-col-reverse gap-2 border-t border-slate-200 bg-white px-4 py-3 sm:flex-row sm:items-center sm:justify-end dark:border-gray-700 dark:bg-gray-900">
          <Button type="button" size="sm" variant="outline" onClick={onRequestChanges}>
            <MessageSquareMore />
            Request changes
          </Button>
          <Button type="button" size="sm" onClick={onApprove}>
            <Check />
            Approve updated plan
          </Button>
        </div>
      ) : (
        <div className="flex items-center gap-2 border-t border-emerald-200 bg-emerald-50 px-4 py-3 text-xs text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300">
          <Play className="size-3.5" />
          Builder started from this approved plan.
        </div>
      )}
    </section>
  );
}

function PlanStep({ step, index }: { step: WorkPlanStep; index: number }) {
  return (
    <li className="grid grid-cols-[1.75rem_minmax(0,1fr)] items-start gap-2 px-4 py-3 sm:grid-cols-[1.75rem_minmax(0,1fr)_auto]">
      <span
        className={cn(
          "flex size-5 items-center justify-center rounded-full border text-[10px] font-semibold",
          planStepClassName(step),
        )}
      >
        {step.state === "complete" ? <Check className="size-3" /> : index + 1}
      </span>
      <div className="min-w-0">
        <p className="text-xs font-medium text-slate-800 dark:text-gray-200">{step.title}</p>
        <p className="mt-0.5 text-xs leading-5 text-slate-500 dark:text-gray-400">{step.description}</p>
      </div>
      {step.state === "changed" ? (
        <Badge
          variant="outline"
          className="col-start-2 w-fit border-sky-200 bg-sky-50 font-normal text-sky-700 sm:col-start-3 sm:row-start-1 dark:border-sky-800 dark:bg-sky-950/50 dark:text-sky-300"
        >
          <PencilLine />
          Revised
        </Badge>
      ) : null}
    </li>
  );
}

function eventMarkerClassName(kind: TimelineEventKind) {
  if (kind === "user") {
    return "border-slate-500 text-slate-700 dark:border-gray-400 dark:text-gray-200";
  }
  if (kind === "approval") {
    return "border-amber-400 text-amber-700 dark:border-amber-500 dark:text-amber-300";
  }
  if (kind === "progress") {
    return "border-emerald-500 text-emerald-700 dark:border-emerald-500 dark:text-emerald-300";
  }
  if (kind === "agent") {
    return "border-sky-400 text-sky-700 dark:border-sky-500 dark:text-sky-300";
  }
  return "border-slate-300 text-slate-500 dark:border-gray-600 dark:text-gray-400";
}

function planStepClassName(step: WorkPlanStep) {
  if (step.state === "complete") {
    return "border-emerald-500 bg-emerald-500 text-white";
  }
  if (step.state === "changed") {
    return "border-sky-400 bg-sky-50 text-sky-700 dark:bg-sky-950 dark:text-sky-300";
  }
  return "border-slate-300 bg-white text-slate-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400";
}
