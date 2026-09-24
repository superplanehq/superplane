import { Frame, FramePanel } from "@/components/reui/frame";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import type { AutomationOutcome } from "./automationsViewModel";
import { StageStatusGlyph } from "./redesignShared";

/**
 * Outcome at a glance (stats-7 pattern on a ReUI Frame): one dominant value
 * per cell, label beneath. Answers "where is this task" before any detail.
 */
export function OutcomeStrip({ outcome, className }: { outcome: AutomationOutcome; className?: string }) {
  const pullRequest = outcome.pullRequests[0];
  return (
    <Frame variant="default" spacing="sm" dense className={cn("[--frame-radius:var(--radius-lg)]", className)}>
      <FramePanel className="grid grid-cols-2 divide-border/70 p-0 sm:grid-cols-5 sm:divide-x">
        <OutcomeCell label={outcome.startedLabel}>
          <span className="inline-flex items-center gap-1.5">
            <StageStatusGlyph status={outcome.status} />
            {outcome.statusLabel}
          </span>
        </OutcomeCell>
        <OutcomeCell label="Duration">{outcome.duration}</OutcomeCell>
        <OutcomeCell label={outcome.tokens}>{outcome.spend}</OutcomeCell>
        <OutcomeCell label={outcome.models.length === 1 ? "Model" : "Models"}>
          <span className="truncate text-[13px] font-medium" title={outcome.models.join(", ")}>
            {outcome.models.join(" · ") || "—"}
          </span>
        </OutcomeCell>
        <OutcomeCell
          label={
            outcome.checksTotal === 0 ? "No checks" : `${outcome.checksPassed}/${outcome.checksTotal} checks passed`
          }
        >
          {pullRequest ? (
            <WorkOrderPullRequestInline pullRequest={pullRequest} className="text-[14px]" />
          ) : (
            "No pull request"
          )}
        </OutcomeCell>
      </FramePanel>
    </Frame>
  );
}

function OutcomeCell({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5 px-3.5 py-2.5">
      <div className="min-w-0 truncate text-[14px] font-semibold tracking-[-0.01em] text-foreground">{children}</div>
      <div className="truncate text-[12px] text-muted-foreground">{label}</div>
    </div>
  );
}
