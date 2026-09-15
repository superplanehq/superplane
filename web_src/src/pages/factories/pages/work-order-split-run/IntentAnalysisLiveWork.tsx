import { useEffect, useState } from "react";

import type { CreateWithAgentMachineStatus } from "../createWithAgentTypes";
import { AgentActivityView } from "./AgentActivityView";
import { AnimatedThinkingState } from "./AnimatedThinkingState";
import { currentLiveActivity, type AgentActivity, type AgentActivityItem, type AgentToolItem } from "./agentActivity";
import { activitySummaryLabel } from "./agentActivitySummary";
import { useAgentActivityStream } from "./useAgentActivityStream";

const STALE_ACTIVITY_MS = 15_000;

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
  const active = machineStatus === "starting" || machineStatus === "running";
  const stream = useAgentActivityStream({ organizationId, canvasId, executionId, active });
  const activity = currentLiveActivity(activities, stream.activities);
  const elapsedMs = useActivityElapsed(activity?.sequence ?? 0, active);
  const status = liveStatus(activity, elapsedMs, stream.hasConnectedOnce ? stream.error : undefined);

  if (!active) return null;
  return (
    <div data-testid="split-run-intent-live-work">
      {activity ? <AgentActivityView activity={activity} live /> : null}
      <p
        role="status"
        aria-label={status}
        aria-live="polite"
        className="px-2 py-1.5 text-[13px] leading-5 text-muted-foreground"
        data-testid="split-run-intent-thinking"
      >
        <AnimatedThinkingState text={status} />
      </p>
    </div>
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

function liveStatus(activity: AgentActivity | undefined, elapsedMs: number, error?: string): string {
  if (error) return "Live activity disconnected. Reconnecting…";
  if (elapsedMs >= STALE_ACTIVITY_MS) {
    return `Still working · ${Math.floor(elapsedMs / 1000)}s`;
  }
  if (!activity || activity.items.length === 0) return "Starting analysis…";

  const latestRunningItem = findLatestRunningItem(activity.items);
  if (latestRunningItem?.type === "content") {
    return latestRunningItem.kind === "reasoning" ? "Thinking…" : "Writing response…";
  }
  if (latestRunningItem?.type === "tool") {
    const runningTools = activity.items.filter(
      (item): item is AgentToolItem => item.type === "tool" && item.status === "running",
    );
    return `${activitySummaryLabel(runningTools)}…`;
  }
  return "Planning next step…";
}

function findLatestRunningItem(items: AgentActivityItem[]): AgentActivityItem | undefined {
  for (let index = items.length - 1; index >= 0; index -= 1) {
    const item = items[index];
    if (item.type !== "notice" && item.status === "running") return item;
  }
  return undefined;
}
