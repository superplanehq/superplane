import { cn } from "@/lib/utils";

import type { AgentActivity } from "../agentActivity";
import { AgentActivityView } from "../AgentActivityView";
import { AgentLiveStatus } from "../IntentAnalysisLiveWork";
import { JumpToLatestPill } from "../JumpToLatestPill";
import type { useFollowLogScroll } from "../useFollowLogScroll";
import type { AgentStep } from "./automationsViewModel";

export function StepMarkerDetail({
  step,
  activity,
  running,
  liveActive,
  liveActivity,
  liveTestId,
  defaultOpen,
  follow,
}: {
  step: AgentStep;
  activity?: AgentActivity;
  running: boolean;
  liveActive: boolean;
  liveActivity?: AgentActivity;
  liveTestId?: string;
  defaultOpen: boolean;
  follow: ReturnType<typeof useFollowLogScroll<HTMLDivElement>>;
}) {
  if (!running) {
    return (
      <div className="min-w-0 py-0.5">
        <StepActivity activity={activity} live={false} />
      </div>
    );
  }
  return (
    <div className="relative min-w-0" data-testid={liveActive ? liveTestId : undefined}>
      <div
        ref={follow.scrollRef}
        onScroll={follow.onScroll}
        className={cn("min-w-0 overflow-y-auto py-0.5", defaultOpen ? "max-h-[60vh]" : "max-h-80")}
        data-testid={`redesign-step-log-${step.id}`}
      >
        <StepActivity activity={activity} live={liveActive} />
      </div>
      {liveActive ? (
        <AgentLiveStatus
          active
          activity={liveActivity ?? activity}
          startingLabel="Starting agent…"
          collapseReasoning={false}
        />
      ) : null}
      {follow.showJumpToLatest ? (
        <JumpToLatestPill
          onJumpToLatest={() => follow.setFollowing(true)}
          testId={`redesign-step-older-${step.id}`}
        />
      ) : null}
    </div>
  );
}

function StepActivity({ activity, live }: { activity?: AgentActivity; live: boolean }) {
  if (!activity) {
    return null;
  }
  return (
    <AgentActivityView
      activity={activity}
      live={live}
      collapseCompleted={false}
      collapseReasoning={false}
      expandableCommands
      tone="log"
    />
  );
}
