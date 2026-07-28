import { AlertTriangle, ChevronLeft } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";

import {
  factoryPanelClassName,
  mutedTextClassName,
  sectionTitleClassName,
  workOrderPageWidthClassName,
} from "./factoryStyles";
import type { WorkOrderPageData } from "./factoryTypes";
import { WorkOrderChronology } from "./WorkOrderChronology";
import { PullRequestLink, WorkOrderStateBadge } from "./WorkOrderListItem";

interface WorkOrderPageProps {
  data: WorkOrderPageData;
  onBackToFactory: () => void;
  /** PRD: approval appends an event and moves `draft` → `ready`. */
  onApprove: () => void;
  /** Steering instructions are durable Work Order Events, not chat. */
  onSteer: (body: string) => void;
  onRetry: () => void;
}

/**
 * Dedicated page for one Work Order.
 *
 * PRD: two primary parts — the work description, which stays visible as the
 * context for the implementation, then the chronology. The chronology "is the
 * main structural model of the page", so everything else stays deliberately
 * light: no side panel competing with the rail, and a narrower reading column
 * than the Factory page.
 */
export function WorkOrderPage({ data, onBackToFactory, onApprove, onSteer, onRetry }: WorkOrderPageProps) {
  const { factory, workOrder, events } = data;
  const isDraft = workOrder.state === "draft";
  const isFinished = workOrder.state === "successful" || workOrder.state === "unsuccessful";

  return (
    <div className={cn(workOrderPageWidthClassName, "py-8")}>
      <button
        type="button"
        onClick={onBackToFactory}
        className={cn("inline-flex items-center gap-1 text-xs underline-offset-4 hover:underline", mutedTextClassName)}
      >
        <ChevronLeft className="size-3.5" aria-hidden />
        {factory.name}
      </button>

      {/* 1. Work description — the requested outcome, kept as context. */}
      <header className="mt-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <h1 className="text-2xl font-medium text-slate-900 dark:text-gray-100">{workOrder.title}</h1>
          <WorkOrderStateBadge state={workOrder.state} />
        </div>
        <p className="mt-3 whitespace-pre-wrap text-sm leading-relaxed text-slate-900 dark:text-gray-100">
          {workOrder.description}
        </p>
        <div className={cn("mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs", mutedTextClassName)}>
          <span>Created {formatTimeAgo(new Date(workOrder.createdAt))}</span>
          {workOrder.currentAutomation && <span>{workOrder.currentAutomation}</span>}
          {workOrder.pullRequests.map((pullRequest) => (
            <PullRequestLink key={pullRequest.url} pullRequest={pullRequest} />
          ))}
        </div>
      </header>

      {workOrder.attention && (
        <div
          className={cn(
            "mt-5 flex flex-wrap items-center justify-between gap-3 rounded-lg px-4 py-3",
            "bg-amber-50 outline outline-amber-200 dark:bg-amber-950/40 dark:outline-amber-900/60",
          )}
        >
          <p className="flex items-center gap-2 text-sm text-amber-900 dark:text-amber-200">
            <AlertTriangle className="size-4 shrink-0" aria-hidden />
            {workOrder.attention.reason}
          </p>
          {isDraft && (
            <Button type="button" size="sm" onClick={onApprove}>
              Approve
            </Button>
          )}
        </div>
      )}

      {/* 2. Chronology — the main structural model of the page. */}
      <section className={cn("mt-6", factoryPanelClassName)}>
        <h2 className={cn(sectionTitleClassName, "mb-4")}>History</h2>
        <WorkOrderChronology events={events} />
      </section>

      {isFinished ? (
        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <p className={cn("text-xs", mutedTextClassName)}>
            History is append-only — a retry starts a new attempt and keeps everything above.
          </p>
          <Button type="button" size="sm" variant="outline" onClick={onRetry}>
            Retry Work Order
          </Button>
        </div>
      ) : (
        <SteeringComposer onSteer={onSteer} />
      )}
    </div>
  );
}

/** Steering is recorded as a durable event, so it reads as part of the record. */
function SteeringComposer({ onSteer }: { onSteer: (body: string) => void }) {
  const [draft, setDraft] = useState("");

  return (
    <div className="mt-4">
      <Textarea
        rows={3}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        placeholder="Add context, answer a question, or redirect the work…"
        aria-label="Add a steering instruction"
      />
      <div className="mt-2 flex items-center justify-between gap-3">
        <p className={cn("text-xs", mutedTextClassName)}>Appended to the history as a steering event.</p>
        <Button
          type="button"
          size="sm"
          disabled={draft.trim().length === 0}
          onClick={() => {
            onSteer(draft.trim());
            setDraft("");
          }}
        >
          Send
        </Button>
      </div>
    </div>
  );
}
