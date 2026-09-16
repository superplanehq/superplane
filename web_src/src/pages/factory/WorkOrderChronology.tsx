import {
  ArrowRightLeft,
  CheckCircle2,
  CircleDot,
  FilePlus2,
  GitPullRequest,
  Hammer,
  HelpCircle,
  MessageSquare,
  Navigation,
  RotateCcw,
  ThumbsUp,
  XCircle,
} from "lucide-react";

import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";

import { factoryCardClassName, mutedTextClassName } from "./factoryStyles";
import type { WorkOrderEvent, WorkOrderEventKind } from "./factoryTypes";
import { PullRequestLink } from "./WorkOrderListItem";

const EVENT_META = {
  created: { icon: FilePlus2, label: "Created" },
  approved: { icon: ThumbsUp, label: "Approved" },
  pickup: { icon: CircleDot, label: "Picked up" },
  progress: { icon: Hammer, label: "Progress" },
  conversation: { icon: MessageSquare, label: "Conversation" },
  decision: { icon: CheckCircle2, label: "Decision" },
  "approval-request": { icon: HelpCircle, label: "Approval requested" },
  steering: { icon: Navigation, label: "Steering" },
  handoff: { icon: ArrowRightLeft, label: "Handoff" },
  "pull-request": { icon: GitPullRequest, label: "Pull request" },
  outcome: { icon: CheckCircle2, label: "Outcome" },
  retry: { icon: RotateCcw, label: "Retry" },
} as const satisfies Record<WorkOrderEventKind, { icon: typeof FilePlus2; label: string }>;

/** Events that open a new attempt get a visible seam in the rail. */
const STARTS_ATTEMPT = new Set<WorkOrderEventKind>(["retry"]);

interface WorkOrderChronologyProps {
  events: WorkOrderEvent[];
}

/**
 * PRD: "The page uses a vertical timeline. The oldest event appears at the top
 * and new events are appended at the bottom." Conversations, decisions,
 * approvals and steering sit in the same rail as Automation events — the point
 * of an append-only record is that human input and machine work share one
 * chronology rather than living in a side panel.
 */
export function WorkOrderChronology({ events }: WorkOrderChronologyProps) {
  return (
    <ol className="relative">
      {events.map((event, index) => (
        <ChronologyRow
          key={event.id}
          event={event}
          isLast={index === events.length - 1}
          startsAttempt={STARTS_ATTEMPT.has(event.kind)}
        />
      ))}
    </ol>
  );
}

function ChronologyRow({
  event,
  isLast,
  startsAttempt,
}: {
  event: WorkOrderEvent;
  isLast: boolean;
  startsAttempt: boolean;
}) {
  const meta = EVENT_META[event.kind];
  const Icon = meta.icon;
  const blocking = event.kind === "approval-request" && event.awaitingResponse === true;
  const failed = event.kind === "outcome" && event.outcome === "unsuccessful";

  return (
    <li className={cn("relative pl-10", isLast ? "pb-0" : "pb-5")}>
      {!isLast && (
        <span aria-hidden className="absolute left-[13px] top-7 bottom-0 w-px bg-slate-200 dark:bg-gray-700" />
      )}
      <span
        aria-hidden
        className={cn(
          "absolute left-0 top-0.5 flex size-7 items-center justify-center rounded-full",
          blocking && "bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300",
          failed && "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300",
          !blocking && !failed && "bg-slate-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
        )}
      >
        <Icon className="size-3.5" />
      </span>

      {startsAttempt && (
        <p className={cn("mb-2 text-xs font-medium uppercase tracking-wide", mutedTextClassName)}>New attempt</p>
      )}

      <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
        <span className="text-sm font-medium text-slate-900 dark:text-gray-100">{event.summary}</span>
        <span className={cn("text-xs", mutedTextClassName)}>{formatTimeAgo(new Date(event.at))}</span>
      </div>

      <div className={cn("mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs", mutedTextClassName)}>
        <span>{event.actor.name}</span>
        {/* An Automation acting as itself would otherwise print its name twice. */}
        {event.automation && event.automation !== event.actor.name && <span>{event.automation}</span>}
      </div>

      <EventBody event={event} />

      {event.pullRequest && (
        <div className={cn("mt-2 text-xs", mutedTextClassName)}>
          <PullRequestLink pullRequest={event.pullRequest} />
        </div>
      )}

      <OutcomeBadge outcome={event.kind === "outcome" ? event.outcome : undefined} />
    </li>
  );
}

/** Conversation text, a decision rationale, or an error body. */
function EventBody({ event }: { event: WorkOrderEvent }) {
  if (!event.body) return null;
  const fromHuman = event.actor.kind === "human";
  return (
    <div
      className={cn(
        "mt-2 whitespace-pre-wrap px-3.5 py-2.5 text-sm text-slate-900 dark:text-gray-100",
        fromHuman ? "rounded-lg bg-slate-100 dark:bg-gray-800" : factoryCardClassName,
      )}
    >
      {event.body}
    </div>
  );
}

function OutcomeBadge({ outcome }: { outcome?: "successful" | "unsuccessful" }) {
  if (!outcome) return null;
  const successful = outcome === "successful";
  const Icon = successful ? CheckCircle2 : XCircle;
  return (
    <p
      className={cn(
        "mt-2 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium",
        successful
          ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/70 dark:text-emerald-300"
          : "bg-red-100 text-red-800 dark:bg-red-950/70 dark:text-red-300",
      )}
    >
      <Icon className="size-3.5" aria-hidden />
      Marked {outcome}
    </p>
  );
}
