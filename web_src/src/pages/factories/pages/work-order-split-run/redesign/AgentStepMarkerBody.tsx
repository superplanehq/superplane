import { cn } from "@/lib/utils";

import type { AgentActivity, AgentToolItem } from "../agentActivity";
import { AgentActivityView } from "../AgentActivityView";
import { CommandLine } from "../AgentCommandLine";
import { AgentLiveStatus } from "../IntentAnalysisLiveWork";
import { JumpToLatestPill } from "../JumpToLatestPill";
import type { useFollowLogScroll } from "../useFollowLogScroll";
import type { AgentStep } from "./automationsViewModel";
import { bashStepCommand, bashTitleIsCommand } from "./bashStepLog";

const COMMAND_FAILED = "Command failed.";

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
  if (step.type === "bash") {
    return (
      <div className="min-w-0 py-0.5 pl-6">
        <BashStepBody step={step} />
      </div>
    );
  }
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
        <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId={`redesign-step-older-${step.id}`} />
      ) : null}
    </div>
  );
}

function BashStepBody({ step }: { step: AgentStep }) {
  const { script, stdout } = bashStepCommand(step);
  const failed = step.status === "failed";
  if (bashTitleIsCommand(step, script)) {
    return <BashCapturedOutput text={stdout} failed={failed} />;
  }
  return (
    <div className="flex min-w-0 flex-col gap-1">
      <CommandLine
        tool={bashTool(step, script, failed ? "" : stdout)}
        expandable
        showOutput={!failed}
        showPromptMark
        defaultOpen
      />
      {failed ? <BashCapturedOutput text={stdout} failed /> : null}
    </div>
  );
}

function BashCapturedOutput({ text, failed }: { text: string; failed: boolean }) {
  const body = text.trim() || (failed ? COMMAND_FAILED : "");
  if (!body) {
    return null;
  }
  return (
    <pre
      className={cn(
        "overflow-x-auto font-mono text-[12px] leading-5 whitespace-pre-wrap break-words [tab-size:2]",
        failed ? "text-destructive" : "text-muted-foreground",
      )}
    >
      {body}
    </pre>
  );
}

function bashTool(step: AgentStep, script: string, stdout: string): AgentToolItem {
  return {
    type: "tool",
    id: `${step.id}-command`,
    kind: "bash",
    name: "bash",
    input: script,
    output: stdout,
    outputStreams: [],
    status: "passed",
    truncated: false,
  };
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
