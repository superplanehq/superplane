import type { FactoriesWorkOrderExecution } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatDuration } from "@/lib/duration";
import { cn } from "@/lib/utils";
import { ChevronDown, ExternalLink, Sparkles } from "lucide-react";
import { useState } from "react";

import { formatWorkOrderDateTime } from "../lib/workOrderDateTime";
import {
  formatStepExecutionDuration,
  type WorkOrderTimelineEvent,
  type WorkOrderTimelineStep,
} from "../lib/workOrderTimelineEvents";
import { formatWorkOrderExecutionUsage } from "../lib/workOrderUsage";
import { getWorkOrderExecutionRunHref } from "../lib/workOrderExecutions";
import { WorkOrderArtifactInline } from "../WorkOrderArtifactInline";
import { TimelineMarker } from "./TimelineMarker";
import { timelineActorClassName, timelineParagraphClassName, timelineTimeClassName } from "./timelineStyles";

interface DispatchTimelineItemProps {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryId: string;
  orderId?: string;
  isLatestDispatch: boolean;
}

export function DispatchTimelineItem({
  event,
  organizationId,
  factoryId,
  orderId,
  isLatestDispatch,
}: DispatchTimelineItemProps) {
  const [isExpanded, setIsExpanded] = useState(isLatestDispatch);
  const steps = event.steps ?? [];
  const latestStep = steps.at(-1);
  const duration = formatOverallDuration(steps);
  const usage = formatWorkOrderExecutionUsage(steps.map((step) => step.execution));
  const status = latestStep ? executionStatus(latestStep.execution) : null;
  const contentId = `line-run-${event.id}`;

  return (
    <li id={event.lineId ? `activity-line-run-${event.lineId}` : undefined} className="scroll-mt-20">
      <div className="flex gap-3">
        <TimelineMarker icon={Sparkles} />
        <div className="min-w-0 flex-1">
          <Button
            type="button"
            variant="ghost"
            aria-expanded={isExpanded}
            aria-controls={contentId}
            onClick={() => setIsExpanded((value) => !value)}
            className="h-auto w-full min-w-0 justify-start gap-2 whitespace-normal p-0 text-left hover:bg-transparent"
          >
            <p className={cn(timelineParagraphClassName, "min-w-0 flex-1 pt-0.5")}>
              <span className={timelineActorClassName}>{event.lineName ?? "Factory line"}</span>
              {status ? (
                <span className={cn("ml-1.5 font-medium", status.textClassName)}>{status.runLabel}</span>
              ) : null}
              {isLatestDispatch ? (
                <Badge
                  variant="outline"
                  className="ml-1.5 inline-flex h-auto rounded-md px-1.5 py-0 align-middle text-[10px] font-medium text-muted-foreground"
                >
                  Current
                </Badge>
              ) : null}
              <span className={timelineTimeClassName}>
                {" · "}Sent {formatWorkOrderDateTime(new Date(event.at))}
                {duration ? ` · ${duration}` : ""}
                {usage ? ` · ${usage}` : ""}
              </span>
            </p>
            <ChevronDown
              className={cn(
                "mt-0.5 size-3.5 shrink-0 text-muted-foreground transition-transform",
                isExpanded && "rotate-180",
              )}
              aria-hidden
            />
          </Button>

          {isExpanded ? (
            <ul id={contentId} className="mt-1 space-y-0.5">
              {steps.map((step) => (
                <DispatchStepRow
                  key={step.id}
                  organizationId={organizationId}
                  factoryId={factoryId}
                  orderId={orderId}
                  step={step}
                />
              ))}
            </ul>
          ) : null}
        </div>
      </div>
    </li>
  );
}

