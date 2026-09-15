import { useEffect, useState } from "react";

import type { CreateWithAgentMachineStatus } from "../createWithAgentTypes";
import { AgentActivityView } from "./AgentActivityView";
import { currentLiveActivity, type AgentActivity } from "./agentActivity";
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
  const status = liveStatus(activity?.items.length ?? 0, elapsedMs, stream.hasConnectedOnce ? stream.error : undefined);

  if (!active) return null;
  return (
    <div data-testid="split-run-intent-live-work">
      {activity ? <AgentActivityView activity={activity} live /> : null}
      {status ? (
        <p
          role="status"
          aria-live="polite"
          className="px-2 py-1.5 text-[13px] leading-5 text-muted-foreground"
          data-testid="split-run-intent-thinking"
        >
          <span className="sp-ai-thinking" data-text={status}>
            {status}
          </span>
        </p>
      ) : null}
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

function liveStatus(itemCount: number, elapsedMs: number, error?: string): string | undefined {
  if (error) return "Live activity disconnected. Reconnecting…";
  if (elapsedMs >= STALE_ACTIVITY_MS) {
    return `Still working · ${Math.floor(elapsedMs / 1000)}s`;
  }
  if (itemCount === 0) return "Starting analysis…";
  return undefined;
}
