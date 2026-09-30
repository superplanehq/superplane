import { Badge } from "@/components/reui/badge";

import type { SplitRunPhase } from "../splitRunMocks";
import type { AutomationStage } from "./automationsViewModel";
import { outputCountLabel, stepOutputSummary } from "./consoleCardText";
import { consolePages } from "./consolePages";

/**
 * Collapsed-card counts. Same three facts as the open-card tabs:
 * agent runs, artifacts, and checks. Badges only — no hover card.
 */
export function StepOutputCounts({
  stage,
  phase,
  runs = [stage],
}: {
  stage: AutomationStage;
  phase?: SplitRunPhase;
  runs?: AutomationStage[];
}) {
  const runCount = consolePages(stage, phase, runs).includes("agent") ? runs.length : 0;
  const summary = stepOutputSummary(stage, runCount);
  if (summary.runCount === 0 && summary.artifactCount === 0 && summary.checkCount === 0) {
    return null;
  }
  return (
    <div className="flex min-w-0 items-center gap-1.5">
      {summary.runCount > 0 ? (
        <CountBadge label={outputCountLabel(summary.runCount, "agent run", "agent runs")} />
      ) : null}
      {summary.artifactCount > 0 ? (
        <CountBadge label={outputCountLabel(summary.artifactCount, "artifact", "artifacts")} />
      ) : null}
      {summary.checkCount > 0 ? <CountBadge label={outputCountLabel(summary.checkCount, "check", "checks")} /> : null}
    </div>
  );
}

function CountBadge({ label }: { label: string }) {
  return (
    <Badge variant="outline" className="h-5 px-1.5 text-[12px] font-normal">
      {label}
    </Badge>
  );
}