function DispatchStepRow({
  organizationId,
  factoryId,
  orderId,
  step,
}: {
  organizationId: string;
  factoryId: string;
  orderId?: string;
  step: WorkOrderTimelineStep;
}) {
  const runHref = getWorkOrderExecutionRunHref(organizationId, factoryId, step.execution, { orderId });
  const duration = formatStepExecutionDuration(step);
  const status = executionStatus(step.execution);

  return (
    <li className="min-w-0 py-1.5">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <StepName name={step.stepName} runHref={runHref} />
        {status ? (
          <span className={cn("inline-flex items-center gap-1.5 text-[12px]", status.textClassName)}>
            <span className={cn("size-1.5 shrink-0 rounded-full", status.dotClassName)} aria-hidden />
            {status.stepLabel}
          </span>
        ) : null}
        {duration ? <span className="text-[12px] tabular-nums text-muted-foreground">{duration}</span> : null}
        {step.artifacts?.map((artifact, index) => (
          <span className="inline-flex min-w-0 max-w-full" key={artifact.id ?? `${artifact.type}-${index}`}>
            <WorkOrderArtifactInline artifact={artifact} />
          </span>
        ))}
      </div>
      {step.comments?.map((comment, index) => (
        <StepDetail
          key={`${comment.label ?? "comment"}-${index}`}
          label={comment.label ?? "Comment"}
          body={comment.body}
        />
      ))}
    </li>
  );
}

function StepName({ name, runHref }: { name: string; runHref: string | null }) {
  const className = "inline-flex items-center gap-1 text-[13px] font-medium tracking-[-0.01em] text-foreground";
  if (!runHref) {
    return <span className={className}>{name}</span>;
  }

  return (
    <a href={runHref} target="_blank" rel="noopener noreferrer" className={cn(className, "hover:underline")}>
      {name}
      <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden />
    </a>
  );
}

function StepDetail({ label, body }: { label: string; body: string }) {
  return (
    <p className="mt-1 text-[12px] leading-relaxed text-muted-foreground">
      <span className="font-medium text-foreground/80">{label}</span>: {body}
    </p>
  );
}

interface ExecutionStatus {
  stepLabel: string;
  runLabel: string;
  textClassName: string;
  dotClassName: string;
}

const STATUS_TONE_CLASSNAMES = {
  success: {
    text: "text-[color:var(--status-success)]",
    dot: "bg-[var(--status-success-dot)]",
  },
  danger: {
    text: "text-[color:var(--status-danger)]",
    dot: "bg-[var(--status-danger-dot)]",
  },
  running: {
    text: "text-[color:var(--status-running)]",
    dot: "bg-[var(--status-running-dot)]",
  },
} as const;

function executionStatus(execution: FactoriesWorkOrderExecution): ExecutionStatus | null {
  switch (execution.result) {
    case "RESULT_PASSED":
      return status("Completed", "Complete", "success");
    case "RESULT_FAILED":
      return status("Failed", "Failed", "danger");
    case "RESULT_CANCELLED":
      return status("Cancelled", "Cancelled", "muted");
  }

  const inFlight = ["STATE_PENDING", "STATE_STARTED", "STATE_CANCELLING"].includes(execution.state ?? "");
  return inFlight ? status("Running", "Running", "running") : null;
}

function status(
  stepLabel: string,
  runLabel: string,
  tone: "success" | "danger" | "running" | "muted",
): ExecutionStatus {
  if (tone === "muted") {
    return {
      stepLabel,
      runLabel,
      textClassName: "text-muted-foreground",
      dotClassName: "bg-muted-foreground/60",
    };
  }

  const classNames = STATUS_TONE_CLASSNAMES[tone];
  return {
    stepLabel,
    runLabel,
    textClassName: classNames.text,
    dotClassName: classNames.dot,
  };
}

function formatOverallDuration(steps: WorkOrderTimelineStep[]): string | null {
  const firstStep = steps[0];
  const lastStep = steps.at(-1);
  if (!firstStep || !lastStep) {
    return null;
  }

  const durationMs = Date.parse(lastStep.finishedAt ?? lastStep.startedAt) - Date.parse(firstStep.startedAt);
  return formatDuration(durationMs, { precision: "second" }) || null;
}
