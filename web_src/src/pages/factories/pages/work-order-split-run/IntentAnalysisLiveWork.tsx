import { useEffect, useState } from "react";

import { cn } from "@/lib/utils";

import type { CreateWithAgentMachineStatus } from "../createWithAgentTypes";
import { AgentActivityView } from "./AgentActivityView";
import { AnimatedThinkingState } from "./AnimatedThinkingState";
import { currentLiveActivity, type AgentActivity, type AgentActivityItem } from "./agentActivity";
import { useAgentActivityStream } from "./useAgentActivityStream";

const STALE_ACTIVITY_MS = 15_000;

type LiveStatus = {
  label: string;
  elapsedSeconds?: number;
};

export function AnalysisLiveWork({
  machineStatus,
  organizationId,
  canvasId,
  executionId,
  activities,
}: {
  machineStatus: CreateWithAgentMachineStatus;
  organizationId?: string;
  canvasId?: string;
  executionId?: string;
  activities?: AgentActivity[];
}) {
  return (
    <AgentLiveActivity
      active={machineStatus === "starting" || machineStatus === "running"}
      organizationId={organizationId}
      canvasId={canvasId}
      executionId={executionId}
      activities={activities}
      startingLabel="Starting analysis…"
      className="mt-3"
      testId="split-run-intent-live-work"
    />
  );
}

/**
 * The agent's work as it streams: the current activity's reasoning and
 * tool rows, then a shimmer line naming what the agent does right now.
 * The refinement chat and the run console share this block.
 */
export function AgentLiveActivity({
  active,
  organizationId,
  canvasId,
  executionId,
  activities,
  startingLabel,
  className,
  testId = "agent-live-activity",
}: {
  active: boolean;
  organizationId?: string;
  canvasId?: string;
  executionId?: string;
  /** Persisted activities to merge under the live stream. */
  activities?: AgentActivity[];
  /** Shimmer text before the first activity arrives. */
  startingLabel: string;
  className?: string;
  testId?: string;
}) {
  const stream = useAgentActivityStream({ organizationId, canvasId, executionId, active });
  return (
    <AgentLiveActivityFeed
      active={active}
      activity={currentLiveActivity(activities, stream.activities)}
      startingLabel={startingLabel}
      error={stream.hasConnectedOnce ? stream.error : undefined}
      className={className}
      testId={testId}
    />
  );
}

/** Live activity plus the shimmer line. Analysis and the run console share this. */
export function AgentLiveActivityFeed({
  active,
  activity,
  startingLabel,
  error,
  className,
  testId = "agent-live-activity",
  collapseReasoning = true,
}: {
  active: boolean;
  activity?: AgentActivity;
  startingLabel: string;
  error?: string;
  className?: string;
  testId?: string;
  collapseReasoning?: boolean;
}) {
  if (!active) return null;
  return (
    <div className={cn(className)} data-testid={testId}>
      {activity ? <AgentActivityView activity={activity} live collapseReasoning={collapseReasoning} /> : null}
      <AgentLiveStatus
        active
        activity={activity}
        startingLabel={startingLabel}
        error={error}
        collapseReasoning={collapseReasoning}
      />
    </div>
  );
}

/** Starting / Still working shimmer. The run console mounts this under a shared transcript. */
export function AgentLiveStatus({
  active,
  activity,
  startingLabel,
  error,
  collapseReasoning = true,
}: {
  active: boolean;
  activity?: AgentActivity;
  startingLabel: string;
  error?: string;
  /** Console hides an empty Thinking row, so the shimmer must keep that label. */
  collapseReasoning?: boolean;
}) {
  const elapsedMs = useActivityElapsed(activity?.sequence ?? 0, active);
  const status = liveStatus(activity, elapsedMs, startingLabel, error, collapseReasoning);
  if (!active || !status) return null;
  return (
    <p
      role="status"
      aria-label={status.label}
      aria-live="polite"
      className="px-3 py-0.5 text-[13px] leading-5 text-muted-foreground"
      data-testid="split-run-intent-thinking"
    >
      <AnimatedThinkingState text={status.label} />
      {status.elapsedSeconds === undefined ? null : <span aria-hidden> · {status.elapsedSeconds}s</span>}
    </p>
  );
}

function useActivityElapsed(sequence: number, active: boolean): number {
  const [elapsedMs, setElapsedMs] = useState(0);
  useEffect(() => {
    setElapsedMs(0);
    if (!active) return;
    const startedAt = Date.now();
    const timer = window.setInterval(() => setElapsedMs(Date.now() - startedAt), 1000);
    return () => window.clearInterval(timer);
  }, [active, sequence]);
  return elapsedMs;
}

function liveStatus(
  activity: AgentActivity | undefined,
  elapsedMs: number,
  startingLabel: string,
  error?: string,
  collapseReasoning = true,
): LiveStatus | undefined {
  if (error) return { label: "Live activity disconnected. Reconnecting…" };
  if (elapsedMs >= STALE_ACTIVITY_MS) {
    return { label: "Still working", elapsedSeconds: Math.floor(elapsedMs / 1000) };
  }
  if (!activity || activity.items.length === 0) return { label: startingLabel };

  const latest = lastActionItem(activity.items);
  if (!latest || latest.status !== "running") {
    return { label: "Planning next step…" };
  }
  if (latest.type === "content") {
    if (latest.kind === "reasoning") {
      if (!latest.text.trim() && !collapseReasoning) return { label: "Thinking" };
      return undefined;
    }
    if (latest.text.trim()) {
      return collapseReasoning ? undefined : { label: "Planning next step…" };
    }
    return { label: "Writing response…" };
  }
  if (latest.type === "tool") return undefined;
  return { label: "Planning next step…" };
}

function lastActionItem(items: AgentActivityItem[]): AgentActivityItem | undefined {
  for (let index = items.length - 1; index >= 0; index -= 1) {
    const item = items[index];
    if (item.type !== "notice") return item;
  }
  return undefined;
}
